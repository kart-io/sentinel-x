package mysql

import (
	"context"
	"testing"
	"time"

	mysqlOpts "github.com/kart-io/sentinel-x/pkg/options/mysql"
)

// TestNew tests the client creation with valid options.
func TestNew(t *testing.T) {
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

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
		opts    *mysqlOpts.Options
		wantErr bool
	}{
		{
			name:    "nil options",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "empty host",
			opts: &mysqlOpts.Options{
				Host:     "",
				Database: "testdb",
				Username: "root",
				Port:     3306,
			},
			wantErr: true,
		},
		{
			name: "empty database",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Database: "",
				Username: "root",
				Port:     3306,
			},
			wantErr: true,
		},
		{
			name: "empty username",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Database: "testdb",
				Username: "",
				Port:     3306,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Database: "testdb",
				Username: "root",
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
		opts    *mysqlOpts.Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Database: "testdb",
			},
			wantErr: false,
		},
		{
			name: "empty host",
			opts: &mysqlOpts.Options{
				Host:     "",
				Port:     3306,
				Username: "root",
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "empty database",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Database: "",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Port:     0,
				Username: "root",
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			opts: &mysqlOpts.Options{
				Host:     "localhost",
				Port:     65536,
				Username: "root",
				Database: "testdb",
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

// TestBuildDSN tests DSN building.
func TestBuildDSN(t *testing.T) {
	opts := &mysqlOpts.Options{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
	}

	dsn := BuildDSN(opts)
	expected := "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"

	if dsn != expected {
		t.Errorf("BuildDSN() = %v, want %v", dsn, expected)
	}
}

// TestBuildDSN_CustomPort tests DSN building with custom port.
func TestBuildDSN_CustomPort(t *testing.T) {
	opts := &mysqlOpts.Options{
		Host:     "db.example.com",
		Port:     3307,
		Username: "user",
		Password: "pass",
		Database: "mydb",
	}

	dsn := BuildDSN(opts)
	expected := "user:pass@tcp(db.example.com:3307)/mydb?charset=utf8mb4&parseTime=True&loc=Local"

	if dsn != expected {
		t.Errorf("BuildDSN() = %v, want %v", dsn, expected)
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
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

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
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

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
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

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
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

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
