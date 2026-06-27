-- +goose Up
CREATE TABLE job_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL,
    offset INTEGER NOT NULL,
    results_per_page INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (job_id, offset, results_per_page),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE job_requests;
