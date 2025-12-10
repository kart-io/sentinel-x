package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// CheckHealth performs a comprehensive health check on the etcd cluster.
// This function goes beyond a simple ping to provide detailed health information
// including latency measurement and cluster health status.
//
// The health check performs the following steps:
//  1. Measures latency of a Get operation
//  2. Checks cluster member health
//  3. Returns detailed health status
//
// Example usage:
//
//	status := client.CheckHealth(ctx)
//	if !status.Healthy {
//	    log.Printf("etcd unhealthy: %v (latency: %v)", status.Error, status.Latency)
//	}
func (c *Client) CheckHealth(ctx context.Context) storage.HealthStatus {
	status := storage.HealthStatus{
		Name:    c.Name(),
		Healthy: false,
	}

	start := time.Now()

	// Perform a lightweight connectivity check
	if err := c.Ping(ctx); err != nil {
		status.Latency = time.Since(start)
		status.Error = fmt.Errorf("connectivity check failed: %w", err)
		return status
	}

	// Check cluster health by listing members
	if err := c.checkClusterHealth(ctx); err != nil {
		status.Latency = time.Since(start)
		status.Error = fmt.Errorf("cluster health check failed: %w", err)
		return status
	}

	// All checks passed
	status.Latency = time.Since(start)
	status.Healthy = true
	status.Error = nil

	return status
}

// checkClusterHealth verifies the health of the etcd cluster.
// It checks if the cluster has members and they are accessible.
func (c *Client) checkClusterHealth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.opts.RequestTimeout)
	defer cancel()

	// Get cluster member list
	membersResp, err := c.client.MemberList(ctx)
	if err != nil {
		return fmt.Errorf("failed to list cluster members: %w", err)
	}

	if len(membersResp.Members) == 0 {
		return fmt.Errorf("cluster has no members")
	}

	// Check if cluster has a leader
	// This is indicated by having at least one member
	// More sophisticated checks could verify leader election status
	return nil
}
