package middleware

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// mockContext implements transport.Context for testing purposes.
type mockContext struct {
	req        *http.Request
	writer     http.ResponseWriter
	headers    map[string]string
	params     map[string]string
	query      map[string]string
	jsonCalled bool
	jsonCode   int
	jsonData   interface{}
	mu         sync.RWMutex
}

// newMockContext creates a new mock context for testing.
func newMockContext(req *http.Request, w http.ResponseWriter) *mockContext {
	return &mockContext{
		req:     req,
		writer:  w,
		headers: make(map[string]string),
		params:  make(map[string]string),
		query:   make(map[string]string),
	}
}

func (m *mockContext) Request() context.Context {
	return m.req.Context()
}

func (m *mockContext) SetRequest(ctx context.Context) {
	m.req = m.req.WithContext(ctx)
}

func (m *mockContext) HTTPRequest() *http.Request {
	return m.req
}

func (m *mockContext) ResponseWriter() http.ResponseWriter {
	return m.writer
}

func (m *mockContext) Body() io.ReadCloser {
	return m.req.Body
}

func (m *mockContext) Param(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.params[key]
}

func (m *mockContext) Query(key string) string {
	// Try query map first
	m.mu.RLock()
	if val, ok := m.query[key]; ok {
		m.mu.RUnlock()
		return val
	}
	m.mu.RUnlock()

	// Fallback to actual request query
	return m.req.URL.Query().Get(key)
}

func (m *mockContext) Header(key string) string {
	return m.req.Header.Get(key)
}

func (m *mockContext) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[key] = value
	if m.writer != nil {
		m.writer.Header().Set(key, value)
	}
}

func (m *mockContext) Bind(v interface{}) error {
	return nil
}

func (m *mockContext) Validate(v interface{}) error {
	return nil
}

func (m *mockContext) ShouldBindAndValidate(v interface{}) error {
	return nil
}

func (m *mockContext) MustBindAndValidate(v interface{}) (string, bool) {
	return "", true
}

func (m *mockContext) JSON(code int, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jsonCalled = true
	m.jsonCode = code
	m.jsonData = v
}

func (m *mockContext) String(code int, s string) {
	// Not implemented for these tests
}

func (m *mockContext) Error(code int, err error) {
	m.JSON(code, map[string]string{"error": err.Error()})
}

func (m *mockContext) GetRawContext() interface{} {
	return nil
}

func (m *mockContext) Lang() string {
	return "en"
}

func (m *mockContext) SetLang(lang string) {
	// Not implemented for these tests
}

// Compile-time check to ensure mockContext implements transport.Context
var _ transport.Context = (*mockContext)(nil)
