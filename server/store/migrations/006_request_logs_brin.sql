-- +goose Up
DROP INDEX ix_request_logs_created_at;
CREATE INDEX ix_request_logs_created_at ON request_logs USING BRIN (created_at);
-- +goose Down
DROP INDEX ix_request_logs_created_at;
CREATE INDEX ix_request_logs_created_at ON request_logs (created_at);
