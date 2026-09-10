-- +goose Up
CREATE UNIQUE INDEX ix_users_only_one_admin ON users (role) WHERE role = 'admin';
-- +goose Down
DROP INDEX ix_users_only_one_admin;
