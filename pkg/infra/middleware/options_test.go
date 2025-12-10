package middleware

import (
	"strings"
	"testing"
	"time"
)

func TestOptionsValidate_DefaultOptions(t *testing.T) {
	opts := NewOptions()
	if err := opts.Validate(); err != nil {
		t.Errorf("NewOptions() should create valid options, got error: %v", err)
	}
}

func TestOptionsValidate_TimeoutEnabled(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		disable bool
		wantErr bool
	}{
		{
			name:    "valid timeout",
			timeout: 30 * time.Second,
			disable: false,
			wantErr: false,
		},
		{
			name:    "zero timeout",
			timeout: 0,
			disable: false,
			wantErr: true,
		},
		{
			name:    "negative timeout",
			timeout: -1 * time.Second,
			disable: false,
			wantErr: true,
		},
		{
			name:    "timeout disabled - invalid value ignored",
			timeout: 0,
			disable: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			opts.DisableTimeout = tt.disable
			opts.Timeout.Timeout = tt.timeout

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && !strings.Contains(err.Error(), "timeout") {
				t.Errorf("Expected error to mention 'timeout', got: %v", err)
			}
		})
	}
}

func TestOptionsValidate_CORSEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid CORS config",
			setup: func(o *Options) {
				o.DisableCORS = false
				o.CORS.AllowOrigins = []string{"*"}
				o.CORS.AllowMethods = []string{"GET", "POST"}
			},
			wantErr: false,
		},
		{
			name: "no allowed origins",
			setup: func(o *Options) {
				o.DisableCORS = false
				o.CORS.AllowOrigins = []string{}
				o.CORS.AllowMethods = []string{"GET"}
			},
			wantErr: true,
			errMsg:  "origin",
		},
		{
			name: "no allowed methods",
			setup: func(o *Options) {
				o.DisableCORS = false
				o.CORS.AllowOrigins = []string{"*"}
				o.CORS.AllowMethods = []string{}
			},
			wantErr: true,
			errMsg:  "method",
		},
		{
			name: "negative max age",
			setup: func(o *Options) {
				o.DisableCORS = false
				o.CORS.AllowOrigins = []string{"*"}
				o.CORS.AllowMethods = []string{"GET"}
				o.CORS.MaxAge = -1
			},
			wantErr: true,
			errMsg:  "max age",
		},
		{
			name: "CORS disabled - invalid config ignored",
			setup: func(o *Options) {
				o.DisableCORS = true
				o.CORS.AllowOrigins = []string{}
				o.CORS.AllowMethods = []string{}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_AuthEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "auth enabled with empty token lookup",
			setup: func(o *Options) {
				o.DisableAuth = false
				o.Auth.TokenLookup = ""
			},
			wantErr: true,
			errMsg:  "token lookup",
		},
		{
			name: "auth enabled with empty auth scheme",
			setup: func(o *Options) {
				o.DisableAuth = false
				o.Auth.AuthScheme = ""
			},
			wantErr: true,
			errMsg:  "auth scheme",
		},
		{
			name: "auth disabled - no authenticator required",
			setup: func(o *Options) {
				o.DisableAuth = true
				o.Auth.Authenticator = nil
			},
			wantErr: false,
		},
		{
			name: "auth enabled without authenticator - no error (runtime check)",
			setup: func(o *Options) {
				o.DisableAuth = false
				o.Auth.Authenticator = nil
				// TokenLookup and AuthScheme have defaults
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_AuthzEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "authz disabled - no authorizer required",
			setup: func(o *Options) {
				o.DisableAuthz = true
				o.Authz.Authorizer = nil
			},
			wantErr: false,
		},
		{
			name: "authz enabled without authorizer - no error (runtime check)",
			setup: func(o *Options) {
				o.DisableAuthz = false
				o.Authz.Authorizer = nil
				// Extractors are validated at runtime
			},
			wantErr: false,
		},
		{
			name: "authz enabled without resource extractor - no error (runtime check)",
			setup: func(o *Options) {
				o.DisableAuthz = false
				o.Authz.ResourceExtractor = nil
				// Runtime validation in middleware
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_HealthEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "health with all paths empty",
			setup: func(o *Options) {
				o.DisableHealth = false
				o.Health.Path = ""
				o.Health.LivenessPath = ""
				o.Health.ReadinessPath = ""
			},
			wantErr: true,
			errMsg:  "health check path",
		},
		{
			name: "health with at least one path",
			setup: func(o *Options) {
				o.DisableHealth = false
				o.Health.Path = "/health"
			},
			wantErr: false,
		},
		{
			name: "health disabled - empty paths ignored",
			setup: func(o *Options) {
				o.DisableHealth = true
				o.Health.Path = ""
				o.Health.LivenessPath = ""
				o.Health.ReadinessPath = ""
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_MetricsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "metrics enabled with empty path",
			setup: func(o *Options) {
				o.DisableMetrics = false
				o.Metrics.Path = ""
			},
			wantErr: true,
			errMsg:  "metrics path",
		},
		{
			name: "metrics enabled with valid path",
			setup: func(o *Options) {
				o.DisableMetrics = false
				o.Metrics.Path = "/metrics"
			},
			wantErr: false,
		},
		{
			name: "metrics disabled - empty path ignored",
			setup: func(o *Options) {
				o.DisableMetrics = true
				o.Metrics.Path = ""
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_PprofEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "pprof enabled with empty prefix",
			setup: func(o *Options) {
				o.DisablePprof = false
				o.Pprof.Prefix = ""
			},
			wantErr: true,
			errMsg:  "pprof prefix",
		},
		{
			name: "pprof enabled with negative block profile rate",
			setup: func(o *Options) {
				o.DisablePprof = false
				o.Pprof.Prefix = "/debug/pprof"
				o.Pprof.BlockProfileRate = -1
			},
			wantErr: true,
			errMsg:  "block profile rate",
		},
		{
			name: "pprof enabled with negative mutex profile fraction",
			setup: func(o *Options) {
				o.DisablePprof = false
				o.Pprof.Prefix = "/debug/pprof"
				o.Pprof.MutexProfileFraction = -1
			},
			wantErr: true,
			errMsg:  "mutex profile fraction",
		},
		{
			name: "pprof disabled - invalid config ignored",
			setup: func(o *Options) {
				o.DisablePprof = true
				o.Pprof.Prefix = ""
				o.Pprof.BlockProfileRate = -1
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_RequestIDEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Options)
		wantErr bool
		errMsg  string
	}{
		{
			name: "request ID enabled with empty header",
			setup: func(o *Options) {
				o.DisableRequestID = false
				o.RequestID.Header = ""
			},
			wantErr: true,
			errMsg:  "header",
		},
		{
			name: "request ID enabled with valid header",
			setup: func(o *Options) {
				o.DisableRequestID = false
				o.RequestID.Header = "X-Request-ID"
			},
			wantErr: false,
		},
		{
			name: "request ID disabled - empty header ignored",
			setup: func(o *Options) {
				o.DisableRequestID = true
				o.RequestID.Header = ""
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.setup(opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestOptionsValidate_MultipleErrors(t *testing.T) {
	opts := NewOptions()
	// Enable features with invalid configurations
	opts.DisableTimeout = false
	opts.Timeout.Timeout = -1
	opts.DisableCORS = false
	opts.CORS.AllowOrigins = []string{}
	opts.CORS.AllowMethods = []string{}

	err := opts.Validate()
	if err == nil {
		t.Fatal("Expected validation error for multiple invalid configurations")
	}

	// Should report multiple errors
	errStr := err.Error()
	if !strings.Contains(errStr, "timeout") {
		t.Errorf("Expected error to mention 'timeout', got: %v", err)
	}
	if !strings.Contains(errStr, "cors") {
		t.Errorf("Expected error to mention 'cors', got: %v", err)
	}
}

func TestOptionsComplete(t *testing.T) {
	opts := &Options{}
	if err := opts.Complete(); err != nil {
		t.Errorf("Complete() should not return error, got: %v", err)
	}

	if opts.Logger.Output == nil {
		t.Error("Complete() should set default logger output")
	}
}

// Benchmark validation performance
func BenchmarkOptionsValidate(b *testing.B) {
	opts := NewOptions()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = opts.Validate()
	}
}
