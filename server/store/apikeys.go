package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CreateAPIKey 插入 key (id 由调用方生成)。
func (s *Store) CreateAPIKey(ctx context.Context, k *APIKey) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO api_keys (id, key_prefix, key_hash, label, created_by) VALUES ($1, $2, $3, $4, $5)`,
		k.ID, k.KeyPrefix, k.KeyHash, k.Label, k.CreatedBy)
	if err != nil {
		return fmt.Errorf("appdb: insert api key: %w", err)
	}
	return nil
}

// ListAPIKeysByCreator 对齐 GET /api-keys (未撤销, created_at desc)。
func (s *Store) ListAPIKeysByCreator(ctx context.Context, creatorID string) ([]APIKey, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id::text, key_prefix, key_hash, label, created_by::text, last_used_at, revoked_at, created_at
		FROM api_keys WHERE created_by = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, creatorID)
	if err != nil {
		return nil, fmt.Errorf("appdb: list api keys: %w", err)
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.KeyPrefix, &k.KeyHash, &k.Label, &k.CreatedBy, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("appdb: scan api key: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// GetAPIKey 按 id。
func (s *Store) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id::text, key_prefix, key_hash, label, created_by::text, last_used_at, revoked_at, created_at FROM api_keys WHERE id = $1`, id)
	var k APIKey
	err := row.Scan(&k.ID, &k.KeyPrefix, &k.KeyHash, &k.Label, &k.CreatedBy, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("appdb: get api key: %w", err)
	}
	return &k, nil
}

// APIKeyCandidatesByPrefix 对齐前缀索引查候选 (未撤销)。
func (s *Store) APIKeyCandidatesByPrefix(ctx context.Context, prefix string) ([]APIKey, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id::text, key_prefix, key_hash, label, created_by::text, last_used_at, revoked_at, created_at
		FROM api_keys WHERE key_prefix = $1 AND revoked_at IS NULL`, prefix)
	if err != nil {
		return nil, fmt.Errorf("appdb: api key 候选查询: %w", err)
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.KeyPrefix, &k.KeyHash, &k.Label, &k.CreatedBy, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("appdb: scan api key: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// TouchAPIKeyLastUsed 更新 last_used_at。
func (s *Store) TouchAPIKeyLastUsed(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("appdb: touch key: %w", err)
	}
	return nil
}

// RevokeAPIKey 软删 (置 revoked_at)。
func (s *Store) RevokeAPIKey(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE api_keys SET revoked_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("appdb: revoke key: %w", err)
	}
	return nil
}
