package store

import (
	"context"
	"fmt"
	"time"
)

// memoryOpsWhere 记忆操作请求的路径过滤 (analytics 口径, 与 doc 语义一致)。
const memoryOpsWhere = `(path = '/memories' OR path LIKE '/memories/%' OR path = '/search')`

// AnalyticsSummary 记忆操作聚合 (窗口内)。
type AnalyticsSummary struct {
	TotalOps     int
	AvgLatencyMS float64
	SuccessPct   float64 // 0-100, status_code < 400 占比
}

// DailyCount 单日操作数 (UTC 日)。
type DailyCount struct {
	Date  string // YYYY-MM-DD
	Count int
}

// GetAnalyticsSummary 聚合窗口内记忆操作请求 (since 起含当天)。
func (s *Store) GetAnalyticsSummary(ctx context.Context, since time.Time) (*AnalyticsSummary, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT count(*), COALESCE(avg(latency_ms), 0),
		COALESCE(100.0 * avg(CASE WHEN status_code < 400 THEN 1.0 ELSE 0.0 END), 0)
		FROM request_logs WHERE created_at >= $1 AND `+memoryOpsWhere, since)
	var out AnalyticsSummary
	if err := row.Scan(&out.TotalOps, &out.AvgLatencyMS, &out.SuccessPct); err != nil {
		return nil, fmt.Errorf("appdb: analytics summary: %w", err)
	}
	return &out, nil
}

// ListDailyOpCounts 按UTC 日聚合窗口内记忆操作数 (仅有数据的天)。
func (s *Store) ListDailyOpCounts(ctx context.Context, since time.Time) ([]DailyCount, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, count(*)
		FROM request_logs WHERE created_at >= $1 AND `+memoryOpsWhere+`
		GROUP BY day ORDER BY day`, since)
	if err != nil {
		return nil, fmt.Errorf("appdb: analytics daily counts: %w", err)
	}
	defer rows.Close()
	var out []DailyCount
	for rows.Next() {
		var d DailyCount
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, fmt.Errorf("appdb: scan analytics daily count: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
