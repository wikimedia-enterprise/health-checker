package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hellofresh/health-go/v5"
)

// HealthChecker is the interface that all health checks must implement.
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
	Type() string // e.g., "redis", "s3", "kafka", "http"
	GetTimeOut() time.Duration
}

// AsyncHealthStore stores the results of asynchronous health checks.
type AsyncHealthStore struct {
	results map[string]error
	mu      sync.RWMutex
}

// NewAsyncHealthStore creates a new AsyncHealthStore.
func NewAsyncHealthStore() *AsyncHealthStore {
	return &AsyncHealthStore{
		results: make(map[string]error),
	}
}

// UpdateStatus updates the status of a health check.
func (s *AsyncHealthStore) UpdateStatus(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[name] = err
}

// GetStatus retrieves the status of a health check.
func (s *AsyncHealthStore) GetStatus(name string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[name]
}

// AsyncCheckerComponent adapts a HealthChecker for asynchronous checks using a store.
type AsyncCheckerComponent struct {
	store     *AsyncHealthStore
	component health.Component
	checkFunc func(ctx context.Context) error
}

// Name returns the name of the async health check component.
func (ac *AsyncCheckerComponent) Name() string {
	return ac.component.Name
}

// Check retrieves the cached health check result for async check.
func (ac *AsyncCheckerComponent) Check(ctx context.Context) error {
	return ac.checkFunc(ctx)
}

// Type returns the type of the async health check component.
func (ac *AsyncCheckerComponent) Type() string {
	return "async"
}

// AsyncChecker manages the asynchronous execution of a HealthChecker.
type AsyncChecker struct {
	store     *AsyncHealthStore
	config    health.Config
	interval  time.Duration
	component *AsyncCheckerComponent
}

// NewAsyncChecker creates a new AsyncChecker.
func NewAsyncChecker(checker HealthChecker, interval time.Duration) *AsyncChecker {
	ac := &AsyncChecker{
		store:    NewAsyncHealthStore(),
		interval: interval,
	}
	ac.component = &AsyncCheckerComponent{
		store: ac.store,
		component: health.Component{
			Name: checker.Name(),
		},
		checkFunc: func(ctx context.Context) error {
			return ac.store.GetStatus(checker.Name())
		},
	}

	ac.config = health.Config{
		Name:  checker.Name(),
		Check: checker.Check,
	}
	return ac
}

// Start starts the asynchronous health check loop.
func (ac *AsyncChecker) Start(ctx context.Context) {
	go func() {
		for {
			err := ac.config.Check(ctx)
			ac.store.UpdateStatus(ac.config.Name, err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(ac.interval):
			}
		}
	}()
}

// HealthCheck returns the health.Component for the health-go library (for async checks).
func (ac *AsyncChecker) HealthCheck() health.Component {
	return ac.component.component
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

// SetupHealthChecks sets up and registers health checks, returning the health.Health instance.
func SetupHealthChecks(componentName, componentVersion string, enableSystemInfo bool, checkers ...HealthChecker) (*health.Health, error) {
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
		checkConfig := health.Config{
			Name:    checker.Name(),
			Timeout: checker.GetTimeOut(),
			Check:   checker.Check,
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
