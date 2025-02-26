package health

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHttpClient is a mock for the http.Client
type MockHttpClient struct {
	mock.Mock
}

// Do is the mocked method for http.Client.Do
func (m *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNewKSQLDBAsyncChecker(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		checker, err := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test-stream")
		assert.NoError(t, err)
		assert.NotNil(t, checker)
		assert.Equal(t, defaultKSQLDBName, checker.config.Name)
		assert.Equal(t, "http://localhost:8088", checker.config.Endpoint)
		assert.Equal(t, 1*time.Second, checker.config.Timeout)
		assert.Equal(t, 5*time.Second, checker.config.Interval)
		assert.Equal(t, "test-stream", checker.config.Stream)

	})

	t.Run("Error - Empty Endpoint", func(t *testing.T) {
		_, err := NewKSQLDBAsyncChecker("", 1*time.Second, 5*time.Second, "test-stream")
		assert.Error(t, err)
		assert.EqualError(t, err, "KSQLDB endpoint is required")
	})
}

func TestKSQLDBAsyncChecker_ksqldbCheck(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Create a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/query-stream", r.URL.Path)
			assert.Equal(t, "application/vnd.ksql.v1+json", r.Header.Get("Content-Type"))

			// Send a successful response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		checker, _ := NewKSQLDBAsyncChecker(server.URL, 1*time.Second, 5*time.Second, "test_stream")

		err := checker.ksqldbCheck(context.Background())
		assert.NoError(t, err)
	})

	t.Run("Request Creation Error", func(t *testing.T) {
		checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test_stream")
		checker.config.Endpoint = ":" // Invalid URL to cause an error
		err := checker.ksqldbCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create request")

	})

	t.Run("HTTP Request Failure", func(t *testing.T) {
		mockClient := new(MockHttpClient)
		mockResponse := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}

		// Expectation: The Do method will be called *once* and return an error
		mockClient.On("Do", mock.Anything).Return(mockResponse, errors.New("simulated network error")).Once()

		checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test_stream")
		checker.httpClient = mockClient

		err := checker.ksqldbCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated network error")
		mockClient.AssertExpectations(t)

	})

	t.Run("Non-200 Status Code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("KSQLDB Error"))
		}))
		defer server.Close()

		checker, _ := NewKSQLDBAsyncChecker(server.URL, 1*time.Second, 5*time.Second, "test_stream")

		err := checker.ksqldbCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "KSQLDB request failed with status 500: KSQLDB Error")
	})

	t.Run("Context Timeout", func(t *testing.T) {
		// Mock HTTP server that *always* times out
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)  // Simulate a delay *longer* than the checker's timeout
			w.WriteHeader(http.StatusOK) // This line will *never* be reached
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		checker, _ := NewKSQLDBAsyncChecker(server.URL, 1*time.Second, 100*time.Millisecond, "test-stream")

		err := checker.ksqldbCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Successful Response with body", func(t *testing.T) {
		// Mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key": "value"}`))
		}))
		defer server.Close()

		checker, _ := NewKSQLDBAsyncChecker(server.URL, 1*time.Second, 5*time.Second, "test-stream")
		err := checker.ksqldbCheck(context.Background())
		assert.NoError(t, err)

	})
}
func TestKSQLDBAsyncChecker_Start(t *testing.T) {
	t.Run("Context Cancellation", func(t *testing.T) {
		checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 50*time.Millisecond, "test_stream")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Immediately cancel the context

		done := make(chan struct{})
		go func() {
			checker.Start(ctx)
			close(done)
		}()

		select {
		case <-done:
			return
		case <-time.After(1 * time.Second):
			t.Fatal("Start did not exit after context cancellation")
		}
	})

	t.Run("Check Execution", func(t *testing.T) {
		mockClient := new(MockHttpClient)
		mockResponse := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}

		// The Do method will get called multiple times due to the ticker
		mockClient.On("Do", mock.Anything).Return(mockResponse, nil).Maybe()

		checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 50*time.Millisecond, 100*time.Millisecond, "test_stream") // Very short interval
		checker.httpClient = mockClient

		ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond) // Enough time for a few ticks
		defer cancel()

		checker.Start(ctx)
		<-ctx.Done()

		mockClient.AssertExpectations(t)

		assert.NoError(t, checker.store.GetStatus(checker.config.Name))
	})
}

func TestKSQLDBAsyncChecker_Check(t *testing.T) {

	t.Run("Check Returns Stored Error", func(t *testing.T) {
		checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test_stream")
		expectedError := fmt.Errorf("simulated error")
		checker.store.UpdateStatus("ksqldb", expectedError)

		err := checker.Check(context.Background())
		assert.Equal(t, expectedError, err)
	})

	t.Run("Check Returns No Error", func(t *testing.T) {
		checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test_stream")
		checker.store.UpdateStatus("ksqldb", nil)

		err := checker.Check(context.Background())
		assert.NoError(t, err)
	})
}

func TestKSQLDBAsyncChecker_Name(t *testing.T) {
	checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test-stream")
	assert.Equal(t, defaultKSQLDBName, checker.Name())
}

func TestKSQLDBAsyncChecker_GetTimeOut(t *testing.T) {
	checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test-stream")
	assert.Equal(t, 5*time.Second, checker.GetTimeOut())
}

func TestKSQLDBAsyncChecker_Type(t *testing.T) {
	checker, _ := NewKSQLDBAsyncChecker("http://localhost:8088", 1*time.Second, 5*time.Second, "test-stream")
	assert.Equal(t, defaultKSQLDBName, checker.Type())
}
