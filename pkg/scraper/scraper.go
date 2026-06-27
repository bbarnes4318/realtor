package scraper

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v4"
	"github.com/suffer-sami/realtor-scraper/internal/database"
	"github.com/suffer-sami/realtor-scraper/internal/logger"
)

type Scraper struct {
	client               *http.Client
	db                   *sql.DB
	dbQueries            *database.Queries
	saveRawAgents        bool
	logger               logger.Logger
	jwtSecret            string
	mu                   *sync.Mutex
	jobID                string
}

func NewScraper(db *sql.DB, client *http.Client, jwtSecret string, saveRawAgents bool, logger logger.Logger, jobID string) *Scraper {
	return &Scraper{
		client:        client,
		db:            db,
		dbQueries:     database.New(db),
		jwtSecret:     jwtSecret,
		saveRawAgents: saveRawAgents,
		logger:        logger,
		mu:            &sync.Mutex{},
		jobID:         jobID,
	}
}

const (
	baseUrl               = "https://www.realtor.com"
	apiEndpoint           = baseUrl + "/realestateagents/api/v3/search"
	defaultResultsPerPage = 20
	tokenTTL              = 1 * time.Minute
)

type JobFilters struct {
	State      string `json:"state"`
	City       string `json:"city"`
	Zip        string `json:"zip"`
	Brokerage  string `json:"brokerage"`
	AgentName  string `json:"agent_name"`
	AreaServed string `json:"area_served"`
}

func (s *Scraper) getRequestParams(offset, resultsPerPage int, filters *JobFilters) SearchRequestParams {
	params := SearchRequestParams{
		Offset:              offset,
		Limit:               resultsPerPage,
		MarketingAreaCities: "_",
		Types:               "agent",
		Sort:                "agent_rating_high",
		FarOptOut:           "false",
		ClientID:            "FAR2.0",
		SeoUserType:         SeoUserType{IsBot: "false", DeviceType: "desktop"},
		IsCountySearch:      "false",
	}

	if filters != nil {
		if filters.Zip != "" {
			params.PostalCode = filters.Zip
			params.IsPostalSearch = "true"
		}
		if filters.AgentName != "" {
			params.Name = filters.AgentName
		}
	}
	return params
}

// GetTotalResults retrieves the total number of matching rows.
func (s *Scraper) GetTotalResults(ctx context.Context, filters *JobFilters) (int, error) {
	payload := s.getRequestParams(0, 0, filters)
	response, err := s.GetSearchResultsWithRetry(ctx, payload)
	if err != nil {
		return 0, fmt.Errorf("error getting total results: %w", err)
	}
	return response.MatchingRows, nil
}

// GetAgents retrieves a list of normalized agents matching the search criteria.
func (s *Scraper) GetAgents(ctx context.Context, offset, resultsPerPage int, filters *JobFilters) ([]Agent, error) {
	payload := s.getRequestParams(offset, resultsPerPage, filters)
	response, err := s.GetSearchResultsWithRetry(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("error getting agents: %w", err)
	}

	for i := range response.Agents {
		s.normalizeAgent(&response.Agents[i])
	}

	return response.Agents, nil
}

// GetSearchResultsWithRetry executes HTTP requests with retries and exponential backoff.
func (s *Scraper) GetSearchResultsWithRetry(ctx context.Context, payload SearchRequestParams) (SearchRequestResponse, error) {
	var response SearchRequestResponse
	var err error
	backoff := 1 * time.Second
	maxRetries := 3

	for i := 0; i <= maxRetries; i++ {
		if ctx.Err() != nil {
			return SearchRequestResponse{}, ctx.Err()
		}

		response, err = s.getSearchResults(payload)
		if err == nil {
			return response, nil
		}

		s.logger.Warnf("Realtor API request failed (attempt %d/%d): %v", i+1, maxRetries+1, err)
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return SearchRequestResponse{}, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}

	return SearchRequestResponse{}, fmt.Errorf("after %d retries, request failed: %w", maxRetries, err)
}

// getSearchResults fetches search results from the API.
func (s *Scraper) getSearchResults(payload SearchRequestParams) (SearchRequestResponse, error) {
	parsedURL, _ := url.Parse(apiEndpoint)
	queryParams, err := buildQueryParams(payload)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to build query params: %w", err)
	}
	parsedURL.RawQuery = queryParams.Encode()

	token, err := generateBearerToken(s.jwtSecret)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to generate token: %w", err)
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	userAgent, err := getRandomUserAgent()
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to get random user agent: %w", err)
	}

	// Fetch home page to populate the client's cookie jar with Kasada credentials
	reqHome, err := http.NewRequest("GET", baseUrl+"/", nil)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to create home preflight request: %w", err)
	}
	reqHome.Header.Set("User-Agent", userAgent)
	reqHome.Header.Set("sec-ch-ua", `"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`)
	reqHome.Header.Set("sec-ch-ua-mobile", "?0")
	reqHome.Header.Set("sec-ch-ua-platform", `"Windows"`)
	reqHome.Header.Set("accept-language", "en-US,en;q=0.9")

	respHome, err := s.client.Do(reqHome)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("home preflight request failed: %w", err)
	}
	bodyHome, _ := io.ReadAll(respHome.Body)
	respHome.Body.Close()
	s.logger.Infof("Preflight Status: %d. Cookies: %+v. Body length: %d", respHome.StatusCode, respHome.Header["Set-Cookie"], len(bodyHome))

	setHeaders(req, token, userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Errorf("Failed request status: %d. Headers: %+v. Body: %s", resp.StatusCode, resp.Header, string(body))
		return SearchRequestResponse{}, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	var response SearchRequestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return SearchRequestResponse{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return response, nil
}

// buildQueryParams converts the search payload into query parameters.
func buildQueryParams(payload SearchRequestParams) (url.Values, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload to JSON: %w", err)
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payloadMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling payload JSON: %w", err)
	}

	queryParams := url.Values{}
	for key, value := range payloadMap {
		queryParams.Add(key, fmt.Sprintf("%v", value))
	}

	return queryParams, nil
}

// setHeaders sets headers for the HTTP request.
func setHeaders(req *http.Request, token string, userAgent string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseUrl)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("sec-ch-ua", `"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
}

// generateBearerToken creates a signed JWT token.
func generateBearerToken(secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": "find_a_realtor",
		"exp": jwt.NewNumericDate(time.Now().UTC().Add(tokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// getRandomUserAgent gives a random useragent for rotating useragent.
func getRandomUserAgent() (string, error) {
	// Enforce Windows Chrome 133 to match the TLS fingerprint and sec-ch-ua headers exactly
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36", nil
}

// StoreAgent calls the internal database saver.
func (s *Scraper) StoreAgent(agent Agent) error {
	return s.storeAgent(agent)
}
