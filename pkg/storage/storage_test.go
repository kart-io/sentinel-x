package storage

import (
	"context"
	"testing"
	"time"
)

// MockClient is a test implementation of the Client interface.
type MockClient struct {
	name    string
	healthy bool
}

func (m *MockClient) Name() string {
	return m.name
}

func (m *MockClient) Ping(ctx context.Context) error {
	if !m.healthy {
		return context.DeadlineExceeded
	}
	return nil
}

func (m *MockClient) Close() error {
	return nil
}

func (m *MockClient) Health() HealthChecker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return m.Ping(ctx)
	}
}

// Compile-time check that MockClient implements Client.
var _ Client = (*MockClient)(nil)

func TestHealthChecker_Healthy(t *testing.T) {
	client := &MockClient{name: "test", healthy: true}
	checker := client.Health()

	if err := checker(); err != nil {
		t.Errorf("expected healthy client to return nil, got %v", err)
	}
}

func TestHealthChecker_Unhealthy(t *testing.T) {
	client := &MockClient{name: "test", healthy: false}
	checker := client.Health()

	if err := checker(); err == nil {
		t.Error("expected unhealthy client to return error")
	}
}

func TestHealthStatus(t *testing.T) {
	status := HealthStatus{
		Name:    "test",
		Healthy: true,
		Latency: 10 * time.Millisecond,
		Error:   nil,
	}

	if status.Name != "test" {
		t.Errorf("expected name 'test', got %s", status.Name)
	}

	if !status.Healthy {
		t.Error("expected status to be healthy")
	}

	if status.Latency != 10*time.Millisecond {
		t.Errorf("expected latency 10ms, got %v", status.Latency)
	}
}

// TestFactoryInterface verifies the Factory interface signature.
func TestFactoryInterface(t *testing.T) {
	// This is a compile-time check, no runtime test needed
	var _ Factory = (*MockFactory)(nil)
}

// MockFactory is a test implementation of the Factory interface.
type MockFactory struct{}

func (m *MockFactory) Create(ctx context.Context) (Client, error) {
	return &MockClient{name: "mock", healthy: true}, nil
}
