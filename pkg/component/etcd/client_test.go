package etcd

import (
	"testing"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// TestClientImplementsStorageInterface verifies that Client implements storage.Client
func TestClientImplementsStorageInterface(_ *testing.T) {
	// This is a compile-time check
	var _ storage.Client = (*Client)(nil)
}

// TestFactoryImplementsStorageFactory verifies that Factory implements storage.Factory
func TestFactoryImplementsStorageFactory(_ *testing.T) {
	// This is a compile-time check
	var _ storage.Factory = (*Factory)(nil)
}

// TestValidateOptions tests the options validation logic
func TestValidateOptions(t *testing.T) {
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
			name: "empty endpoints",
			opts: &Options{
				Endpoints: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid dial timeout",
			opts: &Options{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid request timeout",
			opts: &Options{
				Endpoints:      []string{"localhost:2379"},
				DialTimeout:    5,
				RequestTimeout: 0,
			},
			wantErr: true,
		},
		{
			name:    "valid options",
			opts:    NewOptions(),
			wantErr: false,
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

// TestNewOptions verifies default options are valid
func TestNewOptions(t *testing.T) {
	opts := NewOptions()
	if err := validateOptions(opts); err != nil {
		t.Errorf("default options should be valid, got error: %v", err)
	}

	if len(opts.Endpoints) == 0 {
		t.Error("default options should have at least one endpoint")
	}

	if opts.DialTimeout <= 0 {
		t.Error("default dial timeout should be positive")
	}

	if opts.RequestTimeout <= 0 {
		t.Error("default request timeout should be positive")
	}

	if opts.LeaseTTL <= 0 {
		t.Error("default lease TTL should be positive")
	}
}

// TestClientName verifies the client returns correct name
func TestClientName(t *testing.T) {
	// Create a client without connecting (we'll test with nil client)
	c := &Client{}
	if name := c.Name(); name != "etcd" {
		t.Errorf("Client.Name() = %v, want %v", name, "etcd")
	}
}

// TestFactoryCreation verifies factory can be created
func TestFactoryCreation(t *testing.T) {
	opts := NewOptions()
	factory := NewFactory(opts)
	if factory == nil {
		t.Fatal("NewFactory() should not return nil")
	}
	if factory.opts != opts {
		t.Error("factory should store the provided options")
	}
}
