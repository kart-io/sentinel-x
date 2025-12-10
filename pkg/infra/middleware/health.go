package middleware

import (
	"net/http"
	"sync"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
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

// RegisterHealthRoutes registers health check routes.
func RegisterHealthRoutes(router transport.Router, opts HealthOptions) {
	manager := GetHealthManager()

	// Register custom checker if provided
	if opts.Checker != nil {
		manager.RegisterChecker("custom", opts.Checker)
	}

	// Health check endpoint
	if opts.Path != "" {
		router.Handle(http.MethodGet, opts.Path, func(c transport.Context) {
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
		router.Handle(http.MethodGet, opts.LivenessPath, func(c transport.Context) {
			c.JSON(http.StatusOK, HealthResponse{
				Status: HealthStatusUp,
			})
		})
	}

	// Readiness probe - returns OK only if service is ready
	if opts.ReadinessPath != "" {
		router.Handle(http.MethodGet, opts.ReadinessPath, func(c transport.Context) {
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
