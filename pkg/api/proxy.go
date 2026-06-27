package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/suffer-sami/realtor-scraper/pkg/proxy"
)

// GET /api/proxies
// POST /api/proxies
func (s *APIServer) handleProxies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rows, err := s.db.Query("SELECT id, url, status, created_at FROM proxies ORDER BY created_at DESC")
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type ProxyItem struct {
			ID        string    `json:"id"`
			URL       string    `json:"url"`
			Status    string    `json:"status"`
			CreatedAt time.Time `json:"created_at"`
		}

		proxies := []ProxyItem{}
		for rows.Next() {
			var p ProxyItem
			if err := rows.Scan(&p.ID, &p.URL, &p.Status, &p.CreatedAt); err != nil {
				s.respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			proxies = append(proxies, p)
		}
		s.respondJSON(w, http.StatusOK, proxies)

	case "POST":
		var req struct {
			URL  string `json:"url"`
			Bulk string `json:"bulk"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Import helper
		importProxy := func(raw string) error {
			parsed, err := proxy.ParseProxyURL(raw)
			if err != nil {
				return err
			}
			id := uuid.New().String()
			_, err = s.db.Exec(`
				INSERT INTO proxies (id, url, status, created_at)
				VALUES (?, ?, 'active', CURRENT_TIMESTAMP)
				ON CONFLICT(url) DO UPDATE SET status = 'active'
			`, id, parsed)
			return err
		}

		if req.Bulk != "" {
			lines := strings.Split(req.Bulk, "\n")
			importedCount := 0
			var importErrors []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if err := importProxy(line); err != nil {
					importErrors = append(importErrors, fmt.Sprintf("%s: %v", line, err))
				} else {
					importedCount++
				}
			}
			// Trigger rotator reload to immediately recognize the new proxies
			if s.jobManager.Rotator != nil {
				_ = s.jobManager.Rotator.Reload()
			}
			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"imported": importedCount,
				"errors":   importErrors,
			})
			return
		}

		if req.URL == "" {
			s.respondError(w, http.StatusBadRequest, "Missing url or bulk parameter")
			return
		}

		if err := importProxy(req.URL); err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Trigger rotator reload
		if s.jobManager.Rotator != nil {
			_ = s.jobManager.Rotator.Reload()
		}
		s.respondJSON(w, http.StatusCreated, map[string]string{"status": "created"})

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProxyActions handles:
// DELETE /api/proxies/:id
// POST /api/proxies/:id (toggle status)
func (s *APIServer) handleProxyActions(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/proxies/")
	if id == "" {
		s.respondError(w, http.StatusBadRequest, "Missing proxy ID")
		return
	}

	if r.Method == "DELETE" {
		_, err := s.db.Exec("DELETE FROM proxies WHERE id = ?", id)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if s.jobManager.Rotator != nil {
			_ = s.jobManager.Rotator.Reload()
		}
		s.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	if r.Method == "POST" {
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && (req.Status == "active" || req.Status == "failed") {
			_, err = s.db.Exec("UPDATE proxies SET status = ? WHERE id = ?", req.Status, id)
			if err != nil {
				s.respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if s.jobManager.Rotator != nil {
				_ = s.jobManager.Rotator.Reload()
			}
			s.respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
			return
		}
	}

	s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// handleFlameProxies handles proxies data from Flame Proxies API Client
// GET /api/flame-proxies/balance
// GET /api/flame-proxies/packages
// POST /api/flame-proxies/orders
// POST /api/flame-proxies/packages/:id/add-data
func (s *APIServer) handleFlameProxies(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("FLAME_PROXIES_API_KEY")
	if apiKey == "" {
		apiKey = "fp_live_g_3SJyC1IIXLs8wvTDQPzhNQ2mCFHFjypSO9_OLHELo"
	}

	client := proxy.NewFlameClient(apiKey)
	path := strings.TrimPrefix(r.URL.Path, "/api/flame-proxies/")

	if path == "balance" {
		if r.Method != "GET" {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		balance, err := client.GetBalance()
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, balance)
		return
	}

	if path == "packages" {
		if r.Method != "GET" {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		packages, err := client.GetPackages()
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, packages)
		return
	}

	if path == "orders" {
		if r.Method != "POST" {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		var req struct {
			Product  string `json:"product"` // residential, premium_residential
			GBAmount int    `json:"gb_amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.GBAmount <= 0 {
			s.respondError(w, http.StatusBadRequest, "gb_amount must be greater than 0")
			return
		}
		p, err := client.OrderPackage(req.Product, req.GBAmount)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, p)
		return
	}

	if strings.HasPrefix(path, "packages/") && strings.HasSuffix(path, "/add-data") {
		if r.Method != "POST" {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		pkgID := strings.TrimPrefix(path, "packages/")
		pkgID = strings.TrimSuffix(pkgID, "/add-data")

		var req struct {
			GBAmount int `json:"gb_amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.GBAmount <= 0 {
			s.respondError(w, http.StatusBadRequest, "gb_amount must be greater than 0")
			return
		}
		err := client.AddPackageData(pkgID, req.GBAmount)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]string{"status": "data_added"})
		return
	}

	s.respondError(w, http.StatusNotFound, "Not found")
}
