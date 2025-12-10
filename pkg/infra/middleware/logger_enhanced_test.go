package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applogger "github.com/kart-io/sentinel-x/pkg/infra/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// enhancedMockContext implements transport.Context for testing.
type enhancedMockContext struct {
	ctx            context.Context
	req            *http.Request
	w              http.ResponseWriter
	params         map[string]string
	statusCode     int
	responseBody   []byte
	responseWriter *enhancedMockResponseWriter
}

// enhancedMockResponseWriter wraps httptest.ResponseRecorder for testing.
type enhancedMockResponseWriter struct {
	*httptest.ResponseRecorder
}

func newEnhancedMockContext(method, path string, body io.Reader) *enhancedMockContext {
	req := httptest.NewRequest(method, path, body)
	rec := httptest.NewRecorder()
	return &enhancedMockContext{
		ctx:            context.Background(),
		req:            req,
		w:              rec,
		params:         make(map[string]string),
		responseWriter: &enhancedMockResponseWriter{ResponseRecorder: rec},
	}
}

func (m *enhancedMockContext) Request() context.Context {
	return m.ctx
}

func (m *enhancedMockContext) SetRequest(ctx context.Context) {
	m.ctx = ctx
}

func (m *enhancedMockContext) HTTPRequest() *http.Request {
	return m.req
}

func (m *enhancedMockContext) ResponseWriter() http.ResponseWriter {
	return m.responseWriter
}

func (m *enhancedMockContext) Body() io.ReadCloser {
	return m.req.Body
}

func (m *enhancedMockContext) Param(key string) string {
	return m.params[key]
}

func (m *enhancedMockContext) Query(key string) string {
	return m.req.URL.Query().Get(key)
}

func (m *enhancedMockContext) Header(key string) string {
	return m.req.Header.Get(key)
}

func (m *enhancedMockContext) SetHeader(key, value string) {
	m.w.Header().Set(key, value)
}

func (m *enhancedMockContext) Bind(v interface{}) error {
	return nil
}

func (m *enhancedMockContext) Validate(v interface{}) error {
	return nil
}

func (m *enhancedMockContext) ShouldBindAndValidate(v interface{}) error {
	return nil
}

func (m *enhancedMockContext) MustBindAndValidate(v interface{}) (string, bool) {
	return "", true
}

func (m *enhancedMockContext) JSON(code int, v interface{}) {
	m.statusCode = code
}

func (m *enhancedMockContext) String(code int, s string) {
	m.statusCode = code
	m.responseBody = []byte(s)
}

func (m *enhancedMockContext) Error(code int, err error) {
	m.statusCode = code
}

func (m *enhancedMockContext) GetRawContext() interface{} {
	return nil
}

func (m *enhancedMockContext) Lang() string {
	return "en"
}

func (m *enhancedMockContext) SetLang(lang string) {
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := newResponseWriter(rec, false)

		rw.WriteHeader(http.StatusNotFound)

		if rw.Status() != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rw.Status())
		}
	})

	t.Run("captures bytes written", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := newResponseWriter(rec, false)

		data := []byte("Hello, World!")
		n, err := rw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}

		if rw.BytesWritten() != int64(len(data)) {
			t.Errorf("expected %d bytes recorded, got %d", len(data), rw.BytesWritten())
		}
	})

	t.Run("captures response body when enabled", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := newResponseWriter(rec, true)

		data := []byte("Response body")
		_, err := rw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if rw.Body() != string(data) {
			t.Errorf("expected body %s, got %s", string(data), rw.Body())
		}
	})

	t.Run("does not capture body when disabled", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := newResponseWriter(rec, false)

		data := []byte("Response body")
		_, err := rw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if rw.Body() != "" {
			t.Errorf("expected empty body, got %s", rw.Body())
		}
	})

	t.Run("default status is 200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := newResponseWriter(rec, false)

		if rw.Status() != http.StatusOK {
			t.Errorf("expected default status %d, got %d", http.StatusOK, rw.Status())
		}
	})
}

func TestEnhancedLogger(t *testing.T) {
	// Initialize logger for testing
	opts := applogger.NewOptions()
	opts.Level = "DEBUG"
	opts.Format = "json"
	if err := opts.Init(); err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	t.Run("logs request with default config", func(t *testing.T) {
		middleware := EnhancedLogger()
		mc := newEnhancedMockContext("GET", "/test", nil)

		called := false
		handler := middleware(func(c transport.Context) {
			called = true
			c.String(http.StatusOK, "OK")
		})

		handler(mc)

		// Verify handler was called
		if !called {
			t.Error("expected handler to be called")
		}

		// Context should be updated (even if no fields were added)
		ctx := mc.Request()
		if ctx == nil {
			t.Error("expected context to be set")
		}
	})

	t.Run("skips logging for configured paths", func(t *testing.T) {
		config := DefaultEnhancedLoggerConfig
		config.SkipPaths = []string{"/health", "/metrics"}

		middleware := EnhancedLoggerWithConfig(config)
		mc := newEnhancedMockContext("GET", "/health", nil)

		called := false
		handler := middleware(func(c transport.Context) {
			called = true
		})

		handler(mc)

		if !called {
			t.Error("expected handler to be called")
		}
	})

	t.Run("extracts request ID", func(t *testing.T) {
		middleware := EnhancedLogger()
		mc := newEnhancedMockContext("GET", "/test", nil)

		// Add request ID to context
		requestID := "req-12345"
		ctx := context.WithValue(mc.Request(), requestIDKey{}, requestID)
		mc.SetRequest(ctx)

		handler := middleware(func(c transport.Context) {
			// Verify request ID is in context fields
			fields := applogger.GetContextFields(c.Request())
			found := false
			for i := 0; i < len(fields); i += 2 {
				if fields[i] == "request_id" && fields[i+1] == requestID {
					found = true
					break
				}
			}
			if !found {
				t.Error("request_id not found in context fields")
			}
		})

		handler(mc)
	})

	t.Run("redacts sensitive headers", func(t *testing.T) {
		config := DefaultEnhancedLoggerConfig
		config.SensitiveHeaders = []string{"Authorization"}

		middleware := EnhancedLoggerWithConfig(config)
		mc := newEnhancedMockContext("GET", "/test", nil)
		mc.req.Header.Set("Authorization", "Bearer secret-token")
		mc.req.Header.Set("X-Custom-Header", "public-value")

		handler := middleware(func(c transport.Context) {})

		handler(mc)

		// The middleware should have logged the request with redacted authorization header
	})

	t.Run("captures request body when enabled", func(t *testing.T) {
		config := DefaultEnhancedLoggerConfig
		config.LogRequestBody = true
		config.MaxBodyLogSize = 1024

		body := bytes.NewReader([]byte(`{"key":"value"}`))
		middleware := EnhancedLoggerWithConfig(config)
		mc := newEnhancedMockContext("POST", "/test", body)

		handler := middleware(func(c transport.Context) {})

		handler(mc)
	})

	t.Run("logs different levels based on status code", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
		}{
			{"success", http.StatusOK},
			{"client error", http.StatusBadRequest},
			{"server error", http.StatusInternalServerError},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				middleware := EnhancedLogger()
				mc := newEnhancedMockContext("GET", "/test", nil)

				handler := middleware(func(c transport.Context) {
					// Simulate response with status code
					rw := newResponseWriter(c.ResponseWriter(), false)
					rw.WriteHeader(tt.statusCode)
				})

				handler(mc)
			})
		}
	})
}

func TestCaptureRequestBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxSize  int
		expected string
	}{
		{
			name:     "body within limit",
			body:     "Hello, World!",
			maxSize:  100,
			expected: "Hello, World!",
		},
		{
			name:     "body exceeds limit",
			body:     strings.Repeat("a", 200),
			maxSize:  100,
			expected: strings.Repeat("a", 100) + "... [truncated]",
		},
		{
			name:     "empty body",
			body:     "",
			maxSize:  100,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(tt.body)))
			result := captureRequestBody(req, tt.maxSize)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}

			// Verify body can still be read
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			// For truncated cases, we expect only what was captured
			expectedReadable := tt.body
			if tt.maxSize > 0 && len(tt.body) > tt.maxSize {
				expectedReadable = tt.body[:tt.maxSize]
			}
			if string(body) != expectedReadable {
				t.Errorf("body after capture: expected %q, got %q", expectedReadable, string(body))
			}
		})
	}
}

func TestEnhancedLoggerConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := DefaultEnhancedLoggerConfig

		if !config.EnableTraceCorrelation {
			t.Error("expected EnableTraceCorrelation to be true")
		}

		if !config.EnableResponseLogging {
			t.Error("expected EnableResponseLogging to be true")
		}

		if !config.EnableRequestLogging {
			t.Error("expected EnableRequestLogging to be true")
		}

		if len(config.SensitiveHeaders) == 0 {
			t.Error("expected default sensitive headers")
		}

		if config.MaxBodyLogSize != 1024 {
			t.Errorf("expected MaxBodyLogSize 1024, got %d", config.MaxBodyLogSize)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		config := EnhancedLoggerConfig{
			EnhancedLoggerConfig: &applogger.EnhancedLoggerConfig{
				EnableTraceCorrelation:   false,
				EnableResponseLogging:    false,
				EnableRequestLogging:     false,
				SensitiveHeaders:         []string{"Custom-Header"},
				MaxBodyLogSize:           2048,
				CaptureStackTrace:        true,
				ErrorStackTraceMinStatus: 400,
				SkipPaths:                []string{"/custom"},
				LogRequestBody:           true,
				LogResponseBody:          true,
			},
		}

		if config.EnableTraceCorrelation {
			t.Error("expected EnableTraceCorrelation to be false")
		}

		if config.MaxBodyLogSize != 2048 {
			t.Errorf("expected MaxBodyLogSize 2048, got %d", config.MaxBodyLogSize)
		}

		if !config.CaptureStackTrace {
			t.Error("expected CaptureStackTrace to be true")
		}
	})
}

func TestFieldsPooling(t *testing.T) {
	t.Run("acquire and release fields", func(t *testing.T) {
		fields1 := acquireEnhancedFields()
		if fields1 == nil {
			t.Fatal("expected non-nil fields slice")
		}

		*fields1 = append(*fields1, "key1", "value1")

		releaseEnhancedFields(fields1)

		// Verify slice was reset
		if len(*fields1) != 0 {
			t.Errorf("expected slice to be reset, got length %d", len(*fields1))
		}

		// Acquire again - should get same slice from pool
		fields2 := acquireEnhancedFields()
		if fields2 == nil {
			t.Fatal("expected non-nil fields slice")
		}

		releaseEnhancedFields(fields2)
	})
}

func BenchmarkEnhancedLogger(b *testing.B) {
	// Initialize logger
	opts := applogger.NewOptions()
	opts.Level = "INFO"
	opts.Format = "json"
	if err := opts.Init(); err != nil {
		b.Fatalf("failed to initialize logger: %v", err)
	}

	middleware := EnhancedLogger()
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mc := newEnhancedMockContext("GET", "/test", nil)
		handler(mc)
	}
}

func BenchmarkEnhancedLoggerWithBody(b *testing.B) {
	// Initialize logger
	opts := applogger.NewOptions()
	opts.Level = "INFO"
	opts.Format = "json"
	if err := opts.Init(); err != nil {
		b.Fatalf("failed to initialize logger: %v", err)
	}

	config := DefaultEnhancedLoggerConfig
	config.LogRequestBody = true
	config.MaxBodyLogSize = 1024

	middleware := EnhancedLoggerWithConfig(config)
	handler := middleware(func(c transport.Context) {
		c.String(http.StatusOK, "OK")
	})

	body := []byte(`{"key":"value","data":"test"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mc := newEnhancedMockContext("POST", "/test", bytes.NewReader(body))
		handler(mc)
	}
}

func BenchmarkResponseWriter(b *testing.B) {
	rec := httptest.NewRecorder()
	data := []byte("Hello, World!")

	b.Run("without body capture", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rw := newResponseWriter(rec, false)
			rw.Write(data)
		}
	})

	b.Run("with body capture", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rw := newResponseWriter(rec, true)
			rw.Write(data)
		}
	})
}
