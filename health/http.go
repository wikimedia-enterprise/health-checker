package health

import (
	"context"
	"fmt"
	"net/http"
)

// HTTPCheckerConfig holds configuration for the HTTPChecker.
type HTTPCheckerConfig struct {
	URL            string
	Name           string
	ExpectedStatus int
}

// HTTPChecker implements the HealthChecker interface for HTTP endpoints.
type HTTPChecker struct {
	config HTTPCheckerConfig
}

// NewHTTPChecker creates a new HTTPChecker.
func NewHTTPChecker(config HTTPCheckerConfig) *HTTPChecker {
	if config.ExpectedStatus == 0 {
		config.ExpectedStatus = http.StatusOK
	}

	return &HTTPChecker{config: config}
}

// Check performs the HTTP health check.
func (c *HTTPChecker) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request failed: %w", err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if c.config.ExpectedStatus != 0 && resp.StatusCode != c.config.ExpectedStatus {
		return fmt.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, c.config.ExpectedStatus)
	}

	return nil
}

// Name returns the name of the health check.
func (c *HTTPChecker) Name() string {
	return c.config.Name
}

// Type returns the type of the health check (http).
func (c *HTTPChecker) Type() string {
	return "http"
}
