package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/suffer-sami/realtor-scraper/pkg/export"
	"github.com/suffer-sami/realtor-scraper/pkg/job"
)

type APIServer struct {
	db         *sql.DB
	jobManager *job.JobManager
	port       int
}

func NewAPIServer(db *sql.DB, jm *job.JobManager, port int) *APIServer {
	return &APIServer{
		db:         db,
		jobManager: jm,
		port:       port,
	}
}

func (s *APIServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/jobs", s.handleJobs)
	mux.HandleFunc("/api/jobs/", s.handleJobDetailAndActions) // Matches /api/jobs/:id and /api/jobs/:id/*
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentDetail)
	mux.HandleFunc("/api/export/agents.csv", s.handleExportAgents)
	mux.HandleFunc("/api/export/jobs/", s.handleExportJobAgents) // Matches /api/export/jobs/:id.csv

	// Proxies routes
	mux.HandleFunc("/api/proxies", s.handleProxies)
	mux.HandleFunc("/api/proxies/", s.handleProxyActions)
	mux.HandleFunc("/api/flame-proxies/", s.handleFlameProxies)

	// Wrap with CORS middleware
	handler := s.corsMiddleware(mux)

	fmt.Printf("HTTP API Server listening on port %d...\n", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), handler)
}

func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *APIServer) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *APIServer) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

// GET /api/health
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/stats
func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var totalAgents, totalPhones, totalOffices, totalBrokerages int
	var totalJobs, activeJobs, failedJobs int
	var lastRunDateNull sql.NullTime

	_ = s.db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&totalAgents)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM phones").Scan(&totalPhones)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM offices").Scan(&totalOffices)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM brokers").Scan(&totalBrokerages)

	_ = s.db.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&totalJobs)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = 'running'").Scan(&activeJobs)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = 'failed'").Scan(&failedJobs)
	_ = s.db.QueryRow("SELECT MAX(started_at) FROM jobs").Scan(&lastRunDateNull)

	var lastRunDateStr string
	if lastRunDateNull.Valid {
		lastRunDateStr = lastRunDateNull.Time.Format(time.RFC3339)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"total_agents":     totalAgents,
		"total_phones":     totalPhones,
		"total_offices":    totalOffices,
		"total_brokerages": totalBrokerages,
		"total_jobs":       totalJobs,
		"active_jobs":      activeJobs,
		"failed_jobs":      failedJobs,
		"last_run_date":    lastRunDateStr,
	})
}

// GET or POST /api/jobs
func (s *APIServer) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jobs, err := s.jobManager.ListJobs()
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, jobs)

	case "POST":
		var req struct {
			Name                 string          `json:"name"`
			MaxAgentsLimit       int             `json:"max_agents_limit"`
			Concurrency          int             `json:"concurrency"`
			ThrottleRequestLimit int             `json:"throttle_request_limit"`
			SaveRawAgents        bool            `json:"save_raw_agents"`
			DBMode               string          `json:"db_mode"`
			Filters              *job.JobFilters `json:"filters"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Apply defaults
		if req.Concurrency <= 0 {
			req.Concurrency = 3
		}
		if req.ThrottleRequestLimit <= 0 {
			req.ThrottleRequestLimit = 5
		}
		if req.DBMode == "" {
			req.DBMode = "local"
		}

		job, err := s.jobManager.CreateJob(req.Name, req.MaxAgentsLimit, req.Concurrency, req.ThrottleRequestLimit, req.SaveRawAgents, req.DBMode, req.Filters)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.respondJSON(w, http.StatusCreated, job)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// GET /api/jobs/:id and POST actions (start, pause, resume, cancel)
func (s *APIServer) handleJobDetailAndActions(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/jobs/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		s.respondError(w, http.StatusBadRequest, "Missing job ID")
		return
	}

	jobID := pathParts[0]

	if len(pathParts) == 1 {
		// GET /api/jobs/:id
		if r.Method != "GET" {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		job, err := s.jobManager.GetJob(jobID)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Job not found")
			return
		}

		s.respondJSON(w, http.StatusOK, job)
		return
	}

	action := pathParts[1]
	if action == "logs" {
		if r.Method != "GET" {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handleJobLogs(w, r, jobID)
		return
	}

	if r.Method != "POST" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var err error
	switch action {
	case "start":
		err = s.jobManager.StartJob(jobID)
	case "pause":
		err = s.jobManager.PauseJob(jobID)
	case "resume":
		err = s.jobManager.ResumeJob(jobID)
	case "cancel":
		err = s.jobManager.CancelJob(jobID)
	default:
		s.respondError(w, http.StatusBadRequest, "Unknown action: "+action)
		return
	}

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// GET /api/jobs/:id/logs
func (s *APIServer) handleJobLogs(w http.ResponseWriter, r *http.Request, jobID string) {
	// Query logs from database
	rows, err := s.db.Query(`
		SELECT timestamp, level, message 
		FROM job_logs 
		WHERE job_id = ? 
		ORDER BY timestamp ASC
	`, jobID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type logLine struct {
		Timestamp time.Time `json:"timestamp"`
		Level     string    `json:"level"`
		Message   string    `json:"message"`
	}

	var logs []logLine
	for rows.Next() {
		var l logLine
		if err := rows.Scan(&l.Timestamp, &l.Level, &l.Message); err == nil {
			logs = append(logs, l)
		}
	}

	s.respondJSON(w, http.StatusOK, logs)
}

// GET /api/agents
func (s *APIServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	state := q.Get("state")
	city := q.Get("city")
	zip := q.Get("zip")
	brokerage := q.Get("brokerage")
	hasPhone := q.Get("has_phone")
	hasMobile := q.Get("has_mobile")
	hasOfficePhone := q.Get("has_office_phone")
	language := q.Get("language")
	areaServed := q.Get("area_served")
	jobID := q.Get("job_id")

	// Dynamic querying
	query := `
		SELECT a.id, a.person_name, a.title, a.slogan, a.email, a.agent_rating, a.profile_url, a.photo,
		       addr.city, addr.state_code, addr.postal_code,
		       b.name, o.name
		FROM agents a
	`
	if jobID != "" {
		query += " JOIN job_agents ja ON a.id = ja.agent_id"
	}
	query += `
		LEFT JOIN addresses addr ON a.address_id = addr.id
		LEFT JOIN brokers b ON a.broker_id = b.id
		LEFT JOIN offices o ON a.office_id = o.id
		WHERE 1=1
	`
	var args []interface{}

	if jobID != "" {
		query += " AND ja.job_id = ?"
		args = append(args, jobID)
	}

	if state != "" {
		query += " AND addr.state_code = ?"
		args = append(args, state)
	}
	if city != "" {
		query += " AND addr.city LIKE ?"
		args = append(args, "%"+city+"%")
	}
	if zip != "" {
		query += " AND addr.postal_code = ?"
		args = append(args, zip)
	}
	if brokerage != "" {
		query += " AND b.name LIKE ?"
		args = append(args, "%"+brokerage+"%")
	}
	if hasPhone == "true" {
		query += " AND a.id IN (SELECT agent_id FROM agent_phones)"
	}
	if hasMobile == "true" {
		query += " AND a.id IN (SELECT ap.agent_id FROM agent_phones ap JOIN phones p ON ap.phone_id = p.id WHERE p.type = 'mobile')"
	}
	if hasOfficePhone == "true" {
		query += " AND (a.office_id IN (SELECT office_id FROM office_phones) OR a.id IN (SELECT ap.agent_id FROM agent_phones ap JOIN phones p ON ap.phone_id = p.id WHERE p.type = 'office'))"
	}
	if language != "" {
		query += " AND a.id IN (SELECT al.agent_id FROM agent_languages al JOIN languages l ON al.language_id = l.id WHERE l.name LIKE ?)"
		args = append(args, "%"+language+"%")
	}
	if areaServed != "" {
		query += " AND a.id IN (SELECT asa.agent_id FROM agent_served_areas asa JOIN areas ar ON asa.area_id = ar.id WHERE ar.name LIKE ?)"
		args = append(args, "%"+areaServed+"%")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	query += " ORDER BY a.person_name ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type agentItem struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		Title      string   `json:"title"`
		Slogan     string   `json:"slogan"`
		Email      string   `json:"email"`
		Rating     int      `json:"rating"`
		ProfileURL string   `json:"profile_url"`
		Photo      string   `json:"photo"`
		City       string   `json:"city"`
		State      string   `json:"state"`
		Zip        string   `json:"zip"`
		Brokerage  string   `json:"brokerage"`
		OfficeName string   `json:"office_name"`
		Phones     []string `json:"phones"`
		Languages  []string `json:"languages"`
		Areas      []string `json:"areas"`
	}

	agents := []agentItem{}
	for rows.Next() {
		var a agentItem
		var titleNull, sloganNull, emailNull, profileNull, photoNull sql.NullString
		var cityNull, stateNull, zipNull, brokerNull, officeNull sql.NullString
		var ratingNull sql.NullInt64

		err := rows.Scan(
			&a.ID, &a.Name, &titleNull, &sloganNull, &emailNull, &ratingNull, &profileNull, &photoNull,
			&cityNull, &stateNull, &zipNull, &brokerNull, &officeNull,
		)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		a.Title = titleNull.String
		a.Slogan = sloganNull.String
		a.Email = emailNull.String
		a.Rating = int(ratingNull.Int64)
		a.ProfileURL = profileNull.String
		a.Photo = photoNull.String
		a.City = cityNull.String
		a.State = stateNull.String
		a.Zip = zipNull.String
		a.Brokerage = brokerNull.String
		a.OfficeName = officeNull.String

		// Populate phones, languages, areas in simple batches/queries
		a.Phones = s.fetchAgentPhones(a.ID)
		a.Languages = s.fetchAgentLanguages(a.ID)
		a.Areas = s.fetchAgentAreas(a.ID)

		agents = append(agents, a)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (s *APIServer) fetchAgentPhones(agentID string) []string {
	rows, err := s.db.Query(`
		SELECT p.number || ' (' || p.type || ')'
		FROM phones p
		JOIN agent_phones ap ON p.id = ap.phone_id
		WHERE ap.agent_id = ?
	`, agentID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	phones := []string{}
	for rows.Next() {
		var num string
		if err := rows.Scan(&num); err == nil {
			phones = append(phones, num)
		}
	}
	return phones
}

func (s *APIServer) fetchAgentLanguages(agentID string) []string {
	rows, err := s.db.Query(`
		SELECT l.name FROM languages l
		JOIN agent_languages al ON l.id = al.language_id
		WHERE al.agent_id = ?
	`, agentID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	langs := []string{}
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err == nil {
			langs = append(langs, lang)
		}
	}
	return langs
}

func (s *APIServer) fetchAgentAreas(agentID string) []string {
	rows, err := s.db.Query(`
		SELECT ar.name || ' (' || ar.state_code || ')'
		FROM areas ar
		JOIN agent_served_areas asa ON ar.id = asa.area_id
		WHERE asa.agent_id = ?
	`, agentID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	areas := []string{}
	for rows.Next() {
		var area string
		if err := rows.Scan(&area); err == nil {
			areas = append(areas, area)
		}
	}
	return areas
}

// GET /api/agents/:id
func (s *APIServer) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	agentID := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	if agentID == "" || strings.Contains(agentID, "/") {
		s.respondError(w, http.StatusBadRequest, "Invalid agent ID")
		return
	}

	var a struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		FirstName      string   `json:"first_name"`
		LastName       string   `json:"last_name"`
		Title          string   `json:"title"`
		Slogan         string   `json:"slogan"`
		Email          string   `json:"email"`
		Description    string   `json:"description"`
		Rating         int      `json:"rating"`
		RecomsCount    int      `json:"recommendations_count"`
		ReviewsCount   int      `json:"review_count"`
		ProfileURL     string   `json:"profile_url"`
		Website        string   `json:"website"`
		Photo          string   `json:"photo"`
		Video          string   `json:"video"`
		AddressLine1   string   `json:"address_line_1"`
		AddressLine2   string   `json:"address_line_2"`
		City           string   `json:"city"`
		State          string   `json:"state"`
		Zip            string   `json:"zip"`
		BrokerageName  string   `json:"brokerage_name"`
		OfficeName     string   `json:"office_name"`
		OfficeEmail    string   `json:"office_email"`
		OfficeWebsite  string   `json:"office_website"`
		OfficeSlogan   string   `json:"office_slogan"`
		Phones         []string `json:"phones"`
		Languages      []string `json:"languages"`
		Areas          []string `json:"areas"`
		Licenses       []string `json:"licenses"`
		MLSCodes       []string `json:"mls_codes"`
		SocialProfiles []string `json:"social_profiles"`
	}

	var titleNull, sloganNull, emailNull, descNull, profileNull, webNull, photoNull, videoNull sql.NullString
	var lineNull, line2Null, cityNull, stateNull, stateCodeNull, zipNull sql.NullString
	var brokerNull, officeNull, oEmailNull, oWebNull, oSloganNull sql.NullString

	err := s.db.QueryRow(`
		SELECT a.id, a.person_name, a.first_name, a.last_name, a.title, a.slogan, a.email, a.description, a.agent_rating, a.recommendations_count, a.review_count, a.profile_url, a.website, a.photo, a.video,
		       addr.line, addr.line2, addr.city, addr.state, addr.state_code, addr.postal_code,
		       b.name, o.name, o.email, o.website, o.slogan
		FROM agents a
		LEFT JOIN addresses addr ON a.address_id = addr.id
		LEFT JOIN brokers b ON a.broker_id = b.id
		LEFT JOIN offices o ON a.office_id = o.id
		WHERE a.id = ?
	`, agentID).Scan(
		&a.ID, &a.Name, &a.FirstName, &a.LastName, &titleNull, &sloganNull, &emailNull, &descNull, &a.Rating, &a.RecomsCount, &a.ReviewsCount, &profileNull, &webNull, &photoNull, &videoNull,
		&lineNull, &line2Null, &cityNull, &stateNull, &stateCodeNull, &zipNull,
		&brokerNull, &officeNull, &oEmailNull, &oWebNull, &oSloganNull,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			s.respondError(w, http.StatusNotFound, "Agent not found")
		} else {
			s.respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	a.Title = titleNull.String
	a.Slogan = sloganNull.String
	a.Email = emailNull.String
	a.Description = descNull.String
	a.ProfileURL = profileNull.String
	a.Website = webNull.String
	a.Photo = photoNull.String
	a.Video = videoNull.String
	a.AddressLine1 = lineNull.String
	a.AddressLine2 = line2Null.String
	a.City = cityNull.String
	a.State = stateCodeNull.String
	a.Zip = zipNull.String
	a.BrokerageName = brokerNull.String
	a.OfficeName = officeNull.String
	a.OfficeEmail = oEmailNull.String
	a.OfficeWebsite = oWebNull.String
	a.OfficeSlogan = oSloganNull.String

	a.Phones = s.fetchAgentPhones(agentID)
	a.Languages = s.fetchAgentLanguages(agentID)
	a.Areas = s.fetchAgentAreas(agentID)

	// Licenses
	lRows, _ := s.db.Query(`
		SELECT fl.license_number || ' (' || fl.state_code || ')' 
		FROM feed_licenses fl 
		JOIN agent_feed_licenses afl ON fl.id = afl.feed_license_id 
		WHERE afl.agent_id = ?
	`, agentID)
	if lRows != nil {
		for lRows.Next() {
			var lic string
			if err := lRows.Scan(&lic); err == nil {
				a.Licenses = append(a.Licenses, lic)
			}
		}
		lRows.Close()
	}

	// MLS
	mRows, _ := s.db.Query(`
		SELECT mls.abbreviation 
		FROM multiple_listing_services mls 
		JOIN agent_multiple_listing_services amls ON mls.id = amls.multiple_listing_service_id 
		WHERE amls.agent_id = ?
	`, agentID)
	if mRows != nil {
		for mRows.Next() {
			var mls string
			if err := mRows.Scan(&mls); err == nil {
				a.MLSCodes = append(a.MLSCodes, mls)
			}
		}
		mRows.Close()
	}

	// Socials
	sRows, _ := s.db.Query(`
		SELECT href || ' (' || type || ')' 
		FROM social_medias 
		WHERE agent_id = ?
	`, agentID)
	if sRows != nil {
		for sRows.Next() {
			var soc string
			if err := sRows.Scan(&soc); err == nil {
				a.SocialProfiles = append(a.SocialProfiles, soc)
			}
		}
		sRows.Close()
	}

	s.respondJSON(w, http.StatusOK, a)
}

// GET /api/export/agents.csv
func (s *APIServer) handleExportAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	q := r.URL.Query()
	opt := export.ExportOptions{
		State:       q.Get("state"),
		City:        q.Get("city"),
		Zip:         q.Get("zip"),
		DedupePhone: q.Get("dedupe_phone") == "true",
		DedupeURL:   q.Get("dedupe_url") == "true",
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=agents.csv")

	if err := export.GenerateCSV(s.db, w, opt); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}
}

// GET /api/export/jobs/:id.csv
func (s *APIServer) handleExportJobAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	suffix := strings.TrimPrefix(r.URL.Path, "/api/export/jobs/")
	if !strings.HasSuffix(suffix, ".csv") {
		s.respondError(w, http.StatusBadRequest, "Invalid request path")
		return
	}

	jobID := strings.TrimSuffix(suffix, ".csv")
	q := r.URL.Query()
	opt := export.ExportOptions{
		JobID:       jobID,
		DedupePhone: q.Get("dedupe_phone") == "true",
		DedupeURL:   q.Get("dedupe_url") == "true",
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=job_%s.csv", jobID))

	if err := export.GenerateCSV(s.db, w, opt); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}
}
