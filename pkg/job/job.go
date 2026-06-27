package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/suffer-sami/realtor-scraper/pkg/proxy"
	"github.com/suffer-sami/realtor-scraper/pkg/scraper"
)

const (
	defaultThrottleTimeout = 5 * time.Second
	defaultResultsPerPage  = 20
)

type JobFilters struct {
	State      string `json:"state"`
	City       string `json:"city"`
	Zip        string `json:"zip"`
	Brokerage  string `json:"brokerage"`
	AgentName  string `json:"agent_name"`
	AreaServed string `json:"area_served"`
}

type Job struct {
	ID                     string      `json:"id"`
	Name                   string      `json:"name"`
	Status                 string      `json:"status"` // queued, running, paused, completed, failed, canceled
	MaxAgentsLimit         int         `json:"max_agents_limit"`
	Concurrency            int         `json:"concurrency"`
	ThrottleRequestLimit   int         `json:"throttle_request_limit"`
	SaveRawAgents          bool        `json:"save_raw_agents"`
	DBMode                 string      `json:"db_mode"`
	Filters                *JobFilters `json:"filters"`
	TotalEstimatedRequests int         `json:"total_estimated_requests"`
	CompletedRequests      int         `json:"completed_requests"`
	FailedRequests         int         `json:"failed_requests"`
	AgentsSaved            int         `json:"agents_saved"`
	StartedAt              *time.Time  `json:"started_at"`
	CompletedAt            *time.Time  `json:"completed_at"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at"`
	ErrorMessage           *string     `json:"error_message"`
}

type ActiveJob struct {
	JobID      string
	Ctx        context.Context
	Cancel     context.CancelFunc
	pauseChan  chan struct{}
	resumeChan chan struct{}
	isPaused   int32 // atomic bool
	wg         sync.WaitGroup
}

type JobManager struct {
	db         *sql.DB
	mu         sync.Mutex
	activeJobs map[string]*ActiveJob
	jwtSecret  string
	Rotator    *proxy.Rotator
}

func NewJobManager(db *sql.DB, jwtSecret string) *JobManager {
	rot, err := proxy.NewRotator(db)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize proxy rotator: %v\n", err)
	}
	return &JobManager{
		db:         db,
		activeJobs: make(map[string]*ActiveJob),
		jwtSecret:  jwtSecret,
		Rotator:    rot,
	}
}

// CreateJob inserts a new job into the database
func (jm *JobManager) CreateJob(name string, maxLimit, concurrency, throttle int, saveRaw bool, dbMode string, filters *JobFilters) (*Job, error) {
	id := uuid.New().String()
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filters: %w", err)
	}

	_, err = jm.db.Exec(`
		INSERT INTO jobs (id, name, status, max_agents_limit, concurrency, throttle_request_limit, save_raw_agents, db_mode, filters, created_at, updated_at)
		VALUES (?, ?, 'queued', ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, name, maxLimit, concurrency, throttle, saveRaw, dbMode, string(filtersJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to insert job: %w", err)
	}

	return jm.GetJob(id)
}

// GetJob retrieves a job details from database
func (jm *JobManager) GetJob(id string) (*Job, error) {
	var job Job
	var filtersStr sql.NullString
	var startedAtNull, completedAtNull sql.NullTime
	var errMsgNull sql.NullString

	err := jm.db.QueryRow(`
		SELECT id, name, status, max_agents_limit, concurrency, throttle_request_limit, save_raw_agents, db_mode, filters, total_estimated_requests, completed_requests, failed_requests, agents_saved, started_at, completed_at, created_at, updated_at, error_message
		FROM jobs WHERE id = ?
	`, id).Scan(
		&job.ID, &job.Name, &job.Status, &job.MaxAgentsLimit, &job.Concurrency, &job.ThrottleRequestLimit,
		&job.SaveRawAgents, &job.DBMode, &filtersStr, &job.TotalEstimatedRequests, &job.CompletedRequests,
		&job.FailedRequests, &job.AgentsSaved, &startedAtNull, &completedAtNull, &job.CreatedAt, &job.UpdatedAt, &errMsgNull,
	)
	if err != nil {
		return nil, err
	}

	if filtersStr.Valid && filtersStr.String != "" {
		var f JobFilters
		if err := json.Unmarshal([]byte(filtersStr.String), &f); err == nil {
			job.Filters = &f
		}
	}

	if startedAtNull.Valid {
		job.StartedAt = &startedAtNull.Time
	}
	if completedAtNull.Valid {
		job.CompletedAt = &completedAtNull.Time
	}
	if errMsgNull.Valid {
		job.ErrorMessage = &errMsgNull.String
	}

	return &job, nil
}

// ListJobs returns all jobs ordered by created_at desc
func (jm *JobManager) ListJobs() ([]*Job, error) {
	rows, err := jm.db.Query(`
		SELECT id, name, status, max_agents_limit, concurrency, throttle_request_limit, save_raw_agents, db_mode, filters, total_estimated_requests, completed_requests, failed_requests, agents_saved, started_at, completed_at, created_at, updated_at, error_message
		FROM jobs ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []*Job{}
	for rows.Next() {
		var job Job
		var filtersStr sql.NullString
		var startedAtNull, completedAtNull sql.NullTime
		var errMsgNull sql.NullString

		err := rows.Scan(
			&job.ID, &job.Name, &job.Status, &job.MaxAgentsLimit, &job.Concurrency, &job.ThrottleRequestLimit,
			&job.SaveRawAgents, &job.DBMode, &filtersStr, &job.TotalEstimatedRequests, &job.CompletedRequests,
			&job.FailedRequests, &job.AgentsSaved, &startedAtNull, &completedAtNull, &job.CreatedAt, &job.UpdatedAt, &errMsgNull,
		)
		if err != nil {
			return nil, err
		}

		if filtersStr.Valid && filtersStr.String != "" {
			var f JobFilters
			if err := json.Unmarshal([]byte(filtersStr.String), &f); err == nil {
				job.Filters = &f
			}
		}

		if startedAtNull.Valid {
			job.StartedAt = &startedAtNull.Time
		}
		if completedAtNull.Valid {
			job.CompletedAt = &completedAtNull.Time
		}
		if errMsgNull.Valid {
			job.ErrorMessage = &errMsgNull.String
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// StartJob spawns a background scrape worker
func (jm *JobManager) StartJob(id string) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	if _, ok := jm.activeJobs[id]; ok {
		return fmt.Errorf("job is already active")
	}

	job, err := jm.GetJob(id)
	if err != nil {
		return err
	}

	if job.Status != "queued" && job.Status != "failed" && job.Status != "canceled" && job.Status != "paused" {
		return fmt.Errorf("job cannot be started from state: %s", job.Status)
	}

	ctx, cancel := context.WithCancel(context.Background())
	aj := &ActiveJob{
		JobID:      id,
		Ctx:        ctx,
		Cancel:     cancel,
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
	jm.activeJobs[id] = aj

	// Mark status as running
	_, err = jm.db.Exec(`
		UPDATE jobs SET status = 'running', started_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP, error_message = NULL
		WHERE id = ?
	`, id)
	if err != nil {
		cancel()
		delete(jm.activeJobs, id)
		return err
	}

	go jm.runScraper(aj, job)
	return nil
}

// PauseJob pauses a running job
func (jm *JobManager) PauseJob(id string) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	aj, ok := jm.activeJobs[id]
	if !ok {
		return fmt.Errorf("job is not running")
	}

	if !atomic.CompareAndSwapInt32(&aj.isPaused, 0, 1) {
		return fmt.Errorf("job is already paused")
	}

	_, err := jm.db.Exec(`UPDATE jobs SET status = 'paused', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return err
	}

	logger := NewJobLogger(jm.db, id)
	logger.Infof("Job paused by user command.")

	return nil
}

// ResumeJob resumes a paused job
func (jm *JobManager) ResumeJob(id string) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	aj, ok := jm.activeJobs[id]
	if !ok {
		// If not in active memory, but paused in database, start it.
		job, err := jm.GetJob(id)
		if err != nil {
			return err
		}
		if job.Status == "paused" {
			jm.mu.Unlock()
			err = jm.StartJob(id)
			jm.mu.Lock()
			return err
		}
		return fmt.Errorf("job is not active or paused")
	}

	if !atomic.CompareAndSwapInt32(&aj.isPaused, 1, 0) {
		return fmt.Errorf("job is not paused")
	}

	_, err := jm.db.Exec(`UPDATE jobs SET status = 'running', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return err
	}

	logger := NewJobLogger(jm.db, id)
	logger.Infof("Job resumed by user command.")

	return nil
}

// CancelJob cancels a running or paused job
func (jm *JobManager) CancelJob(id string) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	aj, ok := jm.activeJobs[id]
	if !ok {
		// Just mark as canceled in DB if not in active jobs (e.g. queued)
		_, err := jm.db.Exec(`UPDATE jobs SET status = 'canceled', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
		return err
	}

	aj.Cancel()
	delete(jm.activeJobs, id)

	_, err := jm.db.Exec(`UPDATE jobs SET status = 'canceled', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// CleanUpActive removes job from memory registry
func (jm *JobManager) cleanUpActive(id string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	delete(jm.activeJobs, id)
}

// runScraper is the core background thread loop
func (jm *JobManager) runScraper(aj *ActiveJob, job *Job) {
	defer jm.cleanUpActive(aj.JobID)

	logger := NewJobLogger(jm.db, aj.JobID)
	logger.Infof("Starting scrape job: %s", job.Name)

	jar, _ := cookiejar.New(nil)
	var httpClient *http.Client
	if jm.Rotator != nil {
		httpClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: jm.Rotator.GetRoundTripper(),
			Jar:       jar,
		}
	} else {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		}
	}

	scr := scraper.NewScraper(jm.db, httpClient, jm.jwtSecret, job.SaveRawAgents, logger, aj.JobID)

	var filters *scraper.JobFilters
	if job.Filters != nil {
		filters = &scraper.JobFilters{
			State:      job.Filters.State,
			City:       job.Filters.City,
			Zip:        job.Filters.Zip,
			Brokerage:  job.Filters.Brokerage,
			AgentName:  job.Filters.AgentName,
			AreaServed: job.Filters.AreaServed,
		}
	}

	logger.Infof("Fetching matching results count...")
	totalResults, err := scr.GetTotalResults(aj.Ctx, filters)
	if err != nil {
		logger.Errorf("Failed to get total results: %v", err)
		jm.markFailed(aj.JobID, err.Error())
		return
	}

	logger.Infof("STATS: Matching agents found: %d", totalResults)

	if job.MaxAgentsLimit > 0 && totalResults > job.MaxAgentsLimit {
		logger.Infof("Applying Max Agent Limit override: capping at %d (original matching count: %d)", job.MaxAgentsLimit, totalResults)
		totalResults = job.MaxAgentsLimit
	}

	// Build all offset requests
	var allOffsets []int
	for offset := 0; offset < totalResults; offset += defaultResultsPerPage {
		allOffsets = append(allOffsets, offset)
	}

	// Load completed requests from job_requests to support resumption
	completedOffsets := make(map[int]bool)
	rows, err := jm.db.Query(`SELECT offset FROM job_requests WHERE job_id = ?`, aj.JobID)
	if err == nil {
		for rows.Next() {
			var off int
			if err := rows.Scan(&off); err == nil {
				completedOffsets[off] = true
			}
		}
		rows.Close()
	}

	var remainingOffsets []int
	for _, off := range allOffsets {
		if !completedOffsets[off] {
			remainingOffsets = append(remainingOffsets, off)
		}
	}

	totalEstimated := len(allOffsets)
	completedCount := len(completedOffsets)

	_, _ = jm.db.Exec(`
		UPDATE jobs SET total_estimated_requests = ?, completed_requests = ? WHERE id = ?
	`, totalEstimated, completedCount, aj.JobID)

	logger.Infof("Total estimated requests: %d, remaining requests: %d", totalEstimated, len(remainingOffsets))

	if len(remainingOffsets) == 0 {
		logger.Infof("========== COMPLETE ==========")
		jm.markCompleted(aj.JobID)
		return
	}

	sem := make(chan struct{}, job.Concurrency)
	var processedCount int32
	var failedCount int32
	var savedCount int32

	// Load currently saved agent count
	_ = jm.db.QueryRow(`SELECT COUNT(DISTINCT agent_id) FROM job_agents WHERE job_id = ?`, aj.JobID).Scan(&savedCount)

	var wg sync.WaitGroup
	count := 0

	for _, offset := range remainingOffsets {
		// Handle pause check
		for {
			if aj.Ctx.Err() != nil {
				break
			}
			if atomic.LoadInt32(&aj.isPaused) == 1 {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			break
		}

		if aj.Ctx.Err() != nil {
			break
		}

		select {
		case <-aj.Ctx.Done():
			break
		case sem <- struct{}{}:
		}

		if aj.Ctx.Err() != nil {
			<-sem
			break
		}

		wg.Add(1)
		go func(off int) {
			defer func() {
				<-sem
				wg.Done()
			}()

			logger.Infof("FETCHING: Agents (offset %d, limit %d)", off, defaultResultsPerPage)

			agents, err := scr.GetAgents(aj.Ctx, off, defaultResultsPerPage, filters)
			if err != nil {
				logger.Errorf("Failed to retrieve request (offset %d): %v", off, err)
				atomic.AddInt32(&failedCount, 1)
				jm.updateProgress(aj.JobID, completedCount+int(atomic.LoadInt32(&processedCount)), int(atomic.LoadInt32(&failedCount)), int(atomic.LoadInt32(&savedCount)))
				return
			}

			// Store agents in DB
			agentsSavedThisPage := 0
			for _, agent := range agents {
				if aj.Ctx.Err() != nil {
					return
				}
				if err := scr.StoreAgent(agent); err != nil {
					logger.Errorf("error storing agent (ID %s): %v", agent.ID, err)
				} else {
					agentsSavedThisPage++
				}
			}

			// Save job request progress
			_, err = jm.db.Exec(`
				INSERT OR IGNORE INTO job_requests (job_id, offset, results_per_page)
				VALUES (?, ?, ?)
			`, aj.JobID, off, defaultResultsPerPage)
			if err != nil {
				logger.Errorf("failed to save job request progress: %v", err)
			}

			atomic.AddInt32(&processedCount, 1)
			atomic.AddInt32(&savedCount, int32(agentsSavedThisPage))

			jm.updateProgress(aj.JobID, completedCount+int(atomic.LoadInt32(&processedCount)), int(atomic.LoadInt32(&failedCount)), int(atomic.LoadInt32(&savedCount)))
		}(offset)

		count++
		if count%job.ThrottleRequestLimit == 0 {
			logger.Infof("THROTTLING: Request limit of %d reached. Pausing for %v.", job.ThrottleRequestLimit, defaultThrottleTimeout)
			select {
			case <-aj.Ctx.Done():
				break
			case <-time.After(defaultThrottleTimeout):
			}
		}
	}

	wg.Wait()

	if aj.Ctx.Err() != nil {
		logger.Warnf("Job execution interrupted or canceled.")
		return
	}

	logger.Infof("========== COMPLETE ==========")
	jm.markCompleted(aj.JobID)
}

func (jm *JobManager) updateProgress(jobID string, completed, failed, saved int) {
	_, _ = jm.db.Exec(`
		UPDATE jobs 
		SET completed_requests = ?, failed_requests = ?, agents_saved = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`, completed, failed, saved, jobID)
}

func (jm *JobManager) markCompleted(jobID string) {
	_, _ = jm.db.Exec(`
		UPDATE jobs 
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`, jobID)
}

func (jm *JobManager) markFailed(jobID string, errMsg string) {
	_, _ = jm.db.Exec(`
		UPDATE jobs 
		SET status = 'failed', completed_at = CURRENT_TIMESTAMP, error_message = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`, errMsg, jobID)
}

// JobLogger helper struct
type JobLogger struct {
	db    *sql.DB
	jobID string
}

func NewJobLogger(db *sql.DB, jobID string) *JobLogger {
	return &JobLogger{db: db, jobID: jobID}
}

func (jl *JobLogger) log(level, message string) {
	_, err := jl.db.Exec(`
		INSERT INTO job_logs (job_id, timestamp, level, message)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?)
	`, jl.jobID, level, message)
	if err != nil {
		fmt.Printf("failed to write job log: %v\n", err)
	}
}

func (jl *JobLogger) Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("[INFO] [Job %s] %s\n", jl.jobID, msg)
	jl.log("INFO", msg)
}

func (jl *JobLogger) Warnf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("[WARN] [Job %s] %s\n", jl.jobID, msg)
	jl.log("WARN", msg)
}

func (jl *JobLogger) Errorf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("[ERROR] [Job %s] %s\n", jl.jobID, msg)
	jl.log("ERROR", msg)
}

func (jl *JobLogger) Debugf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	jl.log("DEBUG", msg)
}

func (jl *JobLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("[FATAL] [Job %s] %s\n", jl.jobID, msg)
	jl.log("FATAL", msg)
}
