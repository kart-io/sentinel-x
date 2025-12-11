package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/mongodb"
	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// TestClientImplementsStorageInterface verifies that mongodb.Client implements storage.Client
func TestClientImplementsStorageInterface(t *testing.T) {
	var _ storage.Client = (*mongodb.Client)(nil)
}

// TestFactoryImplementsStorageFactory verifies that mongodb.Factory implements storage.Factory
func TestFactoryImplementsStorageFactory(t *testing.T) {
	var _ storage.Factory = (*mongodb.Factory)(nil)
}

// Example demonstrates basic usage of the MongoDB client
func ExampleNew() {
	// Create options
	opts := mongodb.NewOptions()
	opts.Host = "localhost"
	opts.Port = 27017
	opts.Database = "testdb"

	// Create client (this will fail if no MongoDB is running)
	client, err := mongodb.New(opts)
	if err != nil {
		// Handle error
		return
	}
	defer func() { _ = client.Close() }()

	// Use the client
	_ = client.Name() // Returns "mongodb"
}

// Example demonstrates factory usage
func ExampleNewFactory() {
	opts := mongodb.NewOptions()
	opts.Host = "localhost"
	opts.Database = "myapp"

	factory := mongodb.NewFactory(opts)
	client, err := factory.Create(context.Background())
	if err != nil {
		// Handle error
		return
	}
	defer func() { _ = client.Close() }()
}

// TestOptionsDefaults verifies default option values
func TestOptionsDefaults(t *testing.T) {
	opts := mongodb.NewOptions()

	if opts.Host != "127.0.0.1" {
		t.Errorf("expected default host 127.0.0.1, got %s", opts.Host)
	}

	if opts.Port != 27017 {
		t.Errorf("expected default port 27017, got %d", opts.Port)
	}

	if opts.MaxPoolSize != 100 {
		t.Errorf("expected default max pool size 100, got %d", opts.MaxPoolSize)
	}

	if opts.MinPoolSize != 10 {
		t.Errorf("expected default min pool size 10, got %d", opts.MinPoolSize)
	}

	if opts.ConnectTimeout != 10*time.Second {
		t.Errorf("expected default connect timeout 10s, got %v", opts.ConnectTimeout)
	}

	if opts.AuthSource != "admin" {
		t.Errorf("expected default auth source admin, got %s", opts.AuthSource)
	}
}

// TestFactoryClone verifies factory cloning
func TestFactoryClone(t *testing.T) {
	opts := mongodb.NewOptions()
	opts.Database = "original"

	factory := mongodb.NewFactory(opts)
	cloned := factory.Clone()

	// Modify cloned options
	cloned.Options().Database = "cloned"

	// Verify original is unchanged
	if factory.Options().Database != "original" {
		t.Errorf("original factory database should be 'original', got %s", factory.Options().Database)
	}

	if cloned.Options().Database != "cloned" {
		t.Errorf("cloned factory database should be 'cloned', got %s", cloned.Options().Database)
	}
}
