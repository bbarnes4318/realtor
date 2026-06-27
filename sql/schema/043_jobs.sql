-- +goose Up
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', -- queued, running, paused, completed, failed, canceled
    max_agents_limit INTEGER NOT NULL,
    concurrency INTEGER NOT NULL,
    throttle_request_limit INTEGER NOT NULL,
    save_raw_agents BOOLEAN NOT NULL DEFAULT 0,
    db_mode TEXT NOT NULL DEFAULT 'local', -- local, turso
    filters TEXT, -- JSON string of filters
    total_estimated_requests INTEGER NOT NULL DEFAULT 0,
    completed_requests INTEGER NOT NULL DEFAULT 0,
    failed_requests INTEGER NOT NULL DEFAULT 0,
    agents_saved INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    error_message TEXT
);

-- +goose Down
DROP TABLE jobs;
