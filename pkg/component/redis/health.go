package redis

import (
	"context"
	"fmt"
)

// CheckHealth performs a health check with a default timeout.
// This is a convenience method for simple health check scenarios.
//
// Returns nil if healthy, error otherwise.
func (c *Client) CheckHealth(ctx context.Context) error {
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
