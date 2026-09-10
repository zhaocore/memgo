package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CountUsers 对齐 select(func.count(User.id))。
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.DB.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil {
		return 0, fmt.Errorf("appdb: count users: %w", err)
	}
	return n, nil
}

// GetUser 按 id。
func (s *Store) GetUser(ctx context.Context, id string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id::text, name, email, password_hash, role, created_at, last_login_at FROM users WHERE id = $1`, id)
	return scanUser(row)
}

// GetUserByEmail 按 email。
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id::text, name, email, password_hash, role, created_at, last_login_at FROM users WHERE email = $1`, email)
	return scanUser(row)
}

// FirstUser 对齐 _get_default_user (created_at asc)。
func (s *Store) FirstUser(ctx context.Context) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id::text, name, email, password_hash, role, created_at, last_login_at FROM users ORDER BY created_at ASC LIMIT 1`)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.LastLoginAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("appdb: scan user: %w", err)
	}
	return &u, nil
}

// CreateUser 插入首个 admin (id 由调用方生成)。
func (s *Store) CreateUser(ctx context.Context, u *User) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO users (id, name, email, password_hash, role) VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Name, u.Email, u.PasswordHash, u.Role)
	if err != nil {
		return fmt.Errorf("appdb: insert user: %w", err)
	}
	return nil
}

// UpdateUserProfile 改 name/email (返回冲突标志)。
func (s *Store) UpdateUserProfile(ctx context.Context, id, name, email string) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM users WHERE email = $1 AND id != $2`, email, id).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("appdb: email 冲突检查: %w", err)
	}
	if n > 0 {
		return true, nil
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE users SET name = $1, email = $2 WHERE id = $3`, name, email, id)
	if err != nil {
		return false, fmt.Errorf("appdb: update user: %w", err)
	}
	return false, nil
}

// UpdatePassword 改密码哈希。
func (s *Store) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("appdb: update password: %w", err)
	}
	return nil
}

// TouchLogin 更新 last_login_at。
func (s *Store) TouchLogin(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("appdb: touch login: %w", err)
	}
	return nil
}
