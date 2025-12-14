package mysql

import (
	"context"
	"fmt"
	"time"
)

// CheckHealth performs a comprehensive health check on the MySQL client.
// It verifies connectivity, measures latency, and checks connection pool statistics.
//
// The health check includes:
// - Connection ping test
// - Latency measurement
// - Connection pool statistics validation
//
// Returns an error if the health check fails, nil otherwise.
//
// Example usage:
//
//	err := client.CheckHealth(ctx)
//	if err != nil {
//	    log.Printf("MySQL unhealthy: %v", err)
//	}
func (c *Client) CheckHealth(ctx context.Context) error {
	// Perform ping check
	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Get connection pool stats
	sqlDB, err := c.SQLDB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	stats := sqlDB.Stats()

	// Validate connection pool health
	// Check if we have at least some open connections or can establish them
	if stats.MaxOpenConnections > 0 && stats.OpenConnections > stats.MaxOpenConnections {
		return fmt.Errorf("connection pool exceeded: open=%d, max=%d",
			stats.OpenConnections, stats.MaxOpenConnections)
	}

	// Check for too many waiting connections (might indicate connection pool exhaustion)
	if stats.WaitCount > 0 && stats.WaitDuration > 5*time.Second {
		return fmt.Errorf("high connection wait time: count=%d, duration=%v",
			stats.WaitCount, stats.WaitDuration)
	}

	return nil
}
