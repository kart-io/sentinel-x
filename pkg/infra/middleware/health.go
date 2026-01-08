package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// HealthStatus represents the health status.
type HealthStatus string

const (
	// HealthStatusUp indicates the service is healthy.
	HealthStatusUp HealthStatus = "UP"
	// HealthStatusDown indicates the service is unhealthy.
	HealthStatusDown HealthStatus = "DOWN"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  HealthStatus           `json:"status"`
	Checks  map[string]CheckResult `json:"checks,omitempty"`
	Version string                 `json:"version,omitempty"`
}

// CheckResult represents an individual health check result.
type CheckResult struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// HealthChecker is a function that performs a health check.
type HealthChecker func() error

// HealthManager manages health checks.
type HealthManager struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
	ready    bool
	version  string
}

// NewHealthManager creates a new health manager.
func NewHealthManager() *HealthManager {
	return &HealthManager{
		checkers: make(map[string]HealthChecker),
		ready:    true,
	}
}

// globalHealthManager is the default health manager.
var globalHealthManager = NewHealthManager()

// GetHealthManager returns the global health manager.
func GetHealthManager() *HealthManager {
	return globalHealthManager
}

// SetVersion sets the service version.
func (h *HealthManager) SetVersion(version string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.version = version
}

// RegisterChecker registers a health checker.
func (h *HealthManager) RegisterChecker(name string, checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// SetReady sets the readiness status.
func (h *HealthManager) SetReady(ready bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ready = ready
}

// IsReady returns the readiness status.
func (h *HealthManager) IsReady() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ready
}

// Check performs all health checks.
func (h *HealthManager) Check() HealthResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp := HealthResponse{
		Status:  HealthStatusUp,
		Version: h.version,
	}

	if len(h.checkers) > 0 {
		resp.Checks = make(map[string]CheckResult)
		for name, checker := range h.checkers {
			if err := checker(); err != nil {
				resp.Status = HealthStatusDown
				resp.Checks[name] = CheckResult{
					Status:  HealthStatusDown,
					Message: err.Error(),
				}
			} else {
				resp.Checks[name] = CheckResult{
					Status: HealthStatusUp,
				}
			}
		}
	}

	return resp
}

// RegisterHealthRoutesWithOptions 注册 Health 路由端点。
// 这是推荐的 API，使用纯配置选项和运行时依赖注入。
//
// 参数：
//   - engine: Gin 引擎
//   - opts: Health 配置选项（纯配置，可 JSON 序列化）
//   - checker: 可选的自定义健康检查函数（运行时依赖）
//
// 示例：
//
//	opts := mwopts.NewHealthOptions()
//	RegisterHealthRoutesWithOptions(engine, *opts, func() error {
//	    // 自定义健康检查逻辑
//	    return nil
//	})
func RegisterHealthRoutesWithOptions(engine *gin.Engine, opts mwopts.HealthOptions, checker func() error) {
	manager := GetHealthManager()

	// Register custom checker if provided
	if checker != nil {
		manager.RegisterChecker("custom", checker)
	}

	// Health check endpoint
	if opts.Path != "" {
		engine.GET(opts.Path, func(c *gin.Context) {
			resp := manager.Check()
			status := http.StatusOK
			if resp.Status == HealthStatusDown {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, resp)
		})
	}

	// Liveness probe - always returns OK if the process is running
	if opts.LivenessPath != "" {
		engine.GET(opts.LivenessPath, func(c *gin.Context) {
			c.JSON(http.StatusOK, HealthResponse{
				Status: HealthStatusUp,
			})
		})
	}

	// Readiness probe - returns OK only if service is ready
	if opts.ReadinessPath != "" {
		engine.GET(opts.ReadinessPath, func(c *gin.Context) {
			if manager.IsReady() {
				resp := manager.Check()
				status := http.StatusOK
				if resp.Status == HealthStatusDown {
					status = http.StatusServiceUnavailable
				}
				c.JSON(status, resp)
			} else {
				c.JSON(http.StatusServiceUnavailable, HealthResponse{
					Status: HealthStatusDown,
				})
			}
		})
	}
}
