-- +goose Up
CREATE TABLE job_agents (
    job_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    scraped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (job_id, agent_id),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE job_agents;
