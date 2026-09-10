-- +goose Up
CREATE TABLE request_logs (
    id UUID PRIMARY KEY,
    method VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status_code INTEGER NOT NULL,
    latency_ms DOUBLE PRECISION NOT NULL,
    auth_type VARCHAR(32) NOT NULL DEFAULT 'none',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_request_logs_created_at ON request_logs (created_at);
-- +goose Down
DROP INDEX ix_request_logs_created_at;
DROP TABLE request_logs;
