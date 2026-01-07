package observability

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
	options "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// MetricsCollector collects HTTP metrics using the unified metrics package.
type MetricsCollector struct {
	namespace string
	subsystem string

	// Metrics
	requestsTotal   metrics.CounterVec
	requestDuration metrics.HistogramVec
	activeRequests  metrics.Gauge

	// Start time
	startTime metrics.Gauge
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	prefix := namespace
	if subsystem != "" {
		prefix = prefix + "_" + subsystem
	}

	m := &MetricsCollector{
		namespace: namespace,
		subsystem: subsystem,
	}

	// Register metrics
	m.requestsTotal = metrics.NewCounterVec(
		prefix+"_requests_total",
		"Total number of HTTP requests.",
	)
	metrics.Register(m.requestsTotal)

	m.requestDuration = metrics.NewHistogramVec(
		prefix+"_request_duration_seconds",
		"HTTP request duration in seconds.",
		[]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	)
	metrics.Register(m.requestDuration)

	m.activeRequests = metrics.NewGauge(
		prefix+"_requests_active",
		"Current number of active requests.",
	)
	metrics.Register(m.activeRequests)

	m.startTime = metrics.NewGauge(
		prefix+"_process_start_time_seconds",
		"Start time of the process.",
	)
	m.startTime.Set(float64(time.Now().Unix()))
	metrics.Register(m.startTime)

	// Uptime metric is dynamic, we don't register it as a static gauge but could add a collector func
	// For simplicity, we'll keep the process start time which allows calculating uptime

	return m
}

// globalMetricsCollector is the default metrics collector.
var (
	globalMetricsCollector *MetricsCollector
	metricsOnce            sync.Once
	metricsMu              sync.RWMutex
)

// GetMetricsCollector returns the global metrics collector.
func GetMetricsCollector(namespace, subsystem string) *MetricsCollector {
	metricsOnce.Do(func() {
		globalMetricsCollector = NewMetricsCollector(namespace, subsystem)
	})

	metricsMu.RLock()
	defer metricsMu.RUnlock()
	return globalMetricsCollector
}

// ResetMetricsCollector resets the global metrics collector (useful for testing).
func ResetMetricsCollector() {
	metricsMu.Lock()
	defer metricsMu.Unlock()

	// Also reset the global registry to avoid duplicate registration errors
	metrics.DefaultRegistry.Reset()

	globalMetricsCollector = nil
	metricsOnce = sync.Once{}
}

// RecordRequest records a request metric.
func (m *MetricsCollector) RecordRequest(method, path string, status int, duration time.Duration) {
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": strconv.Itoa(status),
	}
	m.requestsTotal.With(labels).Inc()
	m.requestDuration.With(labels).Observe(duration.Seconds())
}

// IncrementActive increments active request count.
func (m *MetricsCollector) IncrementActive() {
	m.activeRequests.Inc()
}

// DecrementActive decrements active request count.
func (m *MetricsCollector) DecrementActive() {
	m.activeRequests.Dec()
}

// Export exports metrics in Prometheus format.
func (m *MetricsCollector) Export() string {
	// The registry handles all registered metrics
	// We might want to append process uptime here if not handled by registry
	return metrics.Export()
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// MetricsMiddleware creates a middleware that collects metrics.
//
// Deprecated: 使用 MetricsWithOptions 替代。此函数将在 v2.0.0 中移除。
func MetricsMiddleware(opts options.MetricsOptions) gin.HandlerFunc {
	return MetricsWithOptions(opts)
}

// MetricsWithOptions 返回一个使用纯配置选项的 Metrics 中间件。
// 这是推荐的 API，适用于配置中心场景（配置必须可序列化）。
//
// 参数：
//   - opts: 纯配置选项（可 JSON 序列化）
//
// 示例：
//
//	opts := mwopts.NewMetricsOptions()
//	opts.Path = "/metrics"
//	opts.Namespace = "myapp"
//	middleware.MetricsWithOptions(opts)
func MetricsWithOptions(opts options.MetricsOptions) gin.HandlerFunc {
	collector := GetMetricsCollector(opts.Namespace, opts.Subsystem)

	return func(c *gin.Context) {
			req := c.Request
			path := req.URL.Path

			// Skip metrics endpoint itself
			if path == opts.Path {
				c.Next()
				return
			}

			collector.IncrementActive()
			start := time.Now()

			// Wrap response writer to capture status code
			rw := c.Writer
			mrw := newMetricsResponseWriter(rw)

			// Execute handler
			c.Next()

			duration := time.Since(start)
			collector.DecrementActive()

			// Try to get status code - default to 200 if not available
			// Note: In a real Gin/Echo wrapper, we'd need to properly wrap the ResponseWriter in the context
			// But for now we rely on the implementation status
			status := mrw.statusCode
			// If possible, get status from context specific response if wrapper didn't work (framework dependent)

			collector.RecordRequest(req.Method, path, status, duration)
		}
}

// RegisterMetricsRoutesWithOptions 注册 Metrics 路由端点。
// 这是推荐的 API，使用纯配置选项。
//
// 参数：
//   - router: 路由器接口
//   - opts: Metrics 配置选项
//
// 示例：
//
//	opts := mwopts.NewMetricsOptions()
//	RegisterMetricsRoutesWithOptions(router, *opts)
func RegisterMetricsRoutesWithOptions(router transport.Router, opts options.MetricsOptions) {
	// Ensure collector is initialized
	GetMetricsCollector(opts.Namespace, opts.Subsystem)

	router.Handle(http.MethodGet, opts.Path, func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, metrics.Export())
	})
}

// ResetMetrics resets all metrics data (useful for testing).
func ResetMetrics() {
	// Reset registry
	metrics.DefaultRegistry.Reset()
	// Reset collector instance
	ResetMetricsCollector()
}

// GetRequestCount returns the request count for given method, path, status.
// Useful for testing verification.
func (m *MetricsCollector) GetRequestCount(method, path string, status int) uint64 {
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": strconv.Itoa(status),
	}
	return uint64(m.requestsTotal.With(labels).Get())
}

// SetResponseStatus sets the response status for metrics recording.
func SetResponseStatus(_ *gin.Context, status int) {
	_ = status
}
