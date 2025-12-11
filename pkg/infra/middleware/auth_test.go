package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockContext implements transport.Context for testing
type MockContext struct {
	req *http.Request
}

func (m *MockContext) Header(key string) string {
	return m.req.Header.Get(key)
}

func (m *MockContext) Query(key string) string {
	return m.req.URL.Query().Get(key)
}

func (m *MockContext) HTTPRequest() *http.Request {
	return m.req
}

// Implement other methods with dummy implementations
func (m *MockContext) Request() context.Context                         { return context.Background() }
func (m *MockContext) SetRequest(ctx context.Context)                   {}
func (m *MockContext) ResponseWriter() http.ResponseWriter              { return nil }
func (m *MockContext) Body() io.ReadCloser                              { return nil }
func (m *MockContext) Param(key string) string                          { return "" }
func (m *MockContext) SetHeader(key, value string)                      {}
func (m *MockContext) Bind(v interface{}) error                         { return nil }
func (m *MockContext) Validate(v interface{}) error                     { return nil }
func (m *MockContext) ShouldBindAndValidate(v interface{}) error        { return nil }
func (m *MockContext) MustBindAndValidate(v interface{}) (string, bool) { return "", true }
func (m *MockContext) JSON(code int, v interface{})                     {}
func (m *MockContext) String(code int, s string)                        {}
func (m *MockContext) Error(code int, err error)                        {}
func (m *MockContext) GetRawContext() interface{}                       { return nil }
func (m *MockContext) Lang() string                                     { return "en" }
func (m *MockContext) SetLang(lang string)                              {}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Normal Bearer token",
			header:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
		},
		{
			name:     "Token with spaces",
			header:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9 . payload . signature",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
		},
		{
			name:     "Token with standard base64 (+ and /)",
			header:   "Bearer header.pay/load.sig+nature",
			expected: "header.pay_load.sig-nature",
		},
		{
			name:     "Token with padding (=)",
			header:   "Bearer header.payload.signature==",
			expected: "header.payload.signature",
		},
		{
			name:     "Mixed issues",
			header:   "Bearer header . pay/load . sig+nature ==",
			expected: "header.pay_load.sig-nature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.header)

			ctx := &MockContext{req: req}
			lookup := tokenLookup{source: "header", name: "Authorization"}

			got := extractToken(ctx, lookup, "Bearer")
			if got != tt.expected {
				t.Errorf("extractToken() = %v, want %v", got, tt.expected)
			}
		})
	}
}
