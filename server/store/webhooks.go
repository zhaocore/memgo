package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Webhook 事件通知端点注册 (webhooks 表; 事件投递见 server/webhook 包)。
type Webhook struct {
	ID         string
	Name       string
	URL        string
	EventTypes []string
	CreatedAt  time.Time
}

// CreateWebhook 插入 (id 由调用方生成)。
func (s *Store) CreateWebhook(ctx context.Context, wh *Webhook) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO webhooks (id, name, url, event_types) VALUES ($1, $2, $3, $4)`,
		wh.ID, wh.Name, wh.URL, wh.EventTypes)
	if err != nil {
		return fmt.Errorf("appdb: insert webhook: %w", err)
	}
	return nil
}

// ListWebhooks 全量列表 (created_at desc)。
func (s *Store) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id::text, name, url, event_types, created_at FROM webhooks ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("appdb: list webhooks: %w", err)
	}
	defer rows.Close()
	var out []Webhook
	for rows.Next() {
		var wh Webhook
		if err := rows.Scan(&wh.ID, &wh.Name, &wh.URL, &wh.EventTypes, &wh.CreatedAt); err != nil {
			return nil, fmt.Errorf("appdb: scan webhook: %w", err)
		}
		out = append(out, wh)
	}
	return out, rows.Err()
}

// GetWebhook 按 id (不存在返回 nil)。
func (s *Store) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	row := s.DB.QueryRowContext(ctx,
		`SELECT id::text, name, url, event_types, created_at FROM webhooks WHERE id = $1`, id)
	var wh Webhook
	err := row.Scan(&wh.ID, &wh.Name, &wh.URL, &wh.EventTypes, &wh.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("appdb: get webhook: %w", err)
	}
	return &wh, nil
}

// UpdateWebhook 按 id 更新非空字段 (部分更新语义: nil 字段保留原值)。
func (s *Store) UpdateWebhook(ctx context.Context, wh *Webhook) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE webhooks SET name = $2, url = $3, event_types = $4 WHERE id = $1`,
		wh.ID, wh.Name, wh.URL, wh.EventTypes)
	if err != nil {
		return fmt.Errorf("appdb: update webhook: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("Webhook with id %s not found", wh.ID)
	}
	return nil
}

// DeleteWebhook 硬删 (不存在返回 not found 错误)。
func (s *Store) DeleteWebhook(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("appdb: delete webhook: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("Webhook with id %s not found", id)
	}
	return nil
}
