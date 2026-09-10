package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// PGVectorConfig 对齐 Python PGVector 构造参数 (含 config 默认值由 config 包填充)。
type PGVectorConfig struct {
	DBName             string
	CollectionName     string
	EmbeddingModelDims int
	User               string
	Password           string
	Host               string
	Port               int
	DiskANN            bool
	HNSW               bool
	MinConn            int
	MaxConn            int
	SSLMode            string
	ConnectionString   string
	Pool               *pgxpool.Pool
}

// PGVector 实现 VectorStore (pgx 连接池)。
type PGVector struct {
	cfg     PGVectorConfig
	pool    *pgxpool.Pool
	ensured bool
}

// NewPGVector 构造并打开连接池 (open=False 等价: pgxpool 惰性连接)。
func NewPGVector(cfg PGVectorConfig) (*PGVector, error) {
	dsn := cfg.ConnectionString
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	}
	if cfg.SSLMode != "" {
		dsn = withSSLMode(dsn, cfg.SSLMode)
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgvector: 解析 DSN 失败: %w", err)
	}
	if cfg.MaxConn > 0 {
		poolCfg.MaxConns = int32(cfg.MaxConn)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pgvector: 创建连接池失败: %w", err)
	}
	return &PGVector{cfg: cfg, pool: pool}, nil
}

// withSSLMode 对齐 Python _with_sslmode (URL 与 keyword 两种 conninfo)。
func withSSLMode(dsn string, sslmode string) string {
	if strings.Contains(dsn, "://") {
		if strings.Contains(dsn, "sslmode=") {
			// 简化: URL 形态追加前先剥除既有 sslmode 参数
			parts := strings.Fields(dsn)
			var kept []string
			for _, p := range parts {
				if !strings.HasPrefix(p, "sslmode=") {
					kept = append(kept, p)
				}
			}
			dsn = strings.Join(kept, " ")
		}
		return dsn + "?sslmode=" + sslmode
	}
	return dsn + " sslmode=" + sslModeKeyword(dsn, sslmode)
}

func sslModeKeyword(dsn string, sslmode string) string { return sslmode }

// ensureCollection 对齐 _ensure_collection: 首次使用时建表。
func (p *PGVector) ensureCollection() error {
	if p.ensured {
		return nil
	}
	cols, err := p.ListCols()
	if err == nil {
		for _, c := range cols {
			if c == p.cfg.CollectionName {
				p.ensured = true
				return nil
			}
		}
	}
	if err := p.CreateCol(); err != nil {
		return err
	}
	p.ensured = true
	return nil
}

// CreateCol 对齐 create_col: vector 扩展 + 表 + 可选 hnsw/diskann 索引 + 文本索引。
func (p *PGVector) CreateCol() error {
	ctx := context.Background()
	if _, err := p.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("pgvector: 建 vector 扩展失败: %w", err)
	}
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id UUID PRIMARY KEY,
		vector vector(%d),
		payload JSONB
	);`, tbl, p.cfg.EmbeddingModelDims)
	if _, err := p.pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("pgvector: 建表失败: %w", err)
	}
	if p.cfg.HNSW {
		idx := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s USING hnsw (vector vector_cosine_ops)`,
			pgx.Identifier{p.cfg.CollectionName + "_hnsw_idx"}.Sanitize(), tbl)
		if _, err := p.pool.Exec(ctx, idx); err != nil {
			return fmt.Errorf("pgvector: 建 hnsw 索引失败: %w", err)
		}
	}
	// pgvector-go 插入的是精确 cosine 距离排序所需数据; text_lemmatized GIN 索引照搬上游
	gin := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s USING gin(to_tsvector('simple', payload->>'text_lemmatized'))`,
		pgx.Identifier{p.cfg.CollectionName + "_text_lemmatized_idx"}.Sanitize(), tbl)
	if _, err := p.pool.Exec(ctx, gin); err != nil {
		return fmt.Errorf("pgvector: 建文本索引失败: %w", err)
	}
	return nil
}

// Insert 对齐 insert (executemany 等价: 批量 INSERT)。
func (p *PGVector) Insert(vectors [][]float64, ids []string, payloads []map[string]any) error {
	if err := p.ensureCollection(); err != nil {
		return err
	}
	ctx := context.Background()
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	sqlText := fmt.Sprintf("INSERT INTO %s (id, vector, payload) VALUES ($1, $2, $3)", tbl)
	batch := &pgx.Batch{}
	for i := range vectors {
		raw, err := json.Marshal(payloads[i])
		if err != nil {
			return fmt.Errorf("pgvector: payload 序列化失败: %w", err)
		}
		batch.Queue(sqlText, ids[i], pgvector.NewVector(toFloat32(vectors[i])), string(raw))
	}
	if err := p.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("pgvector: 插入失败: %w", err)
	}
	return nil
}

// Search 对齐 search: 余弦距离 <=>, score=max(0,1-d)。
func (p *PGVector) Search(query string, vectors []float64, topK int, filters map[string]any) ([]OutputData, error) {
	if err := p.ensureCollection(); err != nil {
		return nil, err
	}
	where, args, err := p.buildWhere(filters, 2)
	if err != nil {
		return nil, err
	}
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	sqlText := fmt.Sprintf(`SELECT id, vector <=> $1::vector AS distance, payload FROM %s %s ORDER BY distance LIMIT $%d`, tbl, where, len(args)+2)
	args = append(args, topK)
	rows, err := p.pool.Query(context.Background(), sqlText, append([]any{pgvector.NewVector(toFloat32(vectors))}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("pgvector: 检索失败: %w", err)
	}
	defer rows.Close()
	var out []OutputData
	for rows.Next() {
		var id string
		var dist float64
		var payloadRaw string
		if err := rows.Scan(&id, &dist, &payloadRaw); err != nil {
			return nil, fmt.Errorf("pgvector: 扫描失败: %w", err)
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(payloadRaw), &payload)
		score := 1.0 - dist
		if score < 0 {
			score = 0
		}
		s := score
		out = append(out, OutputData{ID: id, Score: &s, Payload: payload})
	}
	return out, rows.Err()
}

// buildWhere 组装 WHERE 子句并返回从 $start 开始的编号占位符参数。
func (p *PGVector) buildWhere(filters map[string]any, start int) (string, []any, error) {
	conds, err := buildFilterConditions(filters)
	if err != nil {
		return "", nil, err
	}
	if len(conds) == 0 {
		return "", nil, nil
	}
	var args []any
	var parts []string
	next := start
	for _, c := range conds {
		sqlText := c.sql
		// 模板里的每个 %s 依序替换为 $n (key 先于 value)
		for strings.Contains(sqlText, "%s") {
			sqlText = strings.Replace(sqlText, "%s", fmt.Sprintf("$%d", next), 1)
			next++
		}
		args = append(args, c.params...)
		parts = append(parts, "("+sqlText+")")
	}
	return "WHERE " + strings.Join(parts, " AND "), args, nil
}

// KeywordSearch 对齐 keyword_search: simple 配置的 ts_rank_cd 排序; 失败返回 nil (无错)。
func (p *PGVector) KeywordSearch(query string, topK int, filters map[string]any) ([]OutputData, error) {
	if err := p.ensureCollection(); err != nil {
		return nil, err
	}
	where, args, err := p.buildWhere(filters, 2)
	if err != nil {
		return nil, err
	}
	// keyword 模板自带 WHERE (tsvector @@), 过滤条件须以 AND 接续 (对齐 Python filter_clause)
	andClause := where
	if andClause != "" {
		andClause = "AND " + strings.TrimPrefix(andClause, "WHERE ")
	}
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	sqlText := fmt.Sprintf(`SELECT id, ts_rank_cd(to_tsvector('simple', payload->>'text_lemmatized'), plainto_tsquery('simple', $1)) AS score, payload
		FROM %s WHERE to_tsvector('simple', payload->>'text_lemmatized') @@ plainto_tsquery('simple', $1) %s
		ORDER BY score DESC LIMIT $%d`, tbl, andClause, len(args)+2) // +1: query 的 $1; +1: limit 自身
	args = append(args, topK)
	rows, err := p.pool.Query(context.Background(), sqlText, append([]any{query}, args...)...)
	if err != nil {
		return nil, nil // 对齐 Python: keyword 失败静默降级
	}
	defer rows.Close()
	var out []OutputData
	for rows.Next() {
		var id string
		var score float64
		var payloadRaw string
		if err := rows.Scan(&id, &score, &payloadRaw); err != nil {
			return nil, nil
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(payloadRaw), &payload)
		s := score
		out = append(out, OutputData{ID: id, Score: &s, Payload: payload})
	}
	return out, nil
}

// Get 对齐 get: 不存在返回 nil。
func (p *PGVector) Get(vectorID string) (*OutputData, error) {
	if err := p.ensureCollection(); err != nil {
		return nil, err
	}
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	var id, payloadRaw string
	err := p.pool.QueryRow(context.Background(),
		fmt.Sprintf("SELECT id, payload FROM %s WHERE id = $1", tbl), vectorID).Scan(&id, &payloadRaw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("pgvector: 查询失败: %w", err)
	}
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(payloadRaw), &payload)
	return &OutputData{ID: id, Score: nil, Payload: payload}, nil
}

// Delete 对齐 delete。
func (p *PGVector) Delete(vectorID string) error {
	if err := p.ensureCollection(); err != nil {
		return err
	}
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	if _, err := p.pool.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s WHERE id = $1", tbl), vectorID); err != nil {
		return fmt.Errorf("pgvector: 删除失败: %w", err)
	}
	return nil
}

// Update 对齐 update: vector 与 payload 各自可空。
func (p *PGVector) Update(vectorID string, vector []float64, payload map[string]any) error {
	if err := p.ensureCollection(); err != nil {
		return err
	}
	ctx := context.Background()
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	if vector != nil {
		if _, err := p.pool.Exec(ctx, fmt.Sprintf("UPDATE %s SET vector = $1 WHERE id = $2", tbl),
			pgvector.NewVector(toFloat32(vector)), vectorID); err != nil {
			return fmt.Errorf("pgvector: 更新向量失败: %w", err)
		}
	}
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("pgvector: payload 序列化失败: %w", err)
		}
		if _, err := p.pool.Exec(ctx, fmt.Sprintf("UPDATE %s SET payload = $1 WHERE id = $2", tbl), string(raw), vectorID); err != nil {
			return fmt.Errorf("pgvector: 更新 payload 失败: %w", err)
		}
	}
	return nil
}

// List 对齐 list: 返回嵌套一层 ([[OutputData]]) 形状。
func (p *PGVector) List(filters map[string]any, topK int) ([][]OutputData, error) {
	if err := p.ensureCollection(); err != nil {
		return nil, err
	}
	where, args, err := p.buildWhere(filters, 1)
	if err != nil {
		return nil, err
	}
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	sqlText := fmt.Sprintf("SELECT id, payload FROM %s %s LIMIT $%d", tbl, where, len(args)+1)
	args = append(args, topK)
	rows, err := p.pool.Query(context.Background(), sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("pgvector: 列出失败: %w", err)
	}
	defer rows.Close()
	var out []OutputData
	for rows.Next() {
		var id, payloadRaw string
		if err := rows.Scan(&id, &payloadRaw); err != nil {
			return nil, fmt.Errorf("pgvector: 扫描失败: %w", err)
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(payloadRaw), &payload)
		out = append(out, OutputData{ID: id, Score: nil, Payload: payload})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return [][]OutputData{out}, nil
}

// ListCols 对齐 list_cols (public schema 表名)。
func (p *PGVector) ListCols() ([]string, error) {
	rows, err := p.pool.Query(context.Background(),
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		return nil, fmt.Errorf("pgvector: 列表失败: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// DeleteCol 对齐 delete_col。
func (p *PGVector) DeleteCol() error {
	tbl := pgx.Identifier{p.cfg.CollectionName}.Sanitize()
	if _, err := p.pool.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", tbl)); err != nil {
		return fmt.Errorf("pgvector: 删表失败: %w", err)
	}
	p.ensured = false
	return nil
}

// Reset 对齐 reset: 删表重建。
func (p *PGVector) Reset() error {
	if err := p.DeleteCol(); err != nil {
		return err
	}
	return p.CreateCol()
}

// Close 释放连接池。
func (p *PGVector) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// toFloat32 向量转换 (pgvector-go 接受 float32)。
func toFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(x)
	}
	return out
}
