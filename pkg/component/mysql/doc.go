// Package mysql provides a MySQL storage client implementation for the sentinel-x project.
//
// This package implements the storage.Client interface and provides a robust,
// production-ready MySQL client built on top of GORM. It includes:
//
// - Connection pooling with configurable parameters
// - Health checking and monitoring
// - Graceful shutdown support
// - Context-aware operations
// - Configurable logging levels
//
// # Basic Usage
//
// Create a MySQL client with default options:
//
//	opts := mysqlOpts.NewOptions()
//	opts.Host = "localhost"
//	opts.Database = "myapp"
//
//	client, err := New(opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
// # Using with Context
//
// Create a client with timeout control:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	client, err := NewWithContext(ctx, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Health Checking
//
// Basic health check:
//
//	if err := client.Ping(context.Background()); err != nil {
//	    log.Printf("MySQL unhealthy: %v", err)
//	}
//
// Comprehensive health check with statistics:
//
//	status, stats := HealthWithStats(client, 5*time.Second)
//	if !status.Healthy {
//	    log.Printf("Health check failed: %v", status.Error)
//	} else {
//	    log.Printf("MySQL healthy (latency: %v)", status.Latency)
//	    log.Printf("Pool stats: %+v", stats)
//	}
//
// # Using the Factory Pattern
//
// Create clients using the factory:
//
//	factory := NewFactory(opts)
//	client, err := factory.Create(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Accessing GORM
//
// Get the underlying GORM DB for advanced operations:
//
//	db := client.DB()
//
//	// Use GORM features
//	db.AutoMigrate(&User{})
//	db.Create(&User{Name: "Alice"})
//
//	var users []User
//	db.Where("name = ?", "Alice").Find(&users)
//
// # Connection Pool Configuration
//
// Configure connection pool parameters:
//
//	opts := mysqlOpts.NewOptions()
//	opts.MaxIdleConnections = 10
//	opts.MaxOpenConnections = 100
//	opts.MaxConnectionLifeTime = 10 * time.Second
//
//	client, err := New(opts)
//
// # Integration with Health Middleware
//
// Register with the health check middleware:
//
//	healthMgr := middleware.GetHealthManager()
//	healthMgr.RegisterChecker("mysql", client.Health())
//
// # Error Handling
//
// The package provides detailed error messages for common scenarios:
//
//	client, err := New(opts)
//	if err != nil {
//	    switch {
//	    case strings.Contains(err.Error(), "invalid mysql options"):
//	        log.Printf("Configuration error: %v", err)
//	    case strings.Contains(err.Error(), "failed to connect"):
//	        log.Printf("Connection error: %v", err)
//	    case strings.Contains(err.Error(), "failed to ping"):
//	        log.Printf("Ping error: %v", err)
//	    default:
//	        log.Printf("Unknown error: %v", err)
//	    }
//	}
//
// # Thread Safety
//
// The MySQL client is safe for concurrent use by multiple goroutines.
// The underlying GORM and database/sql packages handle connection pooling
// and thread safety automatically.
package mysql
