package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// HealthStats contains detailed health information about the Redis connection.
type HealthStats struct {
	// Healthy indicates whether Redis is reachable.
	Healthy bool `json:"healthy"`

	// Latency is the time taken to complete a ping operation.
	Latency time.Duration `json:"latency"`

	// PoolStats contains connection pool statistics.
	PoolStats *PoolStats `json:"pool_stats,omitempty"`

	// Error contains any error message if unhealthy.
	Error string `json:"error,omitempty"`
}

// PoolStats contains Redis connection pool statistics.
type PoolStats struct {
	// Hits is the number of times a free connection was found in the pool.
	Hits uint32 `json:"hits"`

	// Misses is the number of times a free connection was NOT found in the pool.
	Misses uint32 `json:"misses"`

	// Timeouts is the number of times a wait timeout occurred.
	Timeouts uint32 `json:"timeouts"`

	// TotalConns is the number of total connections in the pool.
	TotalConns uint32 `json:"total_conns"`

	// IdleConns is the number of idle connections in the pool.
	IdleConns uint32 `json:"idle_conns"`

	// StaleConns is the number of stale connections removed from the pool.
	StaleConns uint32 `json:"stale_conns"`
}

// HealthWithStats performs a health check and returns detailed statistics.
// This provides more information than the basic Health() function.
//
// Example:
//
//	stats := client.HealthWithStats(ctx)
//	fmt.Printf("Healthy: %v, Latency: %v\n", stats.Healthy, stats.Latency)
func (c *Client) HealthWithStats(ctx context.Context) *HealthStats {
	stats := &HealthStats{}

	start := time.Now()
	err := c.Ping(ctx)
	stats.Latency = time.Since(start)

	if err != nil {
		stats.Healthy = false
		stats.Error = err.Error()
		return stats
	}

	stats.Healthy = true

	// Get pool statistics
	poolStats := c.client.PoolStats()
	stats.PoolStats = &PoolStats{
		Hits:       poolStats.Hits,
		Misses:     poolStats.Misses,
		Timeouts:   poolStats.Timeouts,
		TotalConns: poolStats.TotalConns,
		IdleConns:  poolStats.IdleConns,
		StaleConns: poolStats.StaleConns,
	}

	return stats
}

// HealthStatus returns a storage.HealthStatus for integration with the
// storage.Manager health check system.
//
// Example:
//
//	status := client.HealthStatus(ctx)
//	fmt.Printf("Name: %s, Healthy: %v\n", status.Name, status.Healthy)
func (c *Client) HealthStatus(ctx context.Context) storage.HealthStatus {
	start := time.Now()
	err := c.Ping(ctx)
	latency := time.Since(start)

	return storage.HealthStatus{
		Name:    c.Name(),
		Healthy: err == nil,
		Latency: latency,
		Error:   err,
	}
}

// CheckHealth performs a health check with a default timeout.
// This is a convenience method for simple health check scenarios.
//
// Returns nil if healthy, error otherwise.
func (c *Client) CheckHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.Ping(ctx)
}

// IsHealthy returns true if the Redis connection is healthy.
// This is a convenience method that returns a boolean instead of an error.
func (c *Client) IsHealthy() bool {
	return c.CheckHealth() == nil
}

// HealthWithTimeout performs a health check with a custom timeout.
//
// Example:
//
//	err := client.HealthWithTimeout(2 * time.Second)
func (c *Client) HealthWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Ping(ctx)
}

// Info returns Redis server information.
// This can be used for detailed diagnostics.
//
// Example:
//
//	info, err := client.Info(ctx, "server")
//	// Returns server section of INFO command
func (c *Client) Info(ctx context.Context, sections ...string) (string, error) {
	result, err := c.client.Info(ctx, sections...).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get redis info: %w", err)
	}
	return result, nil
}

// DBSize returns the number of keys in the current database.
func (c *Client) DBSize(ctx context.Context) (int64, error) {
	return c.client.DBSize(ctx).Result()
}
