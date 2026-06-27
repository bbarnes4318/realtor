package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const BaseURL = "https://flameproxies.com"

type FlameClient struct {
	apiKey string
	client *http.Client
}

func NewFlameClient(apiKey string) *FlameClient {
	return &FlameClient{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// BalanceResponse represents the structure of the wallet balance API response
type BalanceResponse struct {
	WalletBalance float64 `json:"wallet_balance"`
	GBCredits     float64 `json:"gb_credits"`
	Credits       float64 `json:"credits"`
}

type FlamePackageUsage struct {
	BytesUsed      float64 `json:"bytes_used"`
	MaxBytes       float64 `json:"max_bytes"`
	RemainingBytes float64 `json:"remaining_bytes"`
	GBUsed         float64 `json:"gb_used"`
	GBTotal        float64 `json:"gb_total"`
	GBRemaining    float64 `json:"gb_remaining"`
	PercentUsed    float64 `json:"percent_used"`
}

// FlamePackage represents an owned proxy package
type FlamePackage struct {
	ID          interface{}        `json:"id"` // can be numeric or string
	Name        string             `json:"name"`
	Product     string             `json:"product"` // residential, premium_residential
	Plan        string             `json:"plan"`
	TrafficMax  float64            `json:"traffic_max"`
	TrafficUsed float64            `json:"traffic_used"`
	Status      string             `json:"status"`
	Username    string             `json:"username"`
	Password    string             `json:"password"`
	Usage       *FlamePackageUsage `json:"usage"`
}

func (c *FlameClient) doRequest(method, path string, body interface{}, responseTarget interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if responseTarget != nil {
		// Try to parse JSON response. To support flexible fields, we decode strictly or fall back.
		if err := json.Unmarshal(respBody, responseTarget); err != nil {
			return fmt.Errorf("failed to decode response JSON: %w, raw response: %s", err, string(respBody))
		}
	}

	return nil
}

// GetBalance retrieves the wallet and GB credits balance
func (c *FlameClient) GetBalance() (*BalanceResponse, error) {
	// API endpoint: /api/customer/balance
	var balance BalanceResponse
	// We use a custom parser or decode directly. Let's try direct.
	err := c.doRequest("GET", "/api/customer/balance", nil, &balance)
	if err != nil {
		// Fallback/Relaxed parsing if the structure differs
		var raw map[string]interface{}
		if errRaw := c.doRequest("GET", "/api/customer/balance", nil, &raw); errRaw == nil {
			if b, ok := raw["wallet_balance"].(float64); ok {
				balance.WalletBalance = b
			}
			if cr, ok := raw["gb_credits"].(float64); ok {
				balance.GBCredits = cr
			}
			return &balance, nil
		}
		return nil, err
	}
	return &balance, nil
}

// GetPackages retrieves active packages
func (c *FlameClient) GetPackages() ([]FlamePackage, error) {
	var allPackages []FlamePackage

	// 1. Try /api/customer/my-packages
	var wrapper struct {
		Packages []FlamePackage `json:"packages"`
	}
	err := c.doRequest("GET", "/api/customer/my-packages", nil, &wrapper)
	if err == nil {
		allPackages = append(allPackages, wrapper.Packages...)
	}

	// 2. Try /api/customer/non-api-packages/usage
	var fallbackWrapper struct {
		Packages []FlamePackage `json:"packages"`
	}
	errFallback := c.doRequest("GET", "/api/customer/non-api-packages/usage", nil, &fallbackWrapper)
	if errFallback == nil {
		allPackages = append(allPackages, fallbackWrapper.Packages...)
	}

	// If both failed, try direct array unmarshal as a last resort
	if err != nil && errFallback != nil {
		var directArray []FlamePackage
		if errDirect := c.doRequest("GET", "/api/customer/my-packages", nil, &directArray); errDirect == nil {
			allPackages = directArray
		} else {
			return nil, fmt.Errorf("failed to fetch packages from all endpoints: %w", err)
		}
	}

	// Post-process and deduplicate by ID
	seen := make(map[interface{}]bool)
	var deduped []FlamePackage

	for i := range allPackages {
		p := &allPackages[i]
		if p.ID == nil {
			continue
		}
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true

		if p.Product == "" && p.Plan != "" {
			p.Product = p.Plan
		}
		if p.Usage != nil {
			p.TrafficMax = p.Usage.GBTotal
			p.TrafficUsed = p.Usage.GBUsed
		}
		if p.Status == "" {
			p.Status = "active"
		}
		deduped = append(deduped, *p)
	}

	return deduped, nil
}

// OrderPackage orders a new package using wallet balance
func (c *FlameClient) OrderPackage(product string, gbAmount int) (*FlamePackage, error) {
	// API endpoint: /api/customer/orders
	payload := map[string]interface{}{
		"product": product, // "residential" or "premium_residential"
		"amount":   gbAmount,
	}
	var newPackage FlamePackage
	err := c.doRequest("POST", "/api/customer/orders", payload, &newPackage)
	if err != nil {
		return nil, err
	}
	return &newPackage, nil
}

// AddPackageData adds traffic to an existing package
func (c *FlameClient) AddPackageData(packageID interface{}, gbAmount int) error {
	// API endpoint: /api/customer/packages/:id/add-data
	path := fmt.Sprintf("/api/customer/packages/%v/add-data", packageID)
	payload := map[string]interface{}{
		"amount": gbAmount,
	}
	return c.doRequest("POST", path, payload, nil)
}
