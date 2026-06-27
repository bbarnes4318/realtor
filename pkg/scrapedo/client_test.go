package scrapedo

import (
	"os"
	"strings"
	"testing"
)

func TestBuildURL_Validation(t *testing.T) {
	// Backup env
	oldEnabled := os.Getenv("SCRAPEDO_ENABLED")
	oldToken := os.Getenv("SCRAPEDO_TOKEN")
	defer func() {
		os.Setenv("SCRAPEDO_ENABLED", oldEnabled)
		os.Setenv("SCRAPEDO_TOKEN", oldToken)
	}()

	target := "https://www.realtor.com/realestateagents/api/v3/search"

	// 1. Scrape.do disabled
	os.Setenv("SCRAPEDO_ENABLED", "false")
	os.Setenv("SCRAPEDO_TOKEN", "")
	res, err := BuildURL(target)
	if err != nil {
		t.Fatalf("Expected no error when disabled, got: %v", err)
	}
	if res != target {
		t.Errorf("Expected URL to be unchanged, got: %s", res)
	}

	// 2. Scrape.do enabled, token missing
	os.Setenv("SCRAPEDO_ENABLED", "true")
	os.Setenv("SCRAPEDO_TOKEN", "")
	_, err = BuildURL(target)
	if err == nil {
		t.Fatal("Expected error when Scrape.do is enabled but token is missing, got nil")
	}
	expectedErr := "SCRAPEDO_TOKEN is required when Scrape.do is enabled"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error to contain %q, got: %v", expectedErr, err)
	}

	// 3. Scrape.do enabled, token present
	os.Setenv("SCRAPEDO_ENABLED", "true")
	os.Setenv("SCRAPEDO_TOKEN", "test-token-123")
	res, err = BuildURL(target)
	if err != nil {
		t.Fatalf("Expected no error when token is present, got: %v", err)
	}
	if !strings.HasPrefix(res, endpoint) {
		t.Errorf("Expected URL to start with %s, got %s", endpoint, res)
	}
	if !strings.Contains(res, "token=test-token-123") {
		t.Errorf("Expected URL to contain token, got %s", res)
	}
}
