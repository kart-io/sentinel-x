package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mysqlOpts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	"github.com/kart-io/sentinel-x/pkg/storage"
)

// Client wraps gorm.DB with storage.Client interface implementation.
// It provides a unified MySQL client that implements the standard storage
// interface while exposing the underlying GORM database for advanced usage.
//
// Example usage:
//
//	opts := mysqlOpts.NewOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//
//	client, err := New(opts)
//	if err != nil {
//	    log.Fatalf("failed to create MySQL client: %v", err)
//	}
//	defer client.Close()
//
//	// Use GORM directly
//	db := client.DB()
//	db.AutoMigrate(&User{})
type Client struct {
	db   *gorm.DB
	opts *mysqlOpts.Options
}

// New creates a new MySQL client from the provided options.
// It validates the options, builds the DSN, and establishes a connection
// with the configured connection pool settings and logging level.
//
// Returns an error if:
// - Options validation fails
// - Connection to MySQL server fails
// - Connection pool configuration fails
func New(opts *mysqlOpts.Options) (*Client, error) {
	return NewWithContext(context.Background(), opts)
}

// NewWithContext creates a new MySQL client with context support.
// This allows for timeout control during the connection establishment phase.
//
// The context is used for:
// - Connection establishment timeout
// - Initial ping verification
//
// Returns an error if:
// - Options validation fails
// - Connection to MySQL server fails
// - Initial ping fails
// - Connection pool configuration fails
func NewWithContext(ctx context.Context, opts *mysqlOpts.Options) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("mysql options cannot be nil")
	}

	// Validate options
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid mysql options: %w", err)
	}

	// Build DSN
	dsn := BuildDSN(opts)

	// Configure GORM logger based on LogLevel
	gormLogger := logger.Default
	switch opts.LogLevel {
	case 1: // Silent
		gormLogger = logger.Default.LogMode(logger.Silent)
	case 2: // Error
		gormLogger = logger.Default.LogMode(logger.Error)
	case 3: // Warn
		gormLogger = logger.Default.LogMode(logger.Warn)
	case 4: // Info
		gormLogger = logger.Default.LogMode(logger.Info)
	default:
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	if opts.MaxIdleConnections > 0 {
		sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)
	}
	if opts.MaxOpenConnections > 0 {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)
	}
	if opts.MaxConnectionLifeTime > 0 {
		sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)
	}

	// Verify connection with context
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	return &Client{
		db:   db,
		opts: opts,
	}, nil
}

// Name returns the storage type identifier.
// Implements storage.Client interface.
func (c *Client) Name() string {
	return "mysql"
}

// Ping checks if the connection to MySQL is alive.
// It performs a lightweight ping operation to verify connectivity.
// Implements storage.Client interface.
func (c *Client) Ping(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the MySQL connection gracefully.
// It ensures all pending operations are completed and resources are released.
// This method is idempotent and safe to call multiple times.
// Implements storage.Client interface.
func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// Health returns a HealthChecker function for MySQL health monitoring.
// The returned function can be called to verify the MySQL connection status.
// Implements storage.Client interface.
func (c *Client) Health() storage.HealthChecker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return c.Ping(ctx)
	}
}

// DB returns the underlying gorm.DB instance.
// This allows direct access to GORM functionality for advanced database operations.
//
// Example:
//
//	db := client.DB()
//	db.AutoMigrate(&User{})
//	db.Create(&User{Name: "John"})
func (c *Client) DB() *gorm.DB {
	return c.db
}

// SqlDB returns the underlying sql.DB instance.
// This provides access to the standard library database/sql functionality
// for operations not available through GORM.
//
// Example:
//
//	sqlDB, err := client.SqlDB()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	stats := sqlDB.Stats()
//	fmt.Printf("Open connections: %d\n", stats.OpenConnections)
func (c *Client) SqlDB() (*sql.DB, error) {
	return c.db.DB()
}

// validateOptions validates MySQL options before creating the client.
// It ensures that required fields are set and values are reasonable.
func validateOptions(opts *mysqlOpts.Options) error {
	if opts.Host == "" {
		return fmt.Errorf("host is required")
	}
	if opts.Database == "" {
		return fmt.Errorf("database is required")
	}
	if opts.Port <= 0 || opts.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if opts.Username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}
