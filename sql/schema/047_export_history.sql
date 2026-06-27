-- +goose Up
CREATE TABLE export_history (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    filters TEXT,
    job_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deduped BOOLEAN NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE export_history;
