package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// CheckHealth performs a comprehensive health check on the MySQL client.
// It verifies connectivity, measures latency, and checks connection pool statistics.
//
// The health check includes:
// - Connection ping test
// - Latency measurement
// - Connection pool statistics validation
//
// Returns a HealthStatus with detailed information about the check results.
//
// Example usage:
//
//	status := CheckHealth(client, 5*time.Second)
//	if !status.Healthy {
//	    log.Printf("MySQL unhealthy: %v", status.Error)
//	} else {
//	    log.Printf("MySQL healthy, latency: %v", status.Latency)
//	}
func CheckHealth(client *Client, timeout time.Duration) storage.HealthStatus {
	status := storage.HealthStatus{
		Name:    client.Name(),
		Healthy: false,
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Perform ping check
	if err := client.Ping(ctx); err != nil {
		status.Error = fmt.Errorf("ping failed: %w", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Latency = time.Since(start)

	// Get connection pool stats
	sqlDB, err := client.SqlDB()
	if err != nil {
		status.Error = fmt.Errorf("failed to get sql.DB: %w", err)
		return status
	}

	stats := sqlDB.Stats()

	// Validate connection pool health
	// Check if we have at least some open connections or can establish them
	if stats.MaxOpenConnections > 0 && stats.OpenConnections > stats.MaxOpenConnections {
		status.Error = fmt.Errorf("connection pool exceeded: open=%d, max=%d",
			stats.OpenConnections, stats.MaxOpenConnections)
		return status
	}

	// Check for too many waiting connections (might indicate connection pool exhaustion)
	if stats.WaitCount > 0 && stats.WaitDuration > 5*time.Second {
		status.Error = fmt.Errorf("high connection wait time: count=%d, duration=%v",
			stats.WaitCount, stats.WaitDuration)
		return status
	}

	// All checks passed
	status.Healthy = true
	return status
}

// HealthWithStats performs a health check and returns detailed connection pool statistics.
// This is useful for monitoring and debugging connection pool behavior.
//
// Returns:
// - storage.HealthStatus: Overall health status
// - map[string]interface{}: Detailed statistics including pool metrics
//
// Example usage:
//
//	status, stats := HealthWithStats(client, 5*time.Second)
//	if status.Healthy {
//	    log.Printf("Pool stats: idle=%d, open=%d, in-use=%d",
//	        stats["idle_connections"],
//	        stats["open_connections"],
//	        stats["in_use_connections"])
//	}
func HealthWithStats(client *Client, timeout time.Duration) (storage.HealthStatus, map[string]interface{}) {
	status := CheckHealth(client, timeout)
	stats := make(map[string]interface{})

	sqlDB, err := client.SqlDB()
	if err != nil {
		stats["error"] = err.Error()
		return status, stats
	}

	dbStats := sqlDB.Stats()
	stats["max_open_connections"] = dbStats.MaxOpenConnections
	stats["open_connections"] = dbStats.OpenConnections
	stats["in_use_connections"] = dbStats.InUse
	stats["idle_connections"] = dbStats.Idle
	stats["wait_count"] = dbStats.WaitCount
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = dbStats.MaxIdleClosed
	stats["max_idle_time_closed"] = dbStats.MaxIdleTimeClosed
	stats["max_lifetime_closed"] = dbStats.MaxLifetimeClosed

	return status, stats
}
