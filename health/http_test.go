package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPChecker_Check_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPCheckerConfig{
		URL:            server.URL,
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)

	err := checker.Check(context.Background())
	assert.NoError(t, err)
}

func TestHTTPChecker_Check_RequestCreationError(t *testing.T) {
	config := HTTPCheckerConfig{
		URL:            "://invalid-url",
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestHTTPChecker_Check_UnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//intentionally return status code other than 200
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := HTTPCheckerConfig{
		URL:            server.URL,
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestHTTPChecker_NewHTTPChecker_DefaultExpectedStatus(t *testing.T) {
	config := HTTPCheckerConfig{
		URL:  "http://example.com",
		Name: "test-http-check",
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, http.StatusOK, checker.config.ExpectedStatus) // default status is 200 OK
}

func TestHTTPChecker_Name(t *testing.T) {
	config := HTTPCheckerConfig{
		URL:  "http://example.com",
		Name: "test-http-check",
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, "test-http-check", checker.Name())
}

func TestHTTPChecker_Type(t *testing.T) {
	config := HTTPCheckerConfig{
		URL:  "http://example.com",
		Name: "test-http-check",
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, "http", checker.Type())
}

func TestHTTPChecker_Check_ContextCancelled_BeforeRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := HTTPCheckerConfig{
		URL:            "http://example.com",
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)
	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "making request")
	assert.ErrorIs(t, err, context.Canceled)
}
