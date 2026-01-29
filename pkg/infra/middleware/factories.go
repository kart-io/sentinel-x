// Package middleware 提供 HTTP 中间件的工厂注册。
// 通过 init() 函数自动注册所有内置中间件工厂。
package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/observability"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/performance"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// 确保所有工厂实现 Factory 接口。
var (
	_ mwopts.Factory = (*recoveryFactory)(nil)
	_ mwopts.Factory = (*requestIDFactory)(nil)
	_ mwopts.Factory = (*loggerFactory)(nil)
	_ mwopts.Factory = (*corsFactory)(nil)
	_ mwopts.Factory = (*timeoutFactory)(nil)
	_ mwopts.Factory = (*bodyLimitFactory)(nil)
	_ mwopts.Factory = (*metricsFactory)(nil)
	_ mwopts.Factory = (*compressionFactory)(nil)
	_ mwopts.Factory = (*securityHeadersFactory)(nil)
	_ mwopts.Factory = (*circuitBreakerFactory)(nil)
)

func init() {
	// 注册所有内置中间件工厂
	mwopts.RegisterFactory(&recoveryFactory{})
	mwopts.RegisterFactory(&requestIDFactory{})
	mwopts.RegisterFactory(&loggerFactory{})
	mwopts.RegisterFactory(&corsFactory{})
	mwopts.RegisterFactory(&timeoutFactory{})
	mwopts.RegisterFactory(&bodyLimitFactory{})
	mwopts.RegisterFactory(&metricsFactory{})
	mwopts.RegisterFactory(&compressionFactory{})
	mwopts.RegisterFactory(&securityHeadersFactory{})
	mwopts.RegisterFactory(&circuitBreakerFactory{})

	// 注册路由注册器
	mwopts.RegisterRouteRegistrar(mwopts.MiddlewareHealth, &healthRouteRegistrar{})
	mwopts.RegisterRouteRegistrar(mwopts.MiddlewareMetrics, &metricsRouteRegistrar{})
	mwopts.RegisterRouteRegistrar(mwopts.MiddlewarePprof, &pprofRouteRegistrar{})
	mwopts.RegisterRouteRegistrar(mwopts.MiddlewareVersion, &versionRouteRegistrar{})
}

// ===== Recovery Factory =====

type recoveryFactory struct{}

func (f *recoveryFactory) Name() string { return mwopts.MiddlewareRecovery }

func (f *recoveryFactory) NeedsRuntime() bool { return false }

func (f *recoveryFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.RecoveryOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for recovery: expected *RecoveryOptions, got %T", cfg)
	}
	return resilience.RecoveryWithOptions(*opts, nil), nil
}

// ===== RequestID Factory =====

type requestIDFactory struct{}

func (f *requestIDFactory) Name() string { return mwopts.MiddlewareRequestID }

func (f *requestIDFactory) NeedsRuntime() bool { return false }

func (f *requestIDFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.RequestIDOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for request-id: expected *RequestIDOptions, got %T", cfg)
	}
	return RequestIDWithOptions(*opts, nil), nil
}

// ===== Logger Factory =====

type loggerFactory struct{}

func (f *loggerFactory) Name() string { return mwopts.MiddlewareLogger }

func (f *loggerFactory) NeedsRuntime() bool { return false }

func (f *loggerFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.LoggerOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for logger: expected *LoggerOptions, got %T", cfg)
	}
	return observability.LoggerWithOptions(*opts, nil), nil
}

// ===== CORS Factory =====

type corsFactory struct{}

func (f *corsFactory) Name() string { return mwopts.MiddlewareCORS }

func (f *corsFactory) NeedsRuntime() bool { return false }

func (f *corsFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.CORSOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for cors: expected *CORSOptions, got %T", cfg)
	}
	return CORSWithOptions(*opts), nil
}

// ===== Timeout Factory =====

type timeoutFactory struct{}

func (f *timeoutFactory) Name() string { return mwopts.MiddlewareTimeout }

func (f *timeoutFactory) NeedsRuntime() bool { return false }

func (f *timeoutFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.TimeoutOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for timeout: expected *TimeoutOptions, got %T", cfg)
	}
	return TimeoutWithOptions(*opts), nil
}

// ===== BodyLimit Factory =====

type bodyLimitFactory struct{}

func (f *bodyLimitFactory) Name() string { return mwopts.MiddlewareBodyLimit }

func (f *bodyLimitFactory) NeedsRuntime() bool { return false }

func (f *bodyLimitFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.BodyLimitOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for body-limit: expected *BodyLimitOptions, got %T", cfg)
	}
	return resilience.BodyLimitWithOptions(*opts), nil
}

// ===== Metrics Factory =====

type metricsFactory struct{}

func (f *metricsFactory) Name() string { return mwopts.MiddlewareMetrics }

func (f *metricsFactory) NeedsRuntime() bool { return false }

func (f *metricsFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.MetricsOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for metrics: expected *MetricsOptions, got %T", cfg)
	}
	return MetricsMiddlewareWithOptions(*opts), nil
}

// ===== Compression Factory =====

type compressionFactory struct{}

func (f *compressionFactory) Name() string { return mwopts.MiddlewareCompression }

func (f *compressionFactory) NeedsRuntime() bool { return false }

func (f *compressionFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.CompressionOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for compression: expected *CompressionOptions, got %T", cfg)
	}
	return performance.CompressionWithOptions(*opts), nil
}

// ===== SecurityHeaders Factory =====

type securityHeadersFactory struct{}

func (f *securityHeadersFactory) Name() string { return mwopts.MiddlewareSecurityHeaders }

func (f *securityHeadersFactory) NeedsRuntime() bool { return false }

func (f *securityHeadersFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.SecurityHeadersOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for security-headers: expected *SecurityHeadersOptions, got %T", cfg)
	}
	return SecurityHeadersWithOptions(*opts), nil
}

// ===== CircuitBreaker Factory =====

type circuitBreakerFactory struct{}

func (f *circuitBreakerFactory) Name() string { return mwopts.MiddlewareCircuitBreaker }

func (f *circuitBreakerFactory) NeedsRuntime() bool { return false }

func (f *circuitBreakerFactory) Create(cfg mwopts.MiddlewareConfig) (gin.HandlerFunc, error) {
	opts, ok := cfg.(*mwopts.CircuitBreakerOptions)
	if !ok {
		return nil, fmt.Errorf("invalid config type for circuit-breaker: expected *CircuitBreakerOptions, got %T", cfg)
	}
	return resilience.CircuitBreakerWithOptions(*opts), nil
}

// ===== 需要运行时依赖的中间件（标记但不自动创建） =====

// ===== 路由注册器实现 =====

type healthRouteRegistrar struct{}

func (r *healthRouteRegistrar) RegisterRoutes(engine *gin.Engine, cfg mwopts.MiddlewareConfig) error {
	opts, ok := cfg.(*mwopts.HealthOptions)
	if !ok {
		return fmt.Errorf("invalid config type for health: expected *HealthOptions, got %T", cfg)
	}
	RegisterHealthRoutesWithOptions(engine, *opts, nil)
	return nil
}

type metricsRouteRegistrar struct{}

func (r *metricsRouteRegistrar) RegisterRoutes(engine *gin.Engine, cfg mwopts.MiddlewareConfig) error {
	opts, ok := cfg.(*mwopts.MetricsOptions)
	if !ok {
		return fmt.Errorf("invalid config type for metrics: expected *MetricsOptions, got %T", cfg)
	}
	RegisterMetricsRoutesWithOptions(engine, *opts)
	return nil
}

type pprofRouteRegistrar struct{}

func (r *pprofRouteRegistrar) RegisterRoutes(engine *gin.Engine, cfg mwopts.MiddlewareConfig) error {
	opts, ok := cfg.(*mwopts.PprofOptions)
	if !ok {
		return fmt.Errorf("invalid config type for pprof: expected *PprofOptions, got %T", cfg)
	}
	RegisterPprofRoutesWithOptions(engine, *opts)
	return nil
}

type versionRouteRegistrar struct{}

func (r *versionRouteRegistrar) RegisterRoutes(engine *gin.Engine, cfg mwopts.MiddlewareConfig) error {
	opts, ok := cfg.(*mwopts.VersionOptions)
	if !ok {
		return fmt.Errorf("invalid config type for version: expected *VersionOptions, got %T", cfg)
	}
	RegisterVersionRoutes(engine, *opts)
	return nil
}
