package mongodb

import (
	"context"
	"fmt"
)

// CheckHealth performs a health check on the MongoDB connection.
// It verifies that the database is accessible and responsive.
//
// Example usage:
//
//	err := client.CheckHealth(ctx)
//	if err != nil {
//	    log.Printf("MongoDB unhealthy: %v", err)
//	}
func (c *Client) CheckHealth(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("mongodb client is nil")
	}
	return c.Ping(ctx)
}
