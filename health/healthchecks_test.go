package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hellofresh/health-go/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHealthChecker is a mock implementation of the HealthChecker interface.
type MockHealthChecker struct {
	mock.Mock
}

func (m *MockHealthChecker) Check(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHealthChecker) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHealthChecker) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHealthChecker) GetTimeOut() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func TestSetupHealthChecks_Success(t *testing.T) {
	mockChecker := new(MockHealthChecker)
	mockChecker.On("Check", mock.Anything).Return(nil)
	mockChecker.On("Name").Return("mock-check")
	mockChecker.On("GetTimeOut").Return(5 * time.Second)

	h, err := SetupHealthChecks("test-service", "v1.0.0", false, mockChecker)

	assert.NoError(t, err)
	assert.NotNil(t, h)

	// Test if the check is invoked
	req, _ := http.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	h.Handler().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	mockChecker.AssertExpectations(t)
}

func TestNewComponent(t *testing.T) {
	config := ComponentConfig{
		Name:    "test-component",
		Version: "v1.0.0",
	}

	component := NewComponent(config)
	assert.Equal(t, "test-component", component.Name)
	assert.Equal(t, "v1.0.0", component.Version)
}

func TestNewHealthOptions(t *testing.T) {
	config := HealthOptionsConfig{
		ComponentConfig: ComponentConfig{
			Name:    "test-component",
			Version: "v1.0.0",
		},
		SystemInfoEnabled: true,
	}

	options := NewHealthOptions(config)
	assert.NotEmpty(t, options)
}

func TestHandler(t *testing.T) {
	h, _ := health.New()
	handler := Handler(h)
	assert.Implements(t, (*http.Handler)(nil), handler)

	req, _ := http.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

}
