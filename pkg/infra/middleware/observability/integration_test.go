package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
	options "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// mockContext mocks transport.Context for testing
type mockContext struct {
	transport.Context
	req *http.Request
	res *httptest.ResponseRecorder
}

func (c *mockContext) HTTPRequest() *http.Request {
	return c.req
}

func (c *mockContext) ResponseWriter() http.ResponseWriter {
	return c.res
}

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

	// Create dummy handler
	handler := func(c transport.Context) {
		// Simulate work
		time.Sleep(10 * time.Millisecond)
		c.ResponseWriter().WriteHeader(http.StatusOK)
	}

	// Create request
	req := httptest.NewRequest("GET", "/api/test", nil)
	res := httptest.NewRecorder()
	c := &mockContext{req: req, res: res}

	// Execute middleware
	middleware(handler)(c)

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
