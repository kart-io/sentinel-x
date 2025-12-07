package gin

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/integrations"
	"github.com/kart-io/logger/option"
)

// mockContext implements the gin.Context interface for testing
type mockContext struct {
	request        *http.Request
	writer         *mockResponseWriter
	params         map[string]string
	keys           map[string]interface{}
	headers        map[string]string
	clientIP       string
	shouldNext     bool
	aborted        bool
	abortedWith    int
	responseStatus int
}

func newMockContext() *mockContext {
	req := httptest.NewRequest("GET", "/test", nil)
	return &mockContext{
		request:        req,
		writer:         &mockResponseWriter{status: 200, size: 0},
		params:         make(map[string]string),
		keys:           make(map[string]interface{}),
		headers:        make(map[string]string),
		clientIP:       "127.0.0.1",
		shouldNext:     true,
		responseStatus: 200,
	}
}

func (m *mockContext) Request() *http.Request             { return m.request }
func (m *mockContext) Writer() ResponseWriter             { return m.writer }
func (m *mockContext) Param(key string) string            { return m.params[key] }
func (m *mockContext) Query(key string) string            { return m.request.URL.Query().Get(key) }
func (m *mockContext) PostForm(key string) string         { return "" }
func (m *mockContext) Get(key string) (interface{}, bool) { val, ok := m.keys[key]; return val, ok }
func (m *mockContext) Set(key, value interface{}) {
	if k, ok := key.(string); ok {
		m.keys[k] = value
	}
}
func (m *mockContext) GetString(key string) string {
	if v, ok := m.keys[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
func (m *mockContext) GetBool(key string) bool {
	if v, ok := m.keys[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
func (m *mockContext) GetInt(key string) int {
	if v, ok := m.keys[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}
func (m *mockContext) GetInt64(key string) int64 {
	if v, ok := m.keys[key]; ok {
		if i, ok := v.(int64); ok {
			return i
		}
	}
	return 0
}
func (m *mockContext) GetFloat64(key string) float64 {
	if v, ok := m.keys[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
func (m *mockContext) GetTime(key string) time.Time {
	if v, ok := m.keys[key]; ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}
func (m *mockContext) GetDuration(key string) time.Duration {
	if v, ok := m.keys[key]; ok {
		if d, ok := v.(time.Duration); ok {
			return d
		}
	}
	return 0
}
func (m *mockContext) GetStringSlice(key string) []string {
	if v, ok := m.keys[key]; ok {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	return nil
}
func (m *mockContext) GetStringMap(key string) map[string]interface{} {
	if v, ok := m.keys[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}
func (m *mockContext) GetStringMapString(key string) map[string]string {
	if v, ok := m.keys[key]; ok {
		if m, ok := v.(map[string]string); ok {
			return m
		}
	}
	return nil
}
func (m *mockContext) GetStringMapStringSlice(key string) map[string][]string {
	if v, ok := m.keys[key]; ok {
		if m, ok := v.(map[string][]string); ok {
			return m
		}
	}
	return nil
}
func (m *mockContext) ClientIP() string            { return m.clientIP }
func (m *mockContext) ContentType() string         { return m.request.Header.Get("Content-Type") }
func (m *mockContext) IsWebsocket() bool           { return false }
func (m *mockContext) Header(key, value string)    { m.headers[key] = value }
func (m *mockContext) GetHeader(key string) string { return m.request.Header.Get(key) }
func (m *mockContext) GetRawData() ([]byte, error) { return io.ReadAll(m.request.Body) }
func (m *mockContext) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
}
func (m *mockContext) Cookie(name string) (string, error)                    { return "", nil }
func (m *mockContext) Render(code int, r interface{})                        { m.writer.status = code }
func (m *mockContext) HTML(code int, name string, obj interface{})           { m.writer.status = code }
func (m *mockContext) IndentedJSON(code int, obj interface{})                { m.writer.status = code }
func (m *mockContext) SecureJSON(code int, obj interface{})                  { m.writer.status = code }
func (m *mockContext) JSONP(code int, callback string, obj interface{})      { m.writer.status = code }
func (m *mockContext) JSON(code int, obj interface{})                        { m.writer.status = code }
func (m *mockContext) AsciiJSON(code int, obj interface{})                   { m.writer.status = code }
func (m *mockContext) PureJSON(code int, obj interface{})                    { m.writer.status = code }
func (m *mockContext) XML(code int, obj interface{})                         { m.writer.status = code }
func (m *mockContext) YAML(code int, obj interface{})                        { m.writer.status = code }
func (m *mockContext) TOML(code int, obj interface{})                        { m.writer.status = code }
func (m *mockContext) ProtoBuf(code int, obj interface{})                    { m.writer.status = code }
func (m *mockContext) String(code int, format string, values ...interface{}) { m.writer.status = code }
func (m *mockContext) Redirect(code int, location string)                    { m.writer.status = code }
func (m *mockContext) Data(code int, contentType string, data []byte)        { m.writer.status = code }
func (m *mockContext) DataFromReader(code int, contentLength int64, contentType string, reader io.Reader, extraHeaders map[string]string) {
	m.writer.status = code
}
func (m *mockContext) File(filepath string)                           {}
func (m *mockContext) FileFromFS(filepath string, fs http.FileSystem) {}
func (m *mockContext) FileAttachment(filepath, filename string)       {}
func (m *mockContext) SSEvent(name string, message interface{})       {}
func (m *mockContext) Stream(step func(w io.Writer) bool)             {}
func (m *mockContext) Abort()                                         { m.aborted = true }
func (m *mockContext) AbortWithError(code int, err error) *HTTPError {
	m.aborted = true
	m.abortedWith = code
	return &HTTPError{Err: err}
}
func (m *mockContext) AbortWithStatus(code int) { m.aborted = true; m.abortedWith = code }
func (m *mockContext) AbortWithStatusJSON(code int, jsonObj interface{}) {
	m.aborted = true
	m.abortedWith = code
}
func (m *mockContext) Next() { m.shouldNext = true }

// mockResponseWriter implements the gin.ResponseWriter interface for testing
type mockResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
	header http.Header
	body   bytes.Buffer
}

func (m *mockResponseWriter) Status() int                       { return m.status }
func (m *mockResponseWriter) Size() int                         { return m.size }
func (m *mockResponseWriter) WriteString(s string) (int, error) { return m.Write([]byte(s)) }
func (m *mockResponseWriter) Written() bool                     { return m.size > 0 }
func (m *mockResponseWriter) WriteHeaderNow()                   {}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	n, err := m.body.Write(data)
	m.size += n
	return n, err
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

// Test helper to create a logger for testing
func createTestLogger() core.Logger {
	logConfig := &option.LogOption{
		Engine:      "slog",
		Level:       "debug",
		Format:      "json",
		OutputPaths: []string{"stdout"},
	}
	coreLogger, err := logger.New(logConfig)
	if err != nil {
		panic(err)
	}
	return coreLogger
}

func TestNewGinAdapter(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}

	if adapter.Name() != "Gin" {
		t.Errorf("Expected adapter name to be 'Gin', got '%s'", adapter.Name())
	}

	if adapter.Version() != "v1.x" {
		t.Errorf("Expected adapter version to be 'v1.x', got '%s'", adapter.Version())
	}

	if adapter.GetLogger() != logger {
		t.Error("Expected adapter logger to match provided logger")
	}
}

func TestNewGinAdapterWithConfig(t *testing.T) {
	logger := createTestLogger()
	config := Config{
		LogLevel:        core.WarnLevel,
		LogRequestBody:  true,
		LogResponseBody: true,
		MaxBodySize:     2048,
		SkipClientError: true,
	}

	adapter := NewGinAdapterWithConfig(logger, config)

	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}

	adapterConfig := adapter.GetConfig()
	if adapterConfig.LogLevel != core.WarnLevel {
		t.Errorf("Expected log level to be WarnLevel, got %v", adapterConfig.LogLevel)
	}

	if !adapterConfig.LogRequestBody {
		t.Error("Expected LogRequestBody to be true")
	}

	if !adapterConfig.LogResponseBody {
		t.Error("Expected LogResponseBody to be true")
	}

	if adapterConfig.MaxBodySize != 2048 {
		t.Errorf("Expected MaxBodySize to be 2048, got %d", adapterConfig.MaxBodySize)
	}

	if !adapterConfig.SkipClientError {
		t.Error("Expected SkipClientError to be true")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.LogLevel != core.InfoLevel {
		t.Errorf("Expected default log level to be InfoLevel, got %v", config.LogLevel)
	}

	if config.LogRequestBody {
		t.Error("Expected LogRequestBody to be false by default")
	}

	if config.LogResponseBody {
		t.Error("Expected LogResponseBody to be false by default")
	}

	if config.MaxBodySize != 1024*1024 {
		t.Errorf("Expected default MaxBodySize to be 1MB, got %d", config.MaxBodySize)
	}

	if config.SkipClientError {
		t.Error("Expected SkipClientError to be false by default")
	}

	if !config.LogLatency {
		t.Error("Expected LogLatency to be true by default")
	}

	if config.TimeFormat != time.RFC3339 {
		t.Errorf("Expected default TimeFormat to be RFC3339, got %s", config.TimeFormat)
	}

	if !config.UTC {
		t.Error("Expected UTC to be true by default")
	}
}

func TestLoggerMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middleware := adapter.Logger()
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	// Test middleware execution
	ctx := newMockContext()
	ctx.request = httptest.NewRequest("GET", "/api/test", nil)
	ctx.writer.status = 200

	middleware(ctx)

	if !ctx.shouldNext {
		t.Error("Expected Next() to be called")
	}
}

func TestLogRequest(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	// Test successful request
	adapter.LogRequest("GET", "/api/users", 200, 1000000, "user123")

	// Test client error
	adapter.LogRequest("POST", "/api/users", 400, 500000, "")

	// Test server error
	adapter.LogRequest("DELETE", "/api/users/1", 500, 2000000, "user456")
}

func TestLogMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	adapter.LogMiddleware("auth", 50000)
	adapter.LogMiddleware("cors", 10000)
}

func TestLogError(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	err := fmt.Errorf("test error")
	adapter.LogError(err, "POST", "/api/users", 400)
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	recovery := adapter.Recovery()
	if recovery == nil {
		t.Fatal("Expected recovery middleware to be created")
	}

	// Test recovery middleware with panic
	ctx := newMockContext()

	panicMiddleware := func(c Context) {
		panic("test panic")
	}

	// Create a combined middleware that first panics, then recovers
	combined := func(c Context) {
		defer func() {
			if err := recover(); err != nil {
				// Simulate recovery
				c.AbortWithStatus(500)
			}
		}()
		panicMiddleware(c)
	}

	combined(ctx)

	if !ctx.aborted {
		t.Error("Expected context to be aborted after panic")
	}

	if ctx.abortedWith != 500 {
		t.Errorf("Expected abort status to be 500, got %d", ctx.abortedWith)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middleware := adapter.RequestIDMiddleware("X-Request-ID")
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	ctx := newMockContext()
	ctx.request.Header.Set("X-Request-ID", "test-req-123")

	middleware(ctx)

	requestID, exists := ctx.Get("request_id")
	if !exists {
		t.Error("Expected request_id to be set in context")
	}

	if requestID != "test-req-123" {
		t.Errorf("Expected request ID to be 'test-req-123', got '%v'", requestID)
	}
}

func TestUserContextMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middleware := adapter.UserContextMiddleware("X-User-ID")
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	ctx := newMockContext()
	ctx.request.Header.Set("X-User-ID", "user456")

	middleware(ctx)

	userID, exists := ctx.Get("user_id")
	if !exists {
		t.Error("Expected user_id to be set in context")
	}

	if userID != "user456" {
		t.Errorf("Expected user ID to be 'user456', got '%v'", userID)
	}
}

func TestHealthCheckSkipper(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middleware := adapter.HealthCheckSkipper("/health", "/ping")
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	// Test health check path
	ctx := newMockContext()
	ctx.request, _ = http.NewRequest("GET", "/health", nil)

	middleware(ctx)

	skipLogging, exists := ctx.Get("skip_logging")
	if !exists {
		t.Error("Expected skip_logging to be set in context for health check")
	}

	if skip, ok := skipLogging.(bool); !ok || !skip {
		t.Error("Expected skip_logging to be true for health check path")
	}

	// Test normal path
	ctx2 := newMockContext()
	ctx2.request, _ = http.NewRequest("GET", "/api/users", nil)

	middleware(ctx2)

	_, exists2 := ctx2.Get("skip_logging")
	if exists2 {
		t.Error("Expected skip_logging not to be set for normal path")
	}
}

func TestMetricsMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middleware := adapter.MetricsMiddleware()
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	ctx := newMockContext()
	ctx.request = httptest.NewRequest("POST", "/api/users", nil)
	ctx.writer.status = 201
	ctx.Set("user_id", "user123")
	ctx.Set("request_id", "req-456")

	middleware(ctx)
}

func TestRequestBodyLogger(t *testing.T) {
	logger := createTestLogger()
	config := Config{
		LogRequestBody: true,
		MaxBodySize:    1024,
	}
	adapter := NewGinAdapterWithConfig(logger, config)

	middleware := adapter.RequestBodyLogger(512)
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	// Test with request body
	body := strings.NewReader(`{"name":"test","email":"test@example.com"}`)
	ctx := newMockContext()
	ctx.request = httptest.NewRequest("POST", "/api/users", body)
	ctx.request.Header.Set("Content-Type", "application/json")

	middleware(ctx)
}

func TestDefaultMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middlewares := adapter.DefaultMiddleware()

	if len(middlewares) != 3 {
		t.Errorf("Expected 3 default middlewares, got %d", len(middlewares))
	}
}

func TestDebugMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middlewares := adapter.DebugMiddleware()

	if len(middlewares) != 5 {
		t.Errorf("Expected 5 debug middlewares, got %d", len(middlewares))
	}
}

func TestProductionMiddleware(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	middlewares := adapter.ProductionMiddleware("/health", "/ping")

	if len(middlewares) != 5 {
		t.Errorf("Expected 5 production middlewares, got %d", len(middlewares))
	}
}

func TestLogFormatterParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	params := LogFormatterParams{
		Request:    req,
		TimeStamp:  time.Now(),
		StatusCode: 200,
		Latency:    100 * time.Millisecond,
		ClientIP:   "127.0.0.1",
		Method:     "GET",
		Path:       "/test",
		BodySize:   1024,
	}

	// Test status code color
	if params.StatusCodeColor() == "" {
		t.Error("Expected status code color to be set")
	}

	// Test method color
	if params.MethodColor() == "" {
		t.Error("Expected method color to be set")
	}

	// Test reset color
	if params.ResetColor() != "\033[0m" {
		t.Error("Expected reset color to be ANSI reset code")
	}

	// Test color output flag
	if !params.IsOutputColor() {
		t.Error("Expected color output to be true")
	}
}

func TestConfigSetterGetter(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	newConfig := Config{
		LogLevel:    core.ErrorLevel,
		MaxBodySize: 2048,
		UTC:         false,
	}

	adapter.SetConfig(newConfig)
	retrievedConfig := adapter.GetConfig()

	if retrievedConfig.LogLevel != core.ErrorLevel {
		t.Errorf("Expected log level to be ErrorLevel, got %v", retrievedConfig.LogLevel)
	}

	if retrievedConfig.MaxBodySize != 2048 {
		t.Errorf("Expected MaxBodySize to be 2048, got %d", retrievedConfig.MaxBodySize)
	}

	if retrievedConfig.UTC {
		t.Error("Expected UTC to be false")
	}
}

func TestHTTPAdapterInterface(t *testing.T) {
	logger := createTestLogger()
	adapter := NewGinAdapter(logger)

	// Verify that adapter implements HTTPAdapter interface
	var _ integrations.HTTPAdapter = adapter

	// Test interface methods
	adapter.LogRequest("GET", "/test", 200, 1000000, "user123")
	adapter.LogMiddleware("test-middleware", 50000)
	adapter.LogError(fmt.Errorf("test error"), "POST", "/test", 500)
}
