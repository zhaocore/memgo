package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// InsertRefreshJTI 落 jti 行。
func (s *Store) InsertRefreshJTI(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO refresh_token_jtis (jti, user_id, expires_at) VALUES ($1, $2, $3)`, jti, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("appdb: insert jti: %w", err)
	}
	return nil
}

// ConsumeRefreshJTI 对齐 consume_refresh_jti: 条件 UPDATE CAS 防并发重放 (T11)。
// 返回 false = 未命中 (缺失/已用/过期)。
func (s *Store) ConsumeRefreshJTI(ctx context.Context, jti string) (bool, error) {
	tag, err := s.DB.ExecContext(ctx,
		`UPDATE refresh_token_jtis SET used_at = now() WHERE jti = $1 AND used_at IS NULL AND expires_at > now()`, jti)
	if err != nil {
		return false, fmt.Errorf("appdb: consume jti: %w", err)
	}
	affected, err := tag.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("appdb: consume jti rowcount: %w", err)
	}
	return affected == 1, nil
}

// GetSetting / UpsertSetting settings 表 (config_overrides)。
func (s *Store) GetSetting(ctx context.Context, key string) (*string, error) {
	var val string
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("appdb: get setting: %w", err)
	}
	return &val, nil
}

// UpsertSetting on_conflict_do_update (对齐 _save_overrides)。
func (s *Store) UpsertSetting(ctx context.Context, key, value string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, key, value)
	if err != nil {
		return fmt.Errorf("appdb: upsert setting: %w", err)
	}
	return nil
}
