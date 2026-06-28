package scrapedo

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

const endpoint = "https://api.scrape.do/"

type Config struct {
	Enabled             bool
	Token               string
	Super               bool
	GeoCode             string
	RegionalGeoCode     string
	SessionID           string
	CustomHeaders       bool
	ForwardHeaders      bool
	ExtraHeaders        bool
	Render              bool
	Timeout             int
	RetryTimeout        int
	DisableRetry        bool
	Device              string
	WaitUntil           string
	CustomWait          int
	WaitSelector        string
	BlockResources      bool
	Output              string
	TransparentResponse bool
}

func FromEnv() Config {
	token := os.Getenv("SCRAPEDO_TOKEN")
	enabledDefault := token != ""

	return Config{
		Enabled:             envBool("SCRAPEDO_ENABLED", enabledDefault),
		Token:               token,
		Super:               envBoolAlias("SCRAPEDO_SUPER", "SCRAPEDO_USE_SUPER", true),
		GeoCode:             envString("SCRAPEDO_GEO_CODE", envString("SCRAPEDO_DEFAULT_GEO", "US")),
		RegionalGeoCode:     os.Getenv("SCRAPEDO_REGIONAL_GEO_CODE"),
		SessionID:           os.Getenv("SCRAPEDO_SESSION_ID"),
		CustomHeaders:       envBool("SCRAPEDO_CUSTOM_HEADERS", true),
		ForwardHeaders:      envBool("SCRAPEDO_FORWARD_HEADERS", true),
		ExtraHeaders:        envBool("SCRAPEDO_EXTRA_HEADERS", false),
		Render:              envBool("SCRAPEDO_RENDER", false),
		Timeout:             envInt("SCRAPEDO_TIMEOUT", 60000),
		RetryTimeout:        envInt("SCRAPEDO_RETRY_TIMEOUT", 15000),
		DisableRetry:        envBool("SCRAPEDO_DISABLE_RETRY", false),
		Device:              envString("SCRAPEDO_DEVICE", "desktop"),
		WaitUntil:           envString("SCRAPEDO_WAIT_UNTIL", "domcontentloaded"),
		CustomWait:          envInt("SCRAPEDO_CUSTOM_WAIT", 0),
		WaitSelector:        os.Getenv("SCRAPEDO_WAIT_SELECTOR"),
		BlockResources:      envBool("SCRAPEDO_BLOCK_RESOURCES", true),
		Output:              envString("SCRAPEDO_OUTPUT", "raw"),
		TransparentResponse: envBool("SCRAPEDO_TRANSPARENT_RESPONSE", false),
	}
}

func Enabled() bool {
	return FromEnv().Enabled
}

func BuildURL(targetURL string) (string, error) {
	cfg := FromEnv()
	return BuildURLWithConfig(targetURL, cfg)
}

func BuildURLWithConfig(targetURL string, cfg Config) (string, error) {
	if !cfg.Enabled {
		return targetURL, nil
	}

	parsedTarget, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("invalid target URL for Scrape.do: %w", err)
	}

	if parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https" {
		return "", fmt.Errorf("Scrape.do target URL must use http or https")
	}

	if cfg.Token == "" {
		return "", fmt.Errorf("SCRAPEDO_TOKEN is required when Scrape.do is enabled")
	}

	params := url.Values{}
	params.Set("token", cfg.Token)
	params.Set("url", targetURL)
	params.Set("super", boolString(cfg.Super))
	if cfg.CustomHeaders {
		params.Set("customHeaders", "true")
	} else if cfg.ForwardHeaders {
		params.Set("forwardHeaders", "true")
	}
	params.Set("extraHeaders", boolString(cfg.ExtraHeaders))
	if cfg.Render {
		params.Set("render", "true")
		if cfg.BlockResources {
			params.Set("blockResources", "true")
		}
	}
	params.Set("timeout", strconv.Itoa(cfg.Timeout))
	params.Set("retryTimeout", strconv.Itoa(cfg.RetryTimeout))
	params.Set("disableRetry", boolString(cfg.DisableRetry))
	params.Set("device", cfg.Device)
	params.Set("output", cfg.Output)
	params.Set("transparentResponse", boolString(cfg.TransparentResponse))

	if cfg.GeoCode != "" {
		params.Set("geoCode", cfg.GeoCode)
	}

	if cfg.RegionalGeoCode != "" {
		params.Set("regionalGeoCode", cfg.RegionalGeoCode)
	}

	if cfg.SessionID != "" {
		params.Set("sessionId", cfg.SessionID)
	}

	if cfg.Render && cfg.WaitUntil != "" {
		params.Set("waitUntil", cfg.WaitUntil)
	}

	if cfg.Render && cfg.CustomWait > 0 {
		params.Set("customWait", strconv.Itoa(cfg.CustomWait))
	}

	if cfg.Render && cfg.WaitSelector != "" {
		params.Set("waitSelector", cfg.WaitSelector)
	}

	return endpoint + "?" + params.Encode(), nil
}

func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envBoolAlias(primary string, alias string, fallback bool) bool {
	if os.Getenv(primary) != "" {
		return envBool(primary, fallback)
	}

	return envBool(alias, fallback)
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func boolString(value bool) string {
	if value {
		return "true"
	}

	return "false"
}
