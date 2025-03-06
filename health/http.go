package health

import (
	"context"
	"fmt"
	"net/http"
	"slices"
)

// HTTPCheckerConfig holds configuration for the HTTPChecker.
type HTTPCheckerConfig struct {
	URL              string
	Name             string
	ExpectedStatuses []int
}

// HTTPChecker implements the HealthChecker interface for HTTP endpoints.
type HTTPChecker struct {
	config HTTPCheckerConfig
}

// NewHTTPChecker creates a new HTTPChecker.
func NewHTTPChecker(config HTTPCheckerConfig) *HTTPChecker {
	if len(config.ExpectedStatuses) == 0 {
		config.ExpectedStatuses = []int{http.StatusOK}
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

	if !slices.Contains(c.config.ExpectedStatuses, resp.StatusCode) {
		return fmt.Errorf("unexpected status code: got %d, want %v", resp.StatusCode, c.config.ExpectedStatuses)
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
