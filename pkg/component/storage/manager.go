package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/pool"
)

// Manager manages multiple storage clients and provides centralized
// health checking, lifecycle management, and client registry functionality.
// It is safe for concurrent use.
//
// Example usage:
//
//	mgr := storage.NewManager()
//
//	// Register clients
//	mgr.Register("redis-cache", redisClient)
//	mgr.Register("mysql-primary", mysqlClient)
//
//	// Get a specific client
//	client, err := mgr.Get("redis-cache")
//
//	// Health check all clients
//	statuses := mgr.HealthCheckAll(ctx)
//
//	// Close all clients on shutdown
//	defer mgr.CloseAll()
type Manager struct {
	mu      sync.RWMutex
	clients map[string]Client
}

// NewManager creates a new storage manager instance.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]Client),
	}
}

// Register registers a storage client with the given name.
// The name should be unique and descriptive (e.g., "redis-cache", "mysql-primary").
//
// Returns an error if a client with the same name is already registered.
//
// Example usage:
//
//	err := mgr.Register("redis-cache", redisClient)
//	if err != nil {
//	    log.Fatalf("failed to register client: %v", err)
//	}
func (m *Manager) Register(name string, client Client) error {
	if name == "" {
		return ErrInvalidConfig.WithMessage("client name cannot be empty")
	}

	if client == nil {
		return ErrInvalidConfig.WithMessage("client cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[name]; exists {
		return ErrClientAlreadyExists.WithMessage(fmt.Sprintf("client '%s' is already registered", name))
	}

	m.clients[name] = client
	return nil
}

// MustRegister registers a storage client and panics if registration fails.
// This is useful for initialization code where failure should be fatal.
//
// Example usage:
//
//	mgr.MustRegister("redis-cache", redisClient)
func (m *Manager) MustRegister(name string, client Client) {
	if err := m.Register(name, client); err != nil {
		panic(fmt.Sprintf("failed to register storage client: %v", err))
	}
}

// Unregister removes a storage client from the manager.
// It does NOT close the client - the caller is responsible for that.
//
// Returns an error if the client is not found.
//
// Example usage:
//
//	client, err := mgr.Get("redis-cache")
//	if err == nil {
//	    client.Close()
//	}
//	mgr.Unregister("redis-cache")
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[name]; !exists {
		return ErrClientNotFound.WithMessage(fmt.Sprintf("client '%s' not found", name))
	}

	delete(m.clients, name)
	return nil
}

// Get retrieves a storage client by name.
//
// Returns the client if found, or an error if not found.
//
// Example usage:
//
//	client, err := mgr.Get("redis-cache")
//	if err != nil {
//	    log.Printf("client not found: %v", err)
//	    return
//	}
//	client.Ping(ctx)
func (m *Manager) Get(name string) (Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[name]
	if !exists {
		return nil, ErrClientNotFound.WithMessage(fmt.Sprintf("client '%s' not found", name))
	}

	return client, nil
}

// Has checks if a client with the given name is registered.
//
// Example usage:
//
//	if mgr.Has("redis-cache") {
//	    client, _ := mgr.Get("redis-cache")
//	    // use client...
//	}
func (m *Manager) Has(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.clients[name]
	return exists
}

// List returns a list of all registered client names.
//
// Example usage:
//
//	names := mgr.List()
//	fmt.Printf("Registered clients: %v\n", names)
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}

	return names
}

// Count returns the number of registered clients.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.clients)
}

// HealthCheck performs a health check on a specific client.
// It measures the latency and captures any errors.
//
// Example usage:
//
//	status := mgr.HealthCheck(ctx, "redis-cache")
//	if !status.Healthy {
//	    log.Printf("health check failed: %v", status.Error)
//	}
func (m *Manager) HealthCheck(ctx context.Context, name string) HealthStatus {
	client, err := m.Get(name)
	if err != nil {
		return HealthStatus{
			Name:    name,
			Healthy: false,
			Error:   err,
		}
	}

	start := time.Now()
	err = client.Ping(ctx)
	latency := time.Since(start)

	return HealthStatus{
		Name:    name,
		Healthy: err == nil,
		Latency: latency,
		Error:   err,
	}
}

// HealthCheckAll performs health checks on all registered clients concurrently.
// It returns a map of client names to their health status.
// 使用 ants 池执行并行健康检查，避免无限制创建 goroutine
//
// Example usage:
//
//	statuses := mgr.HealthCheckAll(ctx)
//	for name, status := range statuses {
//	    if !status.Healthy {
//	        log.Printf("%s is unhealthy: %v", name, status.Error)
//	    }
//	}
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]HealthStatus {
	m.mu.RLock()
	clients := make(map[string]Client, len(m.clients))
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mu.RUnlock()

	statuses := make(map[string]HealthStatus, len(clients))
	var statusMu sync.Mutex
	var wg sync.WaitGroup

	// 获取健康检查池
	healthPool, err := pool.GetByType(pool.HealthCheckPool)
	usePool := err == nil && healthPool != nil

	// Perform health checks concurrently
	for name, client := range clients {
		wg.Add(1)
		task := func(n string, c Client) {
			defer wg.Done()

			start := time.Now()
			err := c.Ping(ctx)
			latency := time.Since(start)

			statusMu.Lock()
			statuses[n] = HealthStatus{
				Name:    n,
				Healthy: err == nil,
				Latency: latency,
				Error:   err,
			}
			statusMu.Unlock()
		}

		// 使用池提交任务，失败时降级为直接创建 goroutine
		if usePool {
			n, c := name, client
			if submitErr := healthPool.Submit(func() { task(n, c) }); submitErr != nil {
				go task(name, client)
			}
		} else {
			go task(name, client)
		}
	}

	wg.Wait()
	return statuses
}

// AllHealthy checks if all registered clients are healthy.
// This is a convenience method that performs health checks and
// returns true only if all clients pass.
//
// Example usage:
//
//	if !mgr.AllHealthy(ctx) {
//	    log.Println("one or more storage clients are unhealthy")
//	}
func (m *Manager) AllHealthy(ctx context.Context) bool {
	statuses := m.HealthCheckAll(ctx)
	for _, status := range statuses {
		if !status.Healthy {
			return false
		}
	}
	return true
}

// Close closes a specific client and removes it from the manager.
//
// Example usage:
//
//	if err := mgr.Close("redis-cache"); err != nil {
//	    log.Printf("failed to close client: %v", err)
//	}
func (m *Manager) Close(name string) error {
	client, err := m.Get(name)
	if err != nil {
		return err
	}

	// Close the client
	if closeErr := client.Close(); closeErr != nil {
		return closeErr
	}

	// Remove from registry
	return m.Unregister(name)
}

// CloseAll closes all registered clients gracefully.
// It attempts to close all clients even if some fail, and returns
// the first error encountered (if any).
//
// This method should be called during application shutdown.
//
// Example usage:
//
//	defer mgr.CloseAll()
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error

	for name, client := range m.clients {
		if err := client.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close client '%s': %w", name, err)
		}
		delete(m.clients, name)
	}

	return firstErr
}

// Clear removes all clients from the manager without closing them.
// This is useful for testing or when clients need to be managed separately.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients = make(map[string]Client)
}
