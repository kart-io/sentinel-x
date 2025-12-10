package storage

// Options is the base interface that all storage configuration options must implement.
// It provides a common contract for validating storage-specific configuration
// before client creation. Each storage type (Redis, MySQL, etc.) should define
// its own concrete options type that implements this interface.
//
// Example implementation:
//
//	type RedisOptions struct {
//	    Addr     string
//	    Password string
//	    DB       int
//	}
//
//	func (o *RedisOptions) Validate() error {
//	    if o.Addr == "" {
//	        return fmt.Errorf("redis address is required")
//	    }
//	    if o.DB < 0 || o.DB > 15 {
//	        return fmt.Errorf("redis DB must be between 0 and 15")
//	    }
//	    return nil
//	}
//
// Example usage:
//
//	var opts storage.Options = &RedisOptions{
//	    Addr: "localhost:6379",
//	    DB:   0,
//	}
//
//	if err := opts.Validate(); err != nil {
//	    log.Fatalf("invalid options: %v", err)
//	}
type Options interface {
	// Validate checks if the options are valid and complete.
	// It should verify that:
	// - Required fields are populated
	// - Field values are within acceptable ranges
	// - Field combinations are logically consistent
	//
	// Returns an error describing what is invalid or missing.
	// Returns nil if all validations pass.
	Validate() error
}

// CommonOptions contains configuration fields that are common across
// different storage types. Specific storage implementations can embed
// this struct to inherit common configuration functionality.
type CommonOptions struct {
	// MaxRetries is the maximum number of retry attempts for operations.
	// Set to 0 to disable retries, -1 for unlimited retries.
	// Default should be 3 for most implementations.
	MaxRetries int

	// Timeout is the operation timeout duration.
	// This applies to individual operations like Ping, Get, Set, etc.
	// Default should be reasonable for network operations (e.g., 5 seconds).
	Timeout int64 // Duration in milliseconds

	// PoolSize is the maximum number of connections in the pool.
	// Set to 0 to use the implementation's default.
	// Higher values allow more concurrent operations but consume more resources.
	PoolSize int

	// MinIdleConns is the minimum number of idle connections to maintain.
	// This helps reduce latency by keeping connections ready.
	// Should be less than or equal to PoolSize.
	MinIdleConns int

	// EnableTracing enables OpenTelemetry tracing for storage operations.
	// When true, all storage operations should emit trace spans.
	EnableTracing bool

	// EnableMetrics enables Prometheus metrics collection.
	// When true, storage operations should record metrics like
	// operation counts, latencies, and error rates.
	EnableMetrics bool
}

// Validate performs validation on CommonOptions fields.
// This can be called by specific storage implementations as part
// of their own Validate methods.
//
// Example usage in a specific storage implementation:
//
//	func (o *RedisOptions) Validate() error {
//	    if err := o.CommonOptions.Validate(); err != nil {
//	        return err
//	    }
//	    // Add Redis-specific validation...
//	    return nil
//	}
func (o *CommonOptions) Validate() error {
	if o.MaxRetries < -1 {
		return ErrInvalidConfig.WithMessage("MaxRetries must be >= -1")
	}

	if o.Timeout < 0 {
		return ErrInvalidConfig.WithMessage("Timeout must be non-negative")
	}

	if o.PoolSize < 0 {
		return ErrInvalidConfig.WithMessage("PoolSize must be non-negative")
	}

	if o.MinIdleConns < 0 {
		return ErrInvalidConfig.WithMessage("MinIdleConns must be non-negative")
	}

	if o.MinIdleConns > o.PoolSize && o.PoolSize > 0 {
		return ErrInvalidConfig.WithMessage("MinIdleConns cannot exceed PoolSize")
	}

	return nil
}

// SetDefaults sets reasonable default values for common options.
// This should be called by storage implementations before validation
// to ensure all fields have sensible defaults.
func (o *CommonOptions) SetDefaults() {
	if o.MaxRetries == 0 {
		o.MaxRetries = 3
	}

	if o.Timeout == 0 {
		o.Timeout = 5000 // 5 seconds
	}

	if o.PoolSize == 0 {
		o.PoolSize = 10
	}

	if o.MinIdleConns == 0 {
		o.MinIdleConns = 2
	}
}
