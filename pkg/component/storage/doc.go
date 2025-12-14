// Package storage provides a unified interface for storage backends in sentinel-x.
//
// This package defines the core abstractions that all storage implementations must follow,
// enabling consistent behavior across different storage types (Redis, MySQL, MongoDB, etc.).
//
// # Overview
//
// The storage package provides:
//   - Client interface: Base interface for all storage clients
//   - Manager: Registry and lifecycle management for multiple storage clients
//   - Options: Configuration validation and common options
//   - Errors: Standardized error types with context support
//   - Health checking: Built-in health check functionality
//
// # Quick Start
//
// Basic usage with a storage client:
//
//	// Create a storage client (Redis example)
//	client := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//
//	// Verify connectivity
//	ctx := context.Background()
//	if err := client.Ping(ctx); err != nil {
//	    log.Fatalf("failed to connect: %v", err)
//	}
//	defer client.Close()
//
// # Using the Manager
//
// For applications using multiple storage backends:
//
//	// Create a manager
//	mgr := storage.NewManager()
//
//	// Register multiple clients
//	mgr.MustRegister("redis-cache", redisClient)
//	mgr.MustRegister("mysql-primary", mysqlClient)
//	mgr.MustRegister("mongo-events", mongoClient)
//
//	// Get a specific client
//	cache, err := mgr.Get("redis-cache")
//	if err != nil {
//	    log.Printf("cache not available: %v", err)
//	}
//
//	// Health check all clients
//	statuses := mgr.HealthCheckAll(ctx)
//	for name, status := range statuses {
//	    if status.Healthy {
//	        log.Printf("%s: healthy (latency: %v)", name, status.Latency)
//	    } else {
//	        log.Printf("%s: unhealthy - %v", name, status.Error)
//	    }
//	}
//
//	// Close all clients on shutdown
//	defer mgr.CloseAll()
//
// # Health Checking
//
// All storage clients support health checking:
//
//	// Direct health check
//	if err := client.Ping(ctx); err != nil {
//	    log.Printf("health check failed: %v", err)
//	}
//
//	// Using the health checker function
//	checker := client.Health()
//	if err := checker(); err != nil {
//	    log.Printf("health check failed: %v", err)
//	}
//
//	// Manager-level health checks
//	status := mgr.HealthCheck(ctx, "redis-cache")
//	if !status.Healthy {
//	    log.Printf("unhealthy: %v (latency: %v)", status.Error, status.Latency)
//	}
//
// # Error Handling
//
// The package provides standardized error types:
//
//	err := client.Ping(ctx)
//	if err != nil {
//	    // Check for specific error types
//	    if errors.Is(err, storage.ErrNotConnected) {
//	        log.Println("client is not connected")
//	    } else if errors.Is(err, storage.ErrTimeout) {
//	        log.Println("operation timed out")
//	    }
//
//	    // Extract storage error details
//	    if storageErr, ok := storage.GetError(err); ok {
//	        log.Printf("error code: %s", storageErr.Code)
//	        log.Printf("message: %s", storageErr.Message)
//	        if ctx, ok := storageErr.GetContext("operation"); ok {
//	            log.Printf("operation: %v", ctx)
//	        }
//	    }
//	}
//
// # Configuration
//
// Storage implementations should embed CommonOptions for consistent configuration:
//
//	type RedisOptions struct {
//	    storage.CommonOptions
//	    Addr     string
//	    Password string
//	    DB       int
//	}
//
//	func (o *RedisOptions) Validate() error {
//	    // Validate common options first
//	    if err := o.CommonOptions.Validate(); err != nil {
//	        return err
//	    }
//
//	    // Add type-specific validation
//	    if o.Addr == "" {
//	        return storage.ErrInvalidConfig.WithMessage("address is required")
//	    }
//
//	    return nil
//	}
//
// # Implementing a Storage Client
//
// To implement a new storage type:
//
//	type MyStorageClient struct {
//	    conn *SomeConnection
//	}
//
//	func (c *MyStorageClient) Name() string {
//	    return "mystorage"
//	}
//
//	func (c *MyStorageClient) Ping(ctx context.Context) error {
//	    // Implement lightweight health check
//	    return c.conn.Ping(ctx)
//	}
//
//	func (c *MyStorageClient) Close() error {
//	    // Cleanup resources
//	    return c.conn.Close()
//	}
//
//	func (c *MyStorageClient) Health() storage.HealthChecker {
//	    return func() error {
//	        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	        defer cancel()
//	        return c.Ping(ctx)
//	    }
//	}
//
// # Thread Safety
//
// The Manager is safe for concurrent use. Storage client implementations
// should document their own thread-safety guarantees.
//
// # Context Support
//
// All operations that may block accept a context.Context for cancellation
// and timeout control:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	if err := client.Ping(ctx); err != nil {
//	    if errors.Is(err, context.DeadlineExceeded) {
//	        log.Println("operation timed out")
//	    }
//	}
package storage
