//go:build cgo
// +build cgo

package job

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/tursodatabase/go-libsql"
)

func TestJobStates(t *testing.T) {
	// Create an in-memory SQLite database for testing transitions
	db, err := sql.Open("libsql", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create tables in memory
	_, err = db.Exec(`
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'queued',
			max_agents_limit INTEGER NOT NULL,
			concurrency INTEGER NOT NULL,
			throttle_request_limit INTEGER NOT NULL,
			save_raw_agents BOOLEAN NOT NULL DEFAULT 0,
			db_mode TEXT NOT NULL DEFAULT 'local',
			filters TEXT,
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
	`)
	if err != nil {
		t.Fatalf("failed to create jobs table: %v", err)
	}

	jm := NewJobManager(db, "secret")

	// 1. Create a job
	j, err := jm.CreateJob("Test Job", 50, 3, 5, false, "local", nil)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if j.Status != "queued" {
		t.Errorf("expected initial status to be 'queued', got %q", j.Status)
	}

	// 2. Pause test (should error since job is not running)
	err = jm.PauseJob(j.ID)
	if err == nil {
		t.Error("expected error pausing a non-running job, got nil")
	}

	// 3. Mark completed manually
	jm.markCompleted(j.ID)
	jUpdated, _ := jm.GetJob(j.ID)
	if jUpdated.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", jUpdated.Status)
	}

	// 4. Mark failed manually
	jm.markFailed(j.ID, "some error occurred")
	jUpdated2, _ := jm.GetJob(j.ID)
	if jUpdated2.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", jUpdated2.Status)
	}
	if jUpdated2.ErrorMessage == nil || *jUpdated2.ErrorMessage != "some error occurred" {
		t.Errorf("expected error message to be set")
	}
}
