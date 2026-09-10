package store

import (
	"context"
	"fmt"
)

// InsertRequestLog 请求日志落库。
func (s *Store) InsertRequestLog(ctx context.Context, l *RequestLog) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO request_logs (id, method, path, status_code, latency_ms, auth_type) VALUES ($1, $2, $3, $4, $5, $6)`,
		l.ID, l.Method, l.Path, l.StatusCode, l.LatencyMS, l.AuthType)
	if err != nil {
		return fmt.Errorf("appdb: insert request log: %w", err)
	}
	return nil
}

// ListRequestLogs 对齐 GET /requests (仅 api_key 类, created_at desc, limit)。
func (s *Store) ListRequestLogs(ctx context.Context, limit int) ([]RequestLog, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id::text, method, path, status_code, latency_ms, auth_type, created_at
		FROM request_logs WHERE auth_type IN ('api_key', 'admin_api_key')
		ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("appdb: list request logs: %w", err)
	}
	defer rows.Close()
	var out []RequestLog
	for rows.Next() {
		var l RequestLog
		if err := rows.Scan(&l.ID, &l.Method, &l.Path, &l.StatusCode, &l.LatencyMS, &l.AuthType, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("appdb: scan request log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
