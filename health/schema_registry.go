package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SchemaRegistryChecker verifies the availability of a Schema Registry.
type SchemaRegistryChecker struct {
	CheckerName string
	URL         string
	HTTPClient  HttpClient
}

// Check verifies Schema Registry is healthy by querying /subjects.
func (c *SchemaRegistryChecker) Check(ctx context.Context) error {
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/subjects", c.URL), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach schema registry: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("schema registry unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// Name returns the unique name of the schema registry health check.
func (c *SchemaRegistryChecker) Name() string {
	return c.CheckerName
}

// Type returns the type identifier for the health check.
func (c *SchemaRegistryChecker) Type() string {
	return "schema-registry"
}
