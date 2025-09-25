package health

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockHTTPClient lets us stub HTTP responses for tests.
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestSchemaRegistryChecker_Check_Success(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`["subject1","subject2"]`)),
			}, nil
		},
	}

	checker := &SchemaRegistryChecker{
		CheckerName: "schema-registry",
		URL:         "http://fake-schema-registry",
		HTTPClient:  client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := checker.Check(ctx); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestSchemaRegistryChecker_Check_Failure_StatusCode(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`server error`)),
			}, nil
		},
	}

	checker := &SchemaRegistryChecker{
		CheckerName: "schema-registry",
		URL:         "http://fake-schema-registry",
		HTTPClient:  client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err == nil {
		t.Fatal("expected error due to non-200 status code, got nil")
	}
	if !strings.Contains(err.Error(), "unhealthy") {
		t.Errorf("expected error to mention 'unhealthy', got: %v", err)
	}
}

func TestSchemaRegistryChecker_Check_Failure_RequestError(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		},
	}

	checker := &SchemaRegistryChecker{
		CheckerName: "schema-registry",
		URL:         "http://fake-schema-registry",
		HTTPClient:  client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err == nil {
		t.Fatal("expected error due to request failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to reach") {
		t.Errorf("expected error to mention 'failed to reach', got: %v", err)
	}
}
