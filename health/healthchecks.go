package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hellofresh/health-go/v5"
)

const (
	NoTimeout = time.Duration(24) * time.Hour
)

// HealthChecker is the interface that all health checks must implement.
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
	Type() string // e.g., "redis", "s3", "kafka", "http"
}

// Handler is a wrapper for health.Handler from "github.com/hellofresh/health-go/v5"
func Handler(h *health.Health) http.Handler {
	return h.Handler()
}

// New is a wrapper for health.New from "github.com/hellofresh/health-go/v5"
func New(options ...health.Option) (*health.Health, error) {
	return health.New(options...)
}

// ComponentConfig is the configuration for the health component.
type ComponentConfig struct {
	Name    string
	Version string
}

// NewComponent creates a new health.Component from ComponentConfig.
func NewComponent(config ComponentConfig) health.Component {
	return health.Component{
		Name:    config.Name,
		Version: config.Version,
	}
}

// HealthOptionsConfig allows configuring various aspects of the health checks.
type HealthOptionsConfig struct {
	ComponentConfig   ComponentConfig
	SystemInfoEnabled bool
}

// NewHealthOptions creates a slice of health.Option from HealthOptionsConfig.
func NewHealthOptions(config HealthOptionsConfig) []health.Option {
	opts := []health.Option{}

	if config.ComponentConfig.Name != "" {
		opts = append(opts, health.WithComponent(NewComponent(config.ComponentConfig)))
	}

	if config.SystemInfoEnabled {
		opts = append(opts, health.WithSystemInfo())
	}

	return opts
}

type CheckCallback func(checkName string, checkType string, result error)

type HealthCheckerWrapper struct {
	HealthChecker
	callback CheckCallback
}

func (w HealthCheckerWrapper) Check(ctx context.Context) error {
	result := w.HealthChecker.Check(ctx)
	if w.callback != nil {
		w.callback(w.Name(), w.Type(), result)
	}
	return result
}

func wrapCheckerForCallback(cb CheckCallback, checker HealthChecker) HealthChecker {
	if cb == nil {
		return checker
	}

	return HealthCheckerWrapper{HealthChecker: checker, callback: cb}
}

// SetupHealthChecks sets up and registers health checks with retries, returning the health.Health instance.
//
// If provided the checkCallback will be invoked with the result of each checker
func SetupHealthChecks(
	componentName, componentVersion string,
	enableSystemInfo bool,
	checkCallback CheckCallback,
	checkers ...HealthChecker,
) (*health.Health, error) {
	maxRetries := 3
	retryInterval := 2 * time.Second

	var h *health.Health
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		h, err = setupHealthChecks(componentName, componentVersion, enableSystemInfo, checkCallback, checkers...)
		if err == nil {
			return h, nil
		}

		fmt.Printf("Attempt %d/%d: Failed to set up health checks: %v", attempt+1, maxRetries+1, err)
		if attempt < maxRetries {
			fmt.Printf("Retrying in %v...", retryInterval)
			time.Sleep(retryInterval)
		}
	}

	return nil, fmt.Errorf("failed to set up health checks after %d attempts: %w", maxRetries+1, err)
}

func setupHealthChecks(componentName, componentVersion string, enableSystemInfo bool, checkCallback CheckCallback, checkers ...HealthChecker) (*health.Health, error) {
	h, err := health.New(
		health.WithComponent(health.Component{
			Name:    componentName,
			Version: componentVersion,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create health instance: %w", err)
	}

	if enableSystemInfo {
		if err := health.WithSystemInfo()(h); err != nil {
			return nil, fmt.Errorf("failed to add system info: %w", err)
		}
	}

	for _, checker := range checkers {
		checker = wrapCheckerForCallback(checkCallback, checker)
		checkConfig := health.Config{
			Name:    checker.Name(),
			Check:   checker.Check,
			Timeout: NoTimeout,
		}
		if err := h.Register(checkConfig); err != nil {
			return nil, fmt.Errorf("failed to register check %q: %w", checker.Name(), err)
		}
	}

	return h, nil
}

// StartHealthCheckServer starts the health check server.
func StartHealthCheckServer(h *health.Health, port string) {
	go func() {
		healthHandler := Handler(h)
		http.Handle("/healthz", healthHandler)
		addr := port
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			fmt.Printf("problem with health check server on %s: %v\n", addr, err)
		}
	}()
}
