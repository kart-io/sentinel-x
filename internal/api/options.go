// Package app provides the API server application.
package app

import (
	jwtopts "github.com/kart-io/sentinel-x/pkg/options/auth/jwt"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	mysqlopts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/spf13/pflag"
)

// Options contains all API server options.
type Options struct {
	// Server contains server configuration (HTTP/gRPC).
	Server *serveropts.Options `json:"server" mapstructure:"server"`

	// Log contains logger configuration.
	Log *logopts.Options `json:"log" mapstructure:"log"`

	// JWT contains JWT authentication configuration.
	JWT *jwtopts.Options `json:"jwt" mapstructure:"jwt"`

	// MySQL contains MySQL database configuration.
	MySQL *mysqlopts.Options `json:"mysql" mapstructure:"mysql"`

	// Redis contains Redis configuration.
	Redis *redisopts.Options `json:"redis" mapstructure:"redis"`

	// Metrics contains top-level metrics configuration.
	Metrics *middlewareopts.MetricsOptions `json:"metrics" mapstructure:"metrics"`

	// Health contains top-level health check configuration.
	Health *middlewareopts.HealthOptions `json:"health" mapstructure:"health"`

	// Pprof contains top-level pprof configuration.
	Pprof *middlewareopts.PprofOptions `json:"pprof" mapstructure:"pprof"`

	// Recovery contains top-level recovery configuration.
	Recovery *middlewareopts.RecoveryOptions `json:"recovery" mapstructure:"recovery"`

	// Logger contains top-level logger middleware configuration.
	Logger *middlewareopts.LoggerOptions `json:"logger" mapstructure:"logger"`

	// CORS contains top-level CORS middleware configuration.
	CORS *middlewareopts.CORSOptions `json:"cors" mapstructure:"cors"`

	// Timeout contains top-level timeout middleware configuration.
	Timeout *middlewareopts.TimeoutOptions `json:"timeout" mapstructure:"timeout"`

	// RequestID contains top-level request ID middleware configuration.
	RequestID *middlewareopts.RequestIDOptions `json:"request-id" mapstructure:"request-id"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		Server:    serveropts.NewOptions(),
		Log:       logopts.NewOptions(),
		JWT:       jwtopts.NewOptions(),
		MySQL:     mysqlopts.NewOptions(),
		Redis:     redisopts.NewOptions(),
		Metrics:   middlewareopts.NewMetricsOptions(),
		Health:    middlewareopts.NewHealthOptions(),
		Pprof:     middlewareopts.NewPprofOptions(),
		Recovery:  middlewareopts.NewRecoveryOptions(),
		Logger:    middlewareopts.NewLoggerOptions(),
		CORS:      middlewareopts.NewCORSOptions(),
		Timeout:   middlewareopts.NewTimeoutOptions(),
		RequestID: middlewareopts.NewRequestIDOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
	o.Log.AddFlags(fs)
	o.JWT.AddFlags(fs)
	o.MySQL.AddFlags(fs, "mysql.")
	o.Redis.AddFlags(fs, "redis.")
}

// Validate validates the options.
func (o *Options) Validate() error {
	if err := o.Log.Validate(); err != nil {
		return err
	}
	if err := o.Server.Validate(); err != nil {
		return err
	}
	if err := o.JWT.Validate(); err != nil {
		return err
	}
	if err := o.MySQL.Validate(); err != nil {
		return err
	}
	if err := o.Redis.Validate(); err != nil {
		return err
	}
	return nil
}

// Complete completes the options.
func (o *Options) Complete() error {
	// 将顶层 Metrics 配置应用到 server.http.middleware.metrics
	if o.Metrics != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Metrics == nil {
			o.Server.HTTP.Middleware.Metrics = o.Metrics
		} else {
			if o.Metrics.Path != "" {
				o.Server.HTTP.Middleware.Metrics.Path = o.Metrics.Path
			}
			if o.Metrics.Namespace != "" {
				o.Server.HTTP.Middleware.Metrics.Namespace = o.Metrics.Namespace
			}
			if o.Metrics.Subsystem != "" {
				o.Server.HTTP.Middleware.Metrics.Subsystem = o.Metrics.Subsystem
			}
		}
	}

	// 将顶层 Health 配置应用到 server.http.middleware.health
	if o.Health != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Health == nil {
			o.Server.HTTP.Middleware.Health = o.Health
		} else {
			if o.Health.Path != "" {
				o.Server.HTTP.Middleware.Health.Path = o.Health.Path
			}
			if o.Health.LivenessPath != "" {
				o.Server.HTTP.Middleware.Health.LivenessPath = o.Health.LivenessPath
			}
			if o.Health.ReadinessPath != "" {
				o.Server.HTTP.Middleware.Health.ReadinessPath = o.Health.ReadinessPath
			}
		}
	}

	// 将顶层 Pprof 配置应用到 server.http.middleware.pprof
	if o.Pprof != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Pprof == nil {
			o.Server.HTTP.Middleware.Pprof = o.Pprof
		} else {
			if o.Pprof.Prefix != "" {
				o.Server.HTTP.Middleware.Pprof.Prefix = o.Pprof.Prefix
			}
			o.Server.HTTP.Middleware.Pprof.EnableCmdline = o.Pprof.EnableCmdline
			o.Server.HTTP.Middleware.Pprof.EnableProfile = o.Pprof.EnableProfile
			o.Server.HTTP.Middleware.Pprof.EnableSymbol = o.Pprof.EnableSymbol
			o.Server.HTTP.Middleware.Pprof.EnableTrace = o.Pprof.EnableTrace
			if o.Pprof.BlockProfileRate != 0 {
				o.Server.HTTP.Middleware.Pprof.BlockProfileRate = o.Pprof.BlockProfileRate
			}
			if o.Pprof.MutexProfileFraction != 0 {
				o.Server.HTTP.Middleware.Pprof.MutexProfileFraction = o.Pprof.MutexProfileFraction
			}
		}
	}

	// 将顶层 Recovery 配置应用到 server.http.middleware.recovery
	if o.Recovery != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Recovery == nil {
			o.Server.HTTP.Middleware.Recovery = o.Recovery
		} else {
			o.Server.HTTP.Middleware.Recovery.EnableStackTrace = o.Recovery.EnableStackTrace
			if o.Recovery.OnPanic != nil {
				o.Server.HTTP.Middleware.Recovery.OnPanic = o.Recovery.OnPanic
			}
		}
	}

	// 将顶层 Logger 配置应用到 server.http.middleware.logger
	if o.Logger != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Logger == nil {
			o.Server.HTTP.Middleware.Logger = o.Logger
		} else {
			if len(o.Logger.SkipPaths) > 0 {
				o.Server.HTTP.Middleware.Logger.SkipPaths = o.Logger.SkipPaths
			}
			o.Server.HTTP.Middleware.Logger.UseStructuredLogger = o.Logger.UseStructuredLogger
			if o.Logger.Output != nil {
				o.Server.HTTP.Middleware.Logger.Output = o.Logger.Output
			}
		}
	}

	// 将顶层 CORS 配置应用到 server.http.middleware.cors
	if o.CORS != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.CORS == nil {
			o.Server.HTTP.Middleware.CORS = o.CORS
		} else {
			if len(o.CORS.AllowOrigins) > 0 {
				o.Server.HTTP.Middleware.CORS.AllowOrigins = o.CORS.AllowOrigins
			}
			if len(o.CORS.AllowMethods) > 0 {
				o.Server.HTTP.Middleware.CORS.AllowMethods = o.CORS.AllowMethods
			}
			if len(o.CORS.AllowHeaders) > 0 {
				o.Server.HTTP.Middleware.CORS.AllowHeaders = o.CORS.AllowHeaders
			}
			if len(o.CORS.ExposeHeaders) > 0 {
				o.Server.HTTP.Middleware.CORS.ExposeHeaders = o.CORS.ExposeHeaders
			}
			o.Server.HTTP.Middleware.CORS.AllowCredentials = o.CORS.AllowCredentials
			if o.CORS.MaxAge != 0 {
				o.Server.HTTP.Middleware.CORS.MaxAge = o.CORS.MaxAge
			}
		}
	}

	// 将顶层 Timeout 配置应用到 server.http.middleware.timeout
	if o.Timeout != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Timeout == nil {
			o.Server.HTTP.Middleware.Timeout = o.Timeout
		} else {
			if o.Timeout.Timeout > 0 {
				o.Server.HTTP.Middleware.Timeout.Timeout = o.Timeout.Timeout
			}
			if len(o.Timeout.SkipPaths) > 0 {
				o.Server.HTTP.Middleware.Timeout.SkipPaths = o.Timeout.SkipPaths
			}
		}
	}

	// 将顶层 RequestID 配置应用到 server.http.middleware.request-id
	if o.RequestID != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.RequestID == nil {
			o.Server.HTTP.Middleware.RequestID = o.RequestID
		} else {
			if o.RequestID.Header != "" {
				o.Server.HTTP.Middleware.RequestID.Header = o.RequestID.Header
			}
			if o.RequestID.Generator != nil {
				o.Server.HTTP.Middleware.RequestID.Generator = o.RequestID.Generator
			}
		}
	}

	if err := o.Server.Complete(); err != nil {
		return err
	}
	return nil
}
