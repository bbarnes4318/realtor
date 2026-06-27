# Realtor Scraper Web Application

![banner](./assets/banner.svg)

## Overview
This project turns the Go-based Realtor.com agent scraper into a full-featured, production-ready internal web application. It features a modern Next.js dashboard panel, SQL database storage (SQLite or Turso), background scrape run scheduling, live streaming console logs, paginated search, and phone/URL deduplicated CSV exports.

---

> [!IMPORTANT]
> **🛡️ LEGAL & COMPLIANCE WARNING**
> - This tool is designed strictly for internal research and data aggregation where the operator has the explicit legal right to collect and process such data.
> - **Anti-Abuse Systems:** The code does **NOT** bypass CAPTCHAs, paywalls, log-ins, or other access control systems. Bypassing these controls is not supported.
> - **Privacy Laws:** Phone numbers, email addresses, and MLS records collected via this scraper may be subject to marketing regulations (TCPA, CAN-SPAM), state/local privacy laws (CCPA, GDPR), and platform terms. Ensure full legal clearance before utilizing the output datasets.

---

## Architecture

- **Backend (Go):** Modular service layer consisting of:
  - `pkg/scraper`: rotation parameters, backoff retries, and normalizers.
  - `pkg/job`: thread-safe concurrent jobs registry (start, pause, resume, cancel, DB-backed logging).
  - `pkg/export`: CSV dataset generator with optional deduplication.
  - `pkg/api`: HTTP router exposing `/api/health`, `/api/stats`, `/api/jobs`, `/api/agents`, `/api/export`.
- **Frontend (Next.js + Tailwind + TypeScript):** A sleek slate dashboard for overview statistics, run setup, real-time log viewers, search tables, and download portals.
- **Database (SQLite/libsql):** Goose-migrated schema storing agents, offices, MLS records, social links, phone numbers, and job runs.

---

## Prerequisites

Ensure you have the following installed:
1. **Go (v1.22+):** [https://golang.org/dl/](https://golang.org/dl/)
2. **Node.js (v18+):** [https://nodejs.org/](https://nodejs.org/)
3. **Goose CLI:** Database migration tool
   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   ```

---

## Setup Instructions

### 1. Configure Environments
Copy the template `.env.example` into `.env` and fill out your variables:
```bash
cp .env.example .env
```
Ensure you set your Realtor API key/secret in `JWT_SECRET`.

### 2. Apply Database Migrations
Use Goose to apply schema migrations to a local SQLite database:
```bash
goose -dir sql/schema sqlite3 local.db up
```

### 3. Run Backend API Server
Start the HTTP API server:
```bash
go run . --server --port 8080
```

### 4. Run Frontend Dashboard
Open a new terminal tab, navigate to the frontend directory, install dependencies, and start the development server:
```bash
cd frontend
npm install
npm run dev
```
Open [http://localhost:3000](http://localhost:3000) to view the dashboard!

---

## CLI Scraper Usage

You can still use the scraper directly via the terminal interface. Running the CLI creates a scrape job in the database to track execution progress.

```bash
# Standard scrape run
go run .

# Run with 5 threads
go run . --threads 5

# Scrape a capped count of 100 agents
go run . --limit 100

# Name the run in the database
go run . --job-name "California Agent Scan"

# Run in short development mode (caps at 10 agents, runs quick checks)
go run . --dev

# Scrape and export to CSV directly
go run . --job-name "CLI Export Run" --limit 50 --export agents_output.csv
```

---

## Docker Compose Setup

To launch the full stack (Go Backend API + Next.js Frontend + local database volume) automatically:

```bash
docker-compose up --build
```

- **Frontend Panel:** [http://localhost:3000](http://localhost:3000)
- **Go API Server:** [http://localhost:8080](http://localhost:8080)
- **SQLite Database volume:** Persisted inside `./data` or Docker volumes.

---

## Production Deployment (87.99.155.241 / realtors.leadsbystorm.com)

To deploy the application to your Ubuntu/Debian production server:

1. **DNS Setup:** Set your domain's DNS `A Record` for `realtors.leadsbystorm.com` to point to the server IP `87.99.155.241`.
2. **Execute Deployment Script:** SSH into your server and run:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/suffer-sami/realtor-scraper/main/deploy/deploy.sh -o deploy.sh
   chmod +x deploy.sh
   ./deploy.sh
   ```
   *Alternatively, copy the `./deploy` folder contents to your server manually and execute `./deploy.sh`.*

The script will automatically install Nginx, Docker, Docker Compose, Certbot, pull the repository, build containers, link the reverse proxy, and obtain a free Let's Encrypt SSL certificate.