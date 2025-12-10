// Package datasource provides unified management for all storage clients.
//
// This package consolidates initialization, registration, and lifecycle management
// for all storage backends: MySQL, PostgreSQL, Redis, MongoDB, and Etcd.
//
// # Design Principles
//
//   - Single entry point for all storage access
//   - Support for multiple instances per storage type
//   - Lazy initialization with on-demand loading
//   - Built-in health checks and graceful shutdown
//   - Thread-safe operations
//
// # Usage Example
//
//	// Initialize the manager
//	mgr := datasource.NewManager()
//
//	// Register storage instances
//	mgr.RegisterMySQL("primary", mysqlOpts)
//	mgr.RegisterRedis("cache", redisOpts)
//
//	// Initialize all registered instances
//	if err := mgr.InitAll(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer mgr.CloseAll()
//
//	// Get clients in business layer
//	db, err := mgr.GetMySQL("primary")
//	cache, err := mgr.GetRedis("cache")
package datasource

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/mongodb"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/postgres"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// StorageType represents the type of storage backend.
type StorageType string

const (
	TypeMySQL    StorageType = "mysql"
	TypePostgres StorageType = "postgres"
	TypeRedis    StorageType = "redis"
	TypeMongoDB  StorageType = "mongodb"
	TypeEtcd     StorageType = "etcd"
)

// storageEntry holds options and client for any storage type
type storageEntry struct {
	storageType StorageType
	opts        interface{}
	client      interface{}
}

// Manager provides unified management for all storage clients.
// It handles registration, initialization, health checks, and graceful shutdown.
type Manager struct {
	mu sync.RWMutex

	// Unified storage map: key -> storageEntry
	entries map[string]*storageEntry

	// Initialization state
	initialized map[string]bool

	// Health check interval
	healthCheckInterval time.Duration
}

// NewManager creates a new storage manager.
func NewManager() *Manager {
	return &Manager{
		entries:             make(map[string]*storageEntry),
		initialized:         make(map[string]bool),
		healthCheckInterval: 30 * time.Second,
	}
}

// =============================================================================
// Registration Methods
// =============================================================================

// RegisterMySQL registers a MySQL instance with the given name and options.
// The actual connection is established lazily when GetMySQL is called or during InitAll.
func (m *Manager) RegisterMySQL(name string, opts *mysql.Options) error {
	optsCopy := *opts
	return m.register(TypeMySQL, name, &optsCopy)
}

// RegisterPostgres registers a PostgreSQL instance with the given name and options.
func (m *Manager) RegisterPostgres(name string, opts *postgres.Options) error {
	optsCopy := *opts
	return m.register(TypePostgres, name, &optsCopy)
}

// RegisterRedis registers a Redis instance with the given name and options.
func (m *Manager) RegisterRedis(name string, opts *redis.Options) error {
	optsCopy := *opts
	return m.register(TypeRedis, name, &optsCopy)
}

// RegisterMongoDB registers a MongoDB instance with the given name and options.
func (m *Manager) RegisterMongoDB(name string, opts *mongodb.Options) error {
	optsCopy := *opts
	return m.register(TypeMongoDB, name, &optsCopy)
}

// RegisterEtcd registers an Etcd instance with the given name and options.
func (m *Manager) RegisterEtcd(name string, opts *etcd.Options) error {
	optsCopy := *opts
	return m.register(TypeEtcd, name, &optsCopy)
}

// register is a unified registration method for all storage types.
func (m *Manager) register(storageType StorageType, name string, opts interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(storageType, name)
	if _, exists := m.entries[key]; exists {
		return fmt.Errorf("%s instance '%s' already registered", storageType, name)
	}

	m.entries[key] = &storageEntry{
		storageType: storageType,
		opts:        opts,
		client:      nil,
	}
	return nil
}

// =============================================================================
// Initialization Methods
// =============================================================================

// InitAll initializes all registered storage instances.
// Returns an error if any initialization fails, with automatic rollback of successful connections.
func (m *Manager) InitAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var initialized []string
	rollback := func() {
		// Clear clients from entries
		for _, key := range initialized {
			if entry := m.entries[key]; entry != nil {
				m.closeClient(entry.client)
				entry.client = nil
			}
		}
		// Clear initialized map
		for _, key := range initialized {
			delete(m.initialized, key)
		}
	}

	for key, entry := range m.entries {
		if m.initialized[key] {
			continue
		}

		client, err := m.createClient(ctx, entry.storageType, entry.opts)
		if err != nil {
			rollback()
			_, name := parseKey(key)
			return fmt.Errorf("failed to init %s '%s': %w", entry.storageType, name, err)
		}

		entry.client = client
		m.initialized[key] = true
		initialized = append(initialized, key)
	}

	return nil
}

// InitMySQL initializes a specific MySQL instance by name.
func (m *Manager) InitMySQL(ctx context.Context, name string) error {
	return m.initByType(ctx, TypeMySQL, name)
}

// InitPostgres initializes a specific PostgreSQL instance by name.
func (m *Manager) InitPostgres(ctx context.Context, name string) error {
	return m.initByType(ctx, TypePostgres, name)
}

// InitRedis initializes a specific Redis instance by name.
func (m *Manager) InitRedis(ctx context.Context, name string) error {
	return m.initByType(ctx, TypeRedis, name)
}

// InitMongoDB initializes a specific MongoDB instance by name.
func (m *Manager) InitMongoDB(ctx context.Context, name string) error {
	return m.initByType(ctx, TypeMongoDB, name)
}

// InitEtcd initializes a specific Etcd instance by name.
func (m *Manager) InitEtcd(ctx context.Context, name string) error {
	return m.initByType(ctx, TypeEtcd, name)
}

// initByType initializes a specific instance by type and name.
func (m *Manager) initByType(ctx context.Context, storageType StorageType, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(storageType, name)
	if m.initialized[key] {
		return nil
	}

	entry, exists := m.entries[key]
	if !exists {
		return fmt.Errorf("%s instance '%s' not registered", storageType, name)
	}

	client, err := m.createClient(ctx, storageType, entry.opts)
	if err != nil {
		return fmt.Errorf("failed to init %s '%s': %w", storageType, name, err)
	}

	entry.client = client
	m.initialized[key] = true
	return nil
}

// =============================================================================
// Getter Methods (with lazy initialization)
// =============================================================================

// GetMySQL returns the MySQL client by name.
// If not initialized, it will be initialized lazily.
func (m *Manager) GetMySQL(name string) (*mysql.Client, error) {
	client, err := m.getClient(TypeMySQL, name)
	if err != nil {
		return nil, err
	}
	return client.(*mysql.Client), nil
}

// GetPostgres returns the PostgreSQL client by name.
func (m *Manager) GetPostgres(name string) (*postgres.Client, error) {
	client, err := m.getClient(TypePostgres, name)
	if err != nil {
		return nil, err
	}
	return client.(*postgres.Client), nil
}

// GetRedis returns the Redis client by name.
func (m *Manager) GetRedis(name string) (*redis.Client, error) {
	client, err := m.getClient(TypeRedis, name)
	if err != nil {
		return nil, err
	}
	return client.(*redis.Client), nil
}

// GetMongoDB returns the MongoDB client by name.
func (m *Manager) GetMongoDB(name string) (*mongodb.Client, error) {
	client, err := m.getClient(TypeMongoDB, name)
	if err != nil {
		return nil, err
	}
	return client.(*mongodb.Client), nil
}

// GetEtcd returns the Etcd client by name.
func (m *Manager) GetEtcd(name string) (*etcd.Client, error) {
	client, err := m.getClient(TypeEtcd, name)
	if err != nil {
		return nil, err
	}
	return client.(*etcd.Client), nil
}

// GetMySQLWithContext returns the MySQL client by name with context support.
func (m *Manager) GetMySQLWithContext(ctx context.Context, name string) (*mysql.Client, error) {
	client, err := m.getClientWithContext(ctx, TypeMySQL, name)
	if err != nil {
		return nil, err
	}
	return client.(*mysql.Client), nil
}

// GetPostgresWithContext returns the PostgreSQL client by name with context support.
func (m *Manager) GetPostgresWithContext(ctx context.Context, name string) (*postgres.Client, error) {
	client, err := m.getClientWithContext(ctx, TypePostgres, name)
	if err != nil {
		return nil, err
	}
	return client.(*postgres.Client), nil
}

// GetRedisWithContext returns the Redis client by name with context support.
func (m *Manager) GetRedisWithContext(ctx context.Context, name string) (*redis.Client, error) {
	client, err := m.getClientWithContext(ctx, TypeRedis, name)
	if err != nil {
		return nil, err
	}
	return client.(*redis.Client), nil
}

// GetMongoDBWithContext returns the MongoDB client by name with context support.
func (m *Manager) GetMongoDBWithContext(ctx context.Context, name string) (*mongodb.Client, error) {
	client, err := m.getClientWithContext(ctx, TypeMongoDB, name)
	if err != nil {
		return nil, err
	}
	return client.(*mongodb.Client), nil
}

// GetEtcdWithContext returns the Etcd client by name with context support.
func (m *Manager) GetEtcdWithContext(ctx context.Context, name string) (*etcd.Client, error) {
	client, err := m.getClientWithContext(ctx, TypeEtcd, name)
	if err != nil {
		return nil, err
	}
	return client.(*etcd.Client), nil
}

// MustGetMySQL returns the MySQL client or panics if not available.
func (m *Manager) MustGetMySQL(name string) *mysql.Client {
	client, err := m.GetMySQL(name)
	if err != nil {
		panic(fmt.Sprintf("mysql instance '%s' not available: %v", name, err))
	}
	return client
}

// MustGetPostgres returns the PostgreSQL client or panics if not available.
func (m *Manager) MustGetPostgres(name string) *postgres.Client {
	client, err := m.GetPostgres(name)
	if err != nil {
		panic(fmt.Sprintf("postgres instance '%s' not available: %v", name, err))
	}
	return client
}

// MustGetRedis returns the Redis client or panics if not available.
func (m *Manager) MustGetRedis(name string) *redis.Client {
	client, err := m.GetRedis(name)
	if err != nil {
		panic(fmt.Sprintf("redis instance '%s' not available: %v", name, err))
	}
	return client
}

// MustGetMongoDB returns the MongoDB client or panics if not available.
func (m *Manager) MustGetMongoDB(name string) *mongodb.Client {
	client, err := m.GetMongoDB(name)
	if err != nil {
		panic(fmt.Sprintf("mongodb instance '%s' not available: %v", name, err))
	}
	return client
}

// MustGetEtcd returns the Etcd client or panics if not available.
func (m *Manager) MustGetEtcd(name string) *etcd.Client {
	client, err := m.GetEtcd(name)
	if err != nil {
		panic(fmt.Sprintf("etcd instance '%s' not available: %v", name, err))
	}
	return client
}

// getClient returns a client with lazy initialization using background context.
func (m *Manager) getClient(storageType StorageType, name string) (interface{}, error) {
	return m.getClientWithContext(context.Background(), storageType, name)
}

// getClientWithContext returns a client with lazy initialization using provided context.
func (m *Manager) getClientWithContext(ctx context.Context, storageType StorageType, name string) (interface{}, error) {
	key := makeKey(storageType, name)

	// Fast path: read lock
	m.mu.RLock()
	entry, exists := m.entries[key]
	if exists && entry.client != nil {
		m.mu.RUnlock()
		return entry.client, nil
	}
	m.mu.RUnlock()

	// Slow path: write lock for lazy initialization
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	entry, exists = m.entries[key]
	if !exists {
		return nil, fmt.Errorf("%s instance '%s' not registered", storageType, name)
	}

	if entry.client != nil {
		return entry.client, nil
	}

	// Lazy initialization
	client, err := m.createClient(ctx, storageType, entry.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to init %s '%s': %w", storageType, name, err)
	}

	entry.client = client
	m.initialized[key] = true
	return client, nil
}

// =============================================================================
// Health Check Methods
// =============================================================================

// HealthCheckAll performs health checks on all initialized storage instances in parallel.
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]storage.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]storage.HealthStatus)
	var resultsMu sync.Mutex
	var wg sync.WaitGroup

	for key, entry := range m.entries {
		if !m.initialized[key] || entry.client == nil {
			continue
		}

		wg.Add(1)
		go func(key string, client interface{}) {
			defer wg.Done()

			start := time.Now()
			err := m.pingClient(ctx, client)

			resultsMu.Lock()
			results[key] = storage.HealthStatus{
				Name:    key,
				Healthy: err == nil,
				Latency: time.Since(start),
				Error:   err,
			}
			resultsMu.Unlock()
		}(key, entry.client)
	}

	wg.Wait()
	return results
}

// IsHealthy returns true if all storage instances are healthy.
func (m *Manager) IsHealthy(ctx context.Context) bool {
	results := m.HealthCheckAll(ctx)
	for _, status := range results {
		if !status.Healthy {
			return false
		}
	}
	return true
}

// =============================================================================
// Close Methods
// =============================================================================

// CloseAll closes all initialized storage connections.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for key, entry := range m.entries {
		if entry.client == nil {
			continue
		}

		if err := m.closeClient(entry.client); err != nil {
			_, name := parseKey(key)
			errs = append(errs, fmt.Errorf("failed to close %s '%s': %w", entry.storageType, name, err))
		}

		entry.client = nil
		delete(m.initialized, key)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing storage: %v", errs)
	}

	return nil
}

// =============================================================================
// Utility Methods
// =============================================================================

// ListRegistered returns a list of all registered storage instances.
func (m *Manager) ListRegistered() map[StorageType][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[StorageType][]string)
	for key, entry := range m.entries {
		_, name := parseKey(key)
		result[entry.storageType] = append(result[entry.storageType], name)
	}

	return result
}

// ListInitialized returns a list of all initialized storage instances.
func (m *Manager) ListInitialized() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []string
	for key := range m.initialized {
		result = append(result, key)
	}
	return result
}

// =============================================================================
// Helper Functions
// =============================================================================

// makeKey creates a storage key from type and name using string concatenation.
func makeKey(storageType StorageType, name string) string {
	return string(storageType) + ":" + name
}

// parseKey parses a storage key into type and name.
func parseKey(key string) (StorageType, string) {
	for i, c := range key {
		if c == ':' {
			return StorageType(key[:i]), key[i+1:]
		}
	}
	return "", key
}

// createClient creates a new client instance based on storage type.
func (m *Manager) createClient(ctx context.Context, storageType StorageType, opts interface{}) (interface{}, error) {
	switch storageType {
	case TypeMySQL:
		return mysql.NewWithContext(ctx, opts.(*mysql.Options))
	case TypePostgres:
		return postgres.NewWithContext(ctx, opts.(*postgres.Options))
	case TypeRedis:
		return redis.NewWithContext(ctx, opts.(*redis.Options))
	case TypeMongoDB:
		return mongodb.NewWithContext(ctx, opts.(*mongodb.Options))
	case TypeEtcd:
		return etcd.NewWithContext(ctx, opts.(*etcd.Options))
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// pingClient performs a health check on a client.
func (m *Manager) pingClient(ctx context.Context, client interface{}) error {
	switch c := client.(type) {
	case *mysql.Client:
		return c.Ping(ctx)
	case *postgres.Client:
		return c.Ping(ctx)
	case *redis.Client:
		return c.Ping(ctx)
	case *mongodb.Client:
		return c.Ping(ctx)
	case *etcd.Client:
		return c.Ping(ctx)
	default:
		return fmt.Errorf("unsupported client type: %T", client)
	}
}

// closeClient closes a client connection.
func (m *Manager) closeClient(client interface{}) error {
	switch c := client.(type) {
	case *mysql.Client:
		return c.Close()
	case *postgres.Client:
		return c.Close()
	case *redis.Client:
		return c.Close()
	case *mongodb.Client:
		return c.Close()
	case *etcd.Client:
		return c.Close()
	default:
		return fmt.Errorf("unsupported client type: %T", client)
	}
}

// =============================================================================
// Global Manager (Optional Singleton)
// =============================================================================

var (
	globalManager    *Manager
	globalManagerMu  sync.RWMutex
	globalManagerSet bool
)

// GetGlobal returns the global singleton manager instance.
// If not set via SetGlobal, it creates a default manager instance.
// This function is thread-safe.
func GetGlobal() *Manager {
	globalManagerMu.RLock()
	if globalManager != nil {
		defer globalManagerMu.RUnlock()
		return globalManager
	}
	globalManagerMu.RUnlock()

	// Double-check pattern for initialization
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	if globalManager == nil {
		globalManager = NewManager()
		globalManagerSet = true
	}
	return globalManager
}

// SetGlobal sets the global manager instance.
// This should be called early in application initialization before any calls to GetGlobal.
// Returns an error if a global manager has already been set or initialized.
// This function is thread-safe.
func SetGlobal(mgr *Manager) error {
	if mgr == nil {
		return fmt.Errorf("cannot set nil manager as global instance")
	}

	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	if globalManagerSet {
		return fmt.Errorf("global manager already set, cannot override existing instance")
	}

	globalManager = mgr
	globalManagerSet = true
	return nil
}

// MustSetGlobal sets the global manager instance or panics if already set.
// This should only be used in initialization code where failure is unrecoverable.
func MustSetGlobal(mgr *Manager) {
	if err := SetGlobal(mgr); err != nil {
		panic(fmt.Sprintf("failed to set global manager: %v", err))
	}
}

// ResetGlobal resets the global manager instance.
// This is primarily intended for testing purposes and should not be used in production code.
// Returns the previous manager instance if any.
func ResetGlobal() *Manager {
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	prev := globalManager
	globalManager = nil
	globalManagerSet = false
	return prev
}
