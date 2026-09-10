-- +goose Up
CREATE TABLE refresh_token_jtis (
    jti UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_refresh_token_jtis_expires_at ON refresh_token_jtis (expires_at);
-- +goose Down
DROP INDEX ix_refresh_token_jtis_expires_at;
DROP TABLE refresh_token_jtis;
