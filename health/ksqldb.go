package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// HttpClient interface defines the method we need from an HTTP client.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// KSQLDBCheckerConfig holds configuration for the KSQLDBChecker.
type KSQLDBCheckerConfig struct {
	Name     string
	Endpoint string        // KSQLDB endpoint
	Interval time.Duration // How often to run the async check
	Timeout  time.Duration // Timeout *for the HTTP request*
	Stream   string
}

// KSQLDBAsyncChecker implements the HealthChecker interface for asynchronous KSQLDB checks.
type KSQLDBAsyncChecker struct {
	store      *AsyncHealthStore
	config     KSQLDBCheckerConfig
	httpClient HttpClient
	mu         sync.RWMutex
}

// Constants for default values
const (
	defaultKSQLDBName = "ksqldb"
)

// NewKSQLDBAsyncChecker creates a new KSQLDBAsyncChecker.
func NewKSQLDBAsyncChecker(KSQLURL string, dkt time.Duration, dki time.Duration, stn string) (*KSQLDBAsyncChecker, error) {

	if KSQLURL == "" {
		return nil, fmt.Errorf("KSQLDB endpoint is required")
	}

	config := KSQLDBCheckerConfig{
		Name:     defaultKSQLDBName,
		Endpoint: KSQLURL,
		Interval: dki,
		Timeout:  dkt,
		Stream:   stn,
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	store := NewAsyncHealthStore()

	checker := &KSQLDBAsyncChecker{
		store:      store,
		config:     config,
		httpClient: client,
		mu:         sync.RWMutex{},
	}

	return checker, nil
}

// ksqldbCheck is the actual asynchronous KSQLDB check function.
func (kac *KSQLDBAsyncChecker) ksqldbCheck(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, kac.config.Timeout)
	defer cancel()

	query := fmt.Sprintf("SELECT * FROM %s EMIT CHANGES LIMIT 1;", kac.config.Stream)

	properties := map[string]string{
		"ksql.streams.auto.offset.reset":     "latest",
		"ksql.query.pull.table.scan.enabled": "true",
	}

	qrq := map[string]interface{}{
		"sql":               query,
		"streamsProperties": properties,
	}
	requestBody, err := json.Marshal(qrq)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(checkCtx, http.MethodPost, kac.config.Endpoint+"/query-stream", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.ksql.v1+json")
	req.Header.Set("Accept", "application/vnd.ksql.v1+json")

	resp, err := kac.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("make KSQLDB request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("KSQLDB request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Start starts the asynchronous health check loop.
func (kac *KSQLDBAsyncChecker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(kac.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Print("KSQLDB async check shutting down")
				return
			case <-ticker.C:
				err := kac.ksqldbCheck(ctx)
				kac.store.UpdateStatus(kac.config.Name, err)
			}
		}
	}()
}

// Check implements the health.Checker interface by fetching the status from the store.
func (kac *KSQLDBAsyncChecker) Check(ctx context.Context) error {
	kac.mu.RLock()
	defer kac.mu.RUnlock()
	return kac.store.GetStatus(kac.config.Name)
}

// Name returns the component name.
func (kac *KSQLDBAsyncChecker) Name() string {
	return kac.config.Name
}

// GetTimeOut returns the timeout
func (kac *KSQLDBAsyncChecker) GetTimeOut() time.Duration {
	return kac.config.Timeout
}

// Type returns the component type.
func (kac *KSQLDBAsyncChecker) Type() string {
	return defaultKSQLDBName
}
