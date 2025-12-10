package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CheckHealth performs a health check on the PostgreSQL database.
// It verifies that the database is accessible and responsive.
func (c *Client) CheckHealth(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("postgres database connection is nil")
	}

	// Get the underlying sql.DB
	sqlDB, err := c.SqlDB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Create a context with timeout for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Ping the database
	if err := sqlDB.PingContext(checkCtx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}

	// Verify database stats
	stats := sqlDB.Stats()
	if stats.OpenConnections == 0 && stats.MaxOpenConnections > 0 {
		return fmt.Errorf("no open connections available")
	}

	return nil
}

// Stats returns database connection statistics.
func (c *Client) Stats() (sql.DBStats, error) {
	if c.db == nil {
		return sql.DBStats{}, fmt.Errorf("postgres client is nil")
	}

	sqlDB, err := c.SqlDB()
	if err != nil {
		return sql.DBStats{}, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	return sqlDB.Stats(), nil
}
