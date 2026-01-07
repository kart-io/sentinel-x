package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
	options "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

func TestMetricsMiddlewareIntegration(t *testing.T) {
	// Reset registry
	metrics.DefaultRegistry.Reset()
	ResetMetricsCollector()

	// Setup middleware
	opts := options.MetricsOptions{
		Namespace: "test_service",
		Subsystem: "http",
		Path:      "/metrics",
	}
	middleware := MetricsMiddleware(opts)

	// Create request
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)

	// Create dummy handler
	r.GET("/api/test", func(c *gin.Context) {
		// Simulate work
		time.Sleep(10 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	// Execute middleware
	r.ServeHTTP(w, req)

	// Check collector state directly
	collector := GetMetricsCollector(opts.Namespace, opts.Subsystem)
	count := collector.GetRequestCount("GET", "/api/test", 200)
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Check Registry Export
	out := metrics.Export()

	// Check for metric presence in Prometheus output
	expectedMetric := `test_service_http_requests_total{method="GET",path="/api/test",status="200"} 1`
	if !strings.Contains(out, expectedMetric) {
		t.Errorf("expected metric %s in output, got:\n%s", expectedMetric, out)
	}
}
