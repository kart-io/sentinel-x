package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

func TestRequestID_GeneratesID(t *testing.T) {
	middleware := RequestID()
	handler := middleware(func(ctx transport.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Check that request ID was set in response header
	requestID := mockCtx.headers[HeaderXRequestID]
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}

	// Check that request ID is a valid hex string
	if len(requestID) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected request ID length 32, got %d", len(requestID))
	}
}

func TestRequestID_PreservesExistingID(t *testing.T) {
	middleware := RequestID()
	handler := middleware(func(ctx transport.Context) {})

	existingID := "existing-request-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderXRequestID, existingID)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Check that existing request ID was preserved
	requestID := mockCtx.headers[HeaderXRequestID]
	if requestID != existingID {
		t.Errorf("Expected request ID %s, got %s", existingID, requestID)
	}
}

func TestRequestIDWithConfig_CustomHeader(t *testing.T) {
	config := RequestIDConfig{
		Header: "X-Custom-Request-ID",
	}

	middleware := RequestIDWithConfig(config)
	handler := middleware(func(ctx transport.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Check that custom header was used
	requestID := mockCtx.headers["X-Custom-Request-ID"]
	if requestID == "" {
		t.Error("Expected X-Custom-Request-ID header to be set")
	}

	// Default header should not be set
	if mockCtx.headers[HeaderXRequestID] != "" {
		t.Error("Expected X-Request-ID header not to be set")
	}
}

func TestRequestIDWithConfig_CustomGenerator(t *testing.T) {
	customID := "custom-generated-id"
	config := RequestIDConfig{
		Generator: func() string {
			return customID
		},
	}

	middleware := RequestIDWithConfig(config)
	handler := middleware(func(ctx transport.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Check that custom generator was used
	requestID := mockCtx.headers[HeaderXRequestID]
	if requestID != customID {
		t.Errorf("Expected request ID %s, got %s", customID, requestID)
	}
}

func TestRequestID_StoresInContext(t *testing.T) {
	middleware := RequestID()
	var capturedCtx context.Context

	handler := middleware(func(ctx transport.Context) {
		capturedCtx = ctx.Request()
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Check that request ID was stored in context
	requestID := GetRequestID(capturedCtx)
	if requestID == "" {
		t.Error("Expected request ID to be stored in context")
	}

	// Should match the header
	headerID := mockCtx.headers[HeaderXRequestID]
	if requestID != headerID {
		t.Errorf("Context request ID %s does not match header %s", requestID, headerID)
	}
}

func TestGetRequestID_NotFound(t *testing.T) {
	ctx := context.Background()
	requestID := GetRequestID(ctx)

	if requestID != "" {
		t.Errorf("Expected empty request ID, got %s", requestID)
	}
}

func TestGetRequestID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey{}, 12345) // Wrong type
	requestID := GetRequestID(ctx)

	if requestID != "" {
		t.Errorf("Expected empty request ID for wrong type, got %s", requestID)
	}
}

func TestRequestIDWithConfig_Defaults(t *testing.T) {
	// Test with empty config
	config := RequestIDConfig{}

	middleware := RequestIDWithConfig(config)
	handler := middleware(func(ctx transport.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Should use default header
	requestID := mockCtx.headers[HeaderXRequestID]
	if requestID == "" {
		t.Error("Expected default header X-Request-ID to be set")
	}

	// Should generate a valid ID with default generator
	if len(requestID) != 32 {
		t.Errorf("Expected default generator to produce 32-char ID, got %d", len(requestID))
	}
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	// Generate multiple IDs and check they are unique
	ids := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		id := generateRequestID()
		if ids[id] {
			t.Errorf("Generated duplicate request ID: %s", id)
		}
		ids[id] = true

		// Verify format
		if len(id) != 32 {
			t.Errorf("Generated ID has wrong length: %d", len(id))
		}
	}

	if len(ids) != iterations {
		t.Errorf("Expected %d unique IDs, got %d", iterations, len(ids))
	}
}

func TestRequestID_MultipleRequests(t *testing.T) {
	middleware := RequestID()

	// Process multiple requests and ensure each gets a unique ID
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		handler := middleware(func(ctx transport.Context) {})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		mockCtx := newMockContext(req, w)

		handler(mockCtx)

		requestID := mockCtx.headers[HeaderXRequestID]
		if ids[requestID] {
			t.Errorf("Duplicate request ID generated: %s", requestID)
		}
		ids[requestID] = true
	}
}

func TestRequestIDWithConfig_EmptyHeader(t *testing.T) {
	config := RequestIDConfig{
		Header: "", // Empty should use default
	}

	middleware := RequestIDWithConfig(config)
	handler := middleware(func(ctx transport.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mockCtx := newMockContext(req, w)

	handler(mockCtx)

	// Should use default header
	requestID := mockCtx.headers[HeaderXRequestID]
	if requestID == "" {
		t.Error("Expected default header to be used when config header is empty")
	}
}
