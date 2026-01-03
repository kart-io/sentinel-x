package component_test

import (
	"testing"

	"github.com/kart-io/sentinel-x/pkg/component"
	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/mongodb"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/postgres"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/spf13/pflag"
)

// TestConfigOptionsInterface verifies that all component options
// implement the component.ConfigOptions interface.
func TestConfigOptionsInterface(t *testing.T) {
	tests := []struct {
		name   string
		option component.ConfigOptions
	}{
		{
			name:   "MySQL Options",
			option: mysql.NewOptions(),
		},
		{
			name:   "Redis Options",
			option: redis.NewOptions(),
		},
		{
			name:   "MongoDB Options",
			option: mongodb.NewOptions(),
		},
		{
			name:   "PostgreSQL Options",
			option: postgres.NewOptions(),
		},
		{
			name:   "Etcd Options",
			option: etcd.NewOptions(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Complete method
			if err := tt.option.Complete(); err != nil {
				t.Errorf("Complete() error = %v", err)
			}

			// Test Validate method
			if err := tt.option.Validate(); err != nil {
				t.Errorf("Validate() error = %v", err)
			}

			// Test AddFlags method
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			tt.option.AddFlags(fs, "test.")

			// Verify that some flags were added by checking if FlagSet has flags
			flagCount := 0
			fs.VisitAll(func(_ *pflag.Flag) {
				flagCount++
			})
			if flagCount == 0 {
				t.Errorf("AddFlags() did not add any flags")
			}
		})
	}
}

// TestConfigOptionsComplete verifies that Complete() can be called
// multiple times without error.
func TestConfigOptionsComplete(t *testing.T) {
	opts := mysql.NewOptions()

	// First call
	if err := opts.Complete(); err != nil {
		t.Fatalf("First Complete() failed: %v", err)
	}

	// Second call should also succeed
	if err := opts.Complete(); err != nil {
		t.Fatalf("Second Complete() failed: %v", err)
	}
}

// TestConfigOptionsValidate verifies that Validate() can be called
// after Complete().
func TestConfigOptionsValidate(t *testing.T) {
	opts := redis.NewOptions()

	// Complete first
	if err := opts.Complete(); err != nil {
		t.Fatalf("Complete() failed: %v", err)
	}

	// Then validate
	if err := opts.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}
}

// TestConfigOptionsAddFlags verifies that AddFlags() properly
// populates a FlagSet.
func TestConfigOptionsAddFlags(t *testing.T) {
	tests := []struct {
		name       string
		option     component.ConfigOptions
		prefixes   []string // Optional prefixes to pass
		expectFlag string   // One expected flag name to verify
	}{
		{
			name:       "MySQL with prefix",
			option:     mysql.NewOptions(),
			prefixes:   nil, // No prefix - component name is already embedded in flag
			expectFlag: "mysql.host",
		},
		{
			name:       "Redis with prefix",
			option:     redis.NewOptions(),
			prefixes:   nil,
			expectFlag: "redis.host",
		},
		{
			name:       "MongoDB with prefix",
			option:     mongodb.NewOptions(),
			prefixes:   nil,
			expectFlag: "mongodb.host",
		},
		{
			name:       "PostgreSQL with prefix",
			option:     postgres.NewOptions(),
			prefixes:   nil,
			expectFlag: "postgres.host",
		},
		{
			name:       "Etcd with prefix",
			option:     etcd.NewOptions(),
			prefixes:   nil,
			expectFlag: "etcd.endpoints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			tt.option.AddFlags(fs, tt.prefixes...)

			// Verify the expected flag exists
			flag := fs.Lookup(tt.expectFlag)
			if flag == nil {
				t.Errorf("Expected flag %q not found", tt.expectFlag)
			}
		})
	}
}
