package health

import (
	"context"
	"errors"
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
	if m.doFunc != nil {
		return m.doFunc(req)
	}

	return nil, errors.New("doFunc not implemented")
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
		CheckerName: "schema-registry-ok",
		URL:         "http://fake-schema-registry:8081",
		HTTPClient:  client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := checker.Check(ctx); err != nil {
		t.Fatalf("expected success, but got error: %v", err)
	}
}

func TestSchemaRegistryChecker_Check_Failure_StatusCode(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error_code":50001,"message":"Error in the backend"}`)),
			}, nil
		},
	}

	checker := &SchemaRegistryChecker{
		CheckerName: "schema-registry-bad-status",
		URL:         "http://fake-schema-registry:8081",
		HTTPClient:  client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err == nil {
		t.Fatal("expected an error due to non-200 status code, but got nil")
	}

	expectedErrorSubstring := "error unable to query /subjects: status 500"
	if !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("expected error message to contain '%s', but got: '%v'", expectedErrorSubstring, err)
	}
}

func TestSchemaRegistryChecker_Check_Failure_ClientDoError(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	}

	checker := &SchemaRegistryChecker{
		CheckerName: "schema-registry-client-error",
		URL:         "http://unreachable-schema-registry:8081",
		HTTPClient:  client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err == nil {
		t.Fatal("expected an error due to request failure, but got nil")
	}

	expectedErrorSubstring := "error /subjects not reachable"
	if !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("expected error message to contain '%s', but got: '%v'", expectedErrorSubstring, err)
	}
}

func TestSchemaRegistryChecker_Check_Failure_NewRequestError(t *testing.T) {
	checker := &SchemaRegistryChecker{
		CheckerName: "schema-registry-bad-url",
		URL:         "http://invalid-url:\n",
		HTTPClient:  &mockHTTPClient{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err == nil {
		t.Fatal("expected an error due to invalid request creation, but got nil")
	}

	expectedErrorSubstring := "error creating schema request"
	if !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("expected error message to contain '%s', but got: '%v'", expectedErrorSubstring, err)
	}
}
