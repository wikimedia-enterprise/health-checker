package health

import "sync"

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
