package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// Client wraps gorm.DB and provides a PostgreSQL database client.
// It implements the storage.Client interface.
type Client struct {
	db     *gorm.DB
	opts   *Options
	health *HealthChecker
}

// Compile-time check that Client implements storage.Client.
var _ storage.Client = (*Client)(nil)

// New creates a new PostgreSQL client from the provided options.
// It validates the options, establishes a connection, and configures the connection pool.
func New(opts *Options) (*Client, error) {
	return NewWithContext(context.Background(), opts)
}

// NewWithContext creates a new PostgreSQL client with the given context.
// This allows for timeout and cancellation during connection establishment.
func NewWithContext(ctx context.Context, opts *Options) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("postgres options cannot be nil")
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid postgres options: %w", err)
	}

	// Additional validation
	if opts.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}

	// Build DSN
	dsn := BuildDSN(opts)

	// Configure GORM logger
	logLevel := gormlogger.Silent
	switch opts.LogLevel {
	case 1:
		logLevel = gormlogger.Silent
	case 2:
		logLevel = gormlogger.Error
	case 3:
		logLevel = gormlogger.Warn
	case 4:
		logLevel = gormlogger.Info
	}

	// Open database connection
	db, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Get underlying sql.DB for configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	// Create client
	client := &Client{
		db:   db,
		opts: opts,
	}

	// Initialize health checker
	client.health = NewHealthChecker(client)

	// Verify connection with context
	if err := client.Ping(ctx); err != nil {
		// Clean up on connection failure
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return client, nil
}

// DB returns the underlying gorm.DB instance.
// This allows direct access to GORM functionality.
func (c *Client) DB() *gorm.DB {
	return c.db
}

// SqlDB returns the underlying sql.DB instance.
// This allows access to the raw database/sql functionality.
func (c *Client) SqlDB() (*sql.DB, error) {
	if c.db == nil {
		return nil, fmt.Errorf("gorm.DB is nil")
	}
	return c.db.DB()
}

// Name returns the name of the storage client.
func (c *Client) Name() string {
	return "postgres"
}

// Ping verifies the connection to the PostgreSQL database.
func (c *Client) Ping(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := c.SqlDB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Create a context with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}

	return nil
}

// Close closes the database connection and releases resources.
func (c *Client) Close() error {
	if c.db == nil {
		return nil
	}

	sqlDB, err := c.SqlDB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for closing: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close postgres connection: %w", err)
	}

	return nil
}

// Health returns a HealthChecker function for PostgreSQL health monitoring.
// The returned function can be called to verify the PostgreSQL connection status.
// Implements storage.Client interface.
func (c *Client) Health() storage.HealthChecker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.Ping(ctx)
	}
}

// HealthCheck performs a health check on the database.
// This implements the service.HealthChecker interface.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.health.HealthCheck(ctx)
}

// Stats returns database connection statistics.
func (c *Client) Stats() (sql.DBStats, error) {
	return c.health.Stats()
}
