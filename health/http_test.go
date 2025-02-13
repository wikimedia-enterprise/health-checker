package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPChecker_Check_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPCheckerConfig{
		URL:            server.URL,
		Timeout:        1 * time.Second,
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
		Timeout:        1 * time.Second,
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestHTTPChecker_Check_RequestExecutionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// add delay for timeout check
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPCheckerConfig{
		URL:            server.URL,
		Timeout:        100 * time.Millisecond,
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "making request")

	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout(), "Expected a timeout error")
	} else {
		assert.Fail(t, "Expected a net.Error")
	}
}

func TestHTTPChecker_Check_UnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//intentionally return status code other than 200
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := HTTPCheckerConfig{
		URL:            server.URL,
		Timeout:        1 * time.Second,
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
		URL:     "http://example.com",
		Name:    "test-http-check",
		Timeout: 1 * time.Second,
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, http.StatusOK, checker.config.ExpectedStatus) // default status is 200 OK
}

func TestHTTPChecker_GetTimeout(t *testing.T) {
	expectedTimeout := 10 * time.Second
	config := HTTPCheckerConfig{
		URL:     "http://example.com",
		Name:    "test-http-check",
		Timeout: expectedTimeout,
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, expectedTimeout, checker.GetTimeOut())
}

func TestHTTPChecker_Name(t *testing.T) {
	config := HTTPCheckerConfig{
		URL:     "http://example.com",
		Name:    "test-http-check",
		Timeout: 1 * time.Second,
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, "test-http-check", checker.Name())
}

func TestHTTPChecker_Type(t *testing.T) {
	config := HTTPCheckerConfig{
		URL:     "http://example.com",
		Name:    "test-http-check",
		Timeout: 1 * time.Second,
	}

	checker := NewHTTPChecker(config)
	assert.Equal(t, "http", checker.Type())
}

func TestHTTPChecker_Check_ContextCancelled_BeforeRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := HTTPCheckerConfig{
		URL:            "http://example.com",
		Timeout:        5 * time.Second,
		Name:           "test-http-check",
		ExpectedStatus: http.StatusOK,
	}
	checker := NewHTTPChecker(config)
	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "making request")
	assert.ErrorIs(t, err, context.Canceled)
}
