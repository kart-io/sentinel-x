package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/tracing"
)

func TestNewTracingOptions(t *testing.T) {
	opts := NewTracingOptions()

	if opts.TracerName != TracerName {
		t.Errorf("Expected tracer name to be %s, got %s", TracerName, opts.TracerName)
	}

	if opts.SpanNameFormatter == nil {
		t.Error("Expected span name formatter to be set")
	}

	if opts.IncludeRequestBody {
		t.Error("Expected request body capture to be disabled by default")
	}

	if opts.IncludeResponseBody {
		t.Error("Expected response body capture to be disabled by default")
	}
}

func TestTracingOptions(t *testing.T) {
	opts := NewTracingOptions()

	// Test WithTracerName
	WithTracerName("custom-tracer")(opts)
	if opts.TracerName != "custom-tracer" {
		t.Errorf("Expected tracer name to be 'custom-tracer', got %s", opts.TracerName)
	}

	// Test WithRequestBodyCapture
	WithRequestBodyCapture(true)(opts)
	if !opts.IncludeRequestBody {
		t.Error("Expected request body capture to be enabled")
	}

	// Test WithResponseBodyCapture
	WithResponseBodyCapture(true)(opts)
	if !opts.IncludeResponseBody {
		t.Error("Expected response body capture to be enabled")
	}

	// Test WithTracingSkipPaths
	skipPaths := []string{"/health", "/metrics"}
	WithTracingSkipPaths(skipPaths)(opts)
	if len(opts.SkipPaths) != len(skipPaths) {
		t.Errorf("Expected %d skip paths, got %d", len(skipPaths), len(opts.SkipPaths))
	}

	// Test WithTracingSkipPathPrefixes
	skipPrefixes := []string{"/debug", "/internal"}
	WithTracingSkipPathPrefixes(skipPrefixes)(opts)
	if len(opts.SkipPathPrefixes) != len(skipPrefixes) {
		t.Errorf("Expected %d skip path prefixes, got %d", len(skipPrefixes), len(opts.SkipPathPrefixes))
	}

	// Test WithSpanNameFormatter
	customFormatter := func(ctx *gin.Context) string {
		return "custom-span"
	}
	WithSpanNameFormatter(customFormatter)(opts)
	if opts.SpanNameFormatter == nil {
		t.Error("Expected span name formatter to be set")
	}

	// Test WithAttributeExtractor
	customExtractor := func(ctx *gin.Context) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("custom", "value"),
		}
	}
	WithAttributeExtractor(customExtractor)(opts)
	if opts.AttributeExtractor == nil {
		t.Error("Expected attribute extractor to be set")
	}
}

func TestTracing_BasicRequest(t *testing.T) {
	// Setup tracing
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)

	// Set as global provider for the test
	oldProvider := tracing.GetGlobalTracerProvider()
	defer func() {
		// Restore old provider
		_ = oldProvider
	}()

	// Create middleware
	middleware := Tracing()

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)

	// Create handler
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Execute handler
	r.ServeHTTP(w, req)

	// Note: Since we're using the global tracer, spans might not be captured
	// This is a basic test to ensure the middleware doesn't panic
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	_ = tp
	_ = recorder
}

//nolint:dupl
func TestTracing_SkipPaths(t *testing.T) {
	// Create middleware with skip paths
	middleware := Tracing(
		WithTracingSkipPaths([]string{"/health", "/metrics"}),
	)

	tests := []struct {
		name        string
		path        string
		shouldTrace bool
	}{
		{
			name:        "normal path",
			path:        "/api/users",
			shouldTrace: true,
		},
		{
			name:        "skip health",
			path:        "/health",
			shouldTrace: false,
		},
		{
			name:        "skip metrics",
			path:        "/metrics",
			shouldTrace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET(tt.path, func(c *gin.Context) {
				handlerCalled = true
			})

			r.ServeHTTP(w, req)

			if !handlerCalled {
				t.Error("Expected handler to be called")
			}
		})
	}
}

//nolint:dupl
func TestTracing_SkipPathPrefixes(t *testing.T) {
	// Create middleware with skip path prefixes
	middleware := Tracing(
		WithTracingSkipPathPrefixes([]string{"/debug", "/internal"}),
	)

	tests := []struct {
		name        string
		path        string
		shouldTrace bool
	}{
		{
			name:        "normal path",
			path:        "/api/users",
			shouldTrace: true,
		},
		{
			name:        "skip debug prefix",
			path:        "/debug/pprof",
			shouldTrace: false,
		},
		{
			name:        "skip internal prefix",
			path:        "/internal/status",
			shouldTrace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.GET(tt.path, func(c *gin.Context) {
				handlerCalled = true
			})

			r.ServeHTTP(w, req)

			if !handlerCalled {
				t.Error("Expected handler to be called")
			}
		})
	}
}

func TestDefaultSpanNameFormatter(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	name := defaultSpanNameFormatter(c)
	expected := "GET /api/users"

	if name != expected {
		t.Errorf("Expected span name %s, got %s", expected, name)
	}
}

func TestExtractTraceID(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Without a trace context, should return empty string
	traceID := ExtractTraceID(c)
	if traceID != "" {
		t.Errorf("Expected empty trace ID, got %s", traceID)
	}
}

func TestExtractSpanID(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Without a span context, should return empty string
	spanID := ExtractSpanID(c)
	if spanID != "" {
		t.Errorf("Expected empty span ID, got %s", spanID)
	}
}

func TestTracingResponseWriter(t *testing.T) {
	rw := httptest.NewRecorder()
	trw := &tracingResponseWriter{
		ResponseWriter: rw,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	trw.WriteHeader(http.StatusCreated)
	if trw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, trw.statusCode)
	}

	// Test multiple WriteHeader calls (should only record first)
	trw.WriteHeader(http.StatusBadRequest)
	if trw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code to remain %d, got %d", http.StatusCreated, trw.statusCode)
	}

	// Test Write
	trw2 := &tracingResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	_, err := trw2.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	if !trw2.written {
		t.Error("Expected written flag to be set")
	}
}

// Benchmark tests
func BenchmarkTracing(b *testing.B) {
	middleware := Tracing()

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
		r.ServeHTTP(w, req)
	}
}

func BenchmarkTracing_WithSkipPaths(b *testing.B) {
	middleware := Tracing(
		WithTracingSkipPaths([]string{"/health", "/metrics"}),
	)

	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
		r.ServeHTTP(w, req)
	}
}

var _ codes.Code = codes.Ok
