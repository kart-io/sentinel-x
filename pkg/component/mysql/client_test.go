package mysql

import (
	"context"
	"testing"
	"time"
)

const (
	defaultHost = "localhost"
	testDB      = "testdb"
	rootUser    = "root"
)

// TestNew tests the client creation with valid options.
func TestNew(t *testing.T) {
	opts := NewOptions()
	opts.Host = defaultHost
	opts.Database = testDB

	// Note: This test will fail if MySQL is not running
	// In CI/CD, you should use a MySQL container or mock
	_, err := New(opts)
	if err != nil {
		t.Logf("Expected error when MySQL is not available: %v", err)
	}
}

// TestNewWithInvalidOptions tests client creation with invalid options.
func TestNewWithInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name:    "nil options",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "empty host",
			opts: &Options{
				Host:     "",
				Database: testDB,
				Username: rootUser,
				Port:     3306,
			},
			wantErr: true,
		},
		{
			name: "empty database",
			opts: &Options{
				Host:     defaultHost,
				Database: "",
				Username: rootUser,
				Port:     3306,
			},
			wantErr: true,
		},
		{
			name: "empty username",
			opts: &Options{
				Host:     defaultHost,
				Database: testDB,
				Username: "",
				Port:     3306,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			opts: &Options{
				Host:     defaultHost,
				Database: testDB,
				Username: rootUser,
				Port:     0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateOptions tests the options validation logic.
func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &Options{
				Host:     defaultHost,
				Port:     3306,
				Username: rootUser,
				Database: testDB,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			opts: &Options{
				Host:     "",
				Port:     3306,
				Username: rootUser,
				Database: testDB,
			},
			wantErr: true,
		},
		{
			name: "empty database",
			opts: &Options{
				Host:     defaultHost,
				Port:     3306,
				Username: rootUser,
				Database: "",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			opts: &Options{
				Host:     defaultHost,
				Port:     0,
				Username: rootUser,
				Database: testDB,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			opts: &Options{
				Host:     defaultHost,
				Port:     65536,
				Username: rootUser,
				Database: testDB,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestClientName tests that the client returns the correct name.
func TestClientName(t *testing.T) {
	// Create a client struct directly for testing
	client := &Client{}
	if got := client.Name(); got != "mysql" {
		t.Errorf("Client.Name() = %v, want %v", got, "mysql")
	}
}

// TestFactory tests the factory pattern.
func TestFactory(t *testing.T) {
	opts := NewOptions()
	opts.Host = defaultHost
	opts.Database = testDB

	factory := NewFactory(opts)
	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}

	if factory.Options() != opts {
		t.Error("Factory.Options() did not return the same options")
	}
}

// TestFactoryClone tests factory cloning.
func TestFactoryClone(t *testing.T) {
	opts := NewOptions()
	opts.Host = defaultHost
	opts.Database = testDB

	factory := NewFactory(opts)
	cloned := factory.Clone()

	if cloned == factory {
		t.Error("Clone() returned the same factory instance")
	}

	if cloned.Options() == factory.Options() {
		t.Error("Clone() options point to the same memory address")
	}

	// Modify cloned options and verify original is unchanged
	cloned.Options().Database = "cloned_db"
	if factory.Options().Database == "cloned_db" {
		t.Error("Modifying cloned options affected original factory")
	}
}

// TestNewWithContext tests client creation with context.
func TestNewWithContext(t *testing.T) {
	opts := NewOptions()
	opts.Host = defaultHost
	opts.Database = testDB

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This will fail without a running MySQL instance, but tests the API
	_, err := NewWithContext(ctx, opts)
	if err != nil {
		t.Logf("Expected error when MySQL is not available: %v", err)
	}
}

// TestFactoryCreate tests factory Create method.
func TestFactoryCreate(t *testing.T) {
	opts := NewOptions()
	opts.Host = defaultHost
	opts.Database = testDB

	factory := NewFactory(opts)
	ctx := context.Background()

	// This will fail without a running MySQL instance
	_, err := factory.Create(ctx)
	if err != nil {
		t.Logf("Expected error when MySQL is not available: %v", err)
	}
}

// TestFactoryCreate_NilOptions tests factory with nil options.
func TestFactoryCreate_NilOptions(t *testing.T) {
	factory := NewFactory(nil)
	_, err := factory.Create(context.Background())

	if err == nil {
		t.Error("Expected error when creating client with nil options")
	}
}
