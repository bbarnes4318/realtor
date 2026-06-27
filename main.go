package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/go-libsql"

	"github.com/suffer-sami/realtor-scraper/pkg/api"
	"github.com/suffer-sami/realtor-scraper/pkg/export"
	"github.com/suffer-sami/realtor-scraper/pkg/job"
)

func main() {
	// Parse CLI flags
	serverMode := flag.Bool("server", false, "Run in HTTP API Server mode")
	port := flag.Int("port", 8080, "Port for the HTTP API Server")
	threads := flag.Int("threads", 3, "Scraper concurrency limit")
	limit := flag.Int("limit", 0, "Max agents to scrape")
	jobName := flag.String("job-name", "CLI Scrape", "Scrape job name")
	devMode := flag.Bool("dev", false, "Run in development mode (applies short request limits)")
	exportFile := flag.String("export", "", "Export scraped agents to specified CSV filename after run")
	flag.Parse()

	// Load env
	_ = godotenv.Load() // optional loader, doesn't panic if file doesn't exist

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default_realtor_jwt_secret_token"
	}

	// Database Connection setup
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Setup Job Manager
	jm := job.NewJobManager(db, jwtSecret)

	if *serverMode {
		// Run API Server
		serverPort := *port
		if envPortStr := os.Getenv("PORT"); envPortStr != "" {
			if p, err := strconv.Atoi(envPortStr); err == nil {
				serverPort = p
			}
		}

		server := api.NewAPIServer(db, jm, serverPort)
		if err := server.Start(); err != nil {
			log.Fatalf("API Server failed: %v", err)
		}
	} else {
		// Run CLI Scraper Mode
		runCLIScrape(db, jm, *threads, *limit, *jobName, *devMode, *exportFile)
	}
}

func initDB() (*sql.DB, error) {
	useDbLocal := true
	if useDbLocalStr := os.Getenv("USE_DB_LOCAL"); useDbLocalStr != "" {
		if val, err := strconv.ParseBool(useDbLocalStr); err == nil {
			useDbLocal = val
		}
	}

	var dbPath string
	if useDbLocal {
		dbFile := os.Getenv("DB_FILE")
		if dbFile == "" {
			dbFile = "file:local.db"
		}
		dbPath = dbFile
	} else {
		dbURL := os.Getenv("DB_URL")
		if dbURL == "" {
			return nil, fmt.Errorf("DB_URL must be set when USE_DB_LOCAL is false")
		}
		dbAuthToken := os.Getenv("DB_AUTH_TOKEN")
		if dbAuthToken == "" {
			return nil, fmt.Errorf("DB_AUTH_TOKEN must be set when USE_DB_LOCAL is false")
		}

		if !strings.HasPrefix(dbURL, "libsql") {
			return nil, fmt.Errorf("invalid database URL format (must start with libsql)")
		}
		dbPath = fmt.Sprintf("%s?authToken=%s", dbURL, dbAuthToken)
	}

	db, err := sql.Open("libsql", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping database
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func runCLIScrape(db *sql.DB, jm *job.JobManager, threads, limit int, jobName string, devMode bool, exportFile string) {
	fmt.Printf("Starting CLI Scraper Run: %s (concurrency: %d, limit: %d, dev: %t)\n", jobName, threads, limit, devMode)

	if devMode && limit == 0 {
		limit = 10 // safe short default for quick tests
	}

	saveRawAgents := false
	if rawStr := os.Getenv("SAVE_RAW_AGENTS"); rawStr != "" {
		if val, err := strconv.ParseBool(rawStr); err == nil {
			saveRawAgents = val
		}
	}

	// Create job in database
	j, err := jm.CreateJob(jobName, limit, threads, 5, saveRawAgents, "local", nil)
	if err != nil {
		log.Fatalf("Failed to create scrape job: %v", err)
	}

	// Start job
	err = jm.StartJob(j.ID)
	if err != nil {
		log.Fatalf("Failed to start scrape job: %v", err)
	}

	// Monitor progress until finished
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		jobDetail, err := jm.GetJob(j.ID)
		if err != nil {
			log.Fatalf("Failed to fetch job status: %v", err)
		}

		fmt.Printf("Job Status: %s | Estimated: %d | Completed: %d | Failed: %d | Agents Saved: %d\n",
			jobDetail.Status, jobDetail.TotalEstimatedRequests, jobDetail.CompletedRequests, jobDetail.FailedRequests, jobDetail.AgentsSaved)

		if jobDetail.Status == "completed" {
			fmt.Println("Job finished successfully!")
			break
		}
		if jobDetail.Status == "failed" {
			errMsg := ""
			if jobDetail.ErrorMessage != nil {
				errMsg = *jobDetail.ErrorMessage
			}
			log.Fatalf("Job failed: %s", errMsg)
		}
		if jobDetail.Status == "canceled" {
			log.Fatalf("Job was canceled.")
		}
	}

	// Direct Export if specified
	if exportFile != "" {
		fmt.Printf("Exporting results to %s...\n", exportFile)
		f, err := os.Create(exportFile)
		if err != nil {
			log.Fatalf("Failed to create export file: %v", err)
		}
		defer f.Close()

		opt := export.ExportOptions{
			JobID:       j.ID,
			DedupePhone: true,
			DedupeURL:   true,
		}

		if err := export.GenerateCSV(db, f, opt); err != nil {
			log.Fatalf("Export failed: %v", err)
		}
		fmt.Println("Export completed successfully.")
	}
}

// Dummy conversion just to avoid compiling issue if strconv isn't imported elsewhere
var _ = time.Nanosecond
var _ = strings.Compare
func strconvDummy() {
	_, _ = strconv.Atoi("1")
}
