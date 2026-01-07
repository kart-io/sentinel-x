package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/requestutil"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

func TestRequestID_GeneratesID(t *testing.T) {
	middleware := RequestID()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {})
	r.ServeHTTP(w, req)

	// Check that request ID was set in response header
	requestID := w.Header().Get(HeaderXRequestID)
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

	existingID := "existing-request-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderXRequestID, existingID)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {})
	r.ServeHTTP(w, req)

	// Check that existing request ID was preserved
	requestID := w.Header().Get(HeaderXRequestID)
	if requestID != existingID {
		t.Errorf("Expected request ID %s, got %s", existingID, requestID)
	}
}

func TestRequestIDWithOptions_CustomHeader(t *testing.T) {
	opts := mwopts.RequestIDOptions{
		Header: "X-Custom-Request-ID",
	}

	middleware := RequestIDWithOptions(opts, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {})
	r.ServeHTTP(w, req)

	// Check that custom header was used
	requestID := w.Header().Get("X-Custom-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Custom-Request-ID header to be set")
	}

	// Default header should not be set
	if w.Header().Get(HeaderXRequestID) != "" {
		t.Error("Expected X-Request-ID header not to be set")
	}
}

func TestRequestIDWithOptions_CustomGenerator(t *testing.T) {
	customID := "custom-generated-id"
	opts := mwopts.RequestIDOptions{}

	customGen := func() string {
		return customID
	}

	middleware := RequestIDWithOptions(opts, customGen)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {})
	r.ServeHTTP(w, req)

	// Check that custom generator was used
	requestID := w.Header().Get(HeaderXRequestID)
	if requestID != customID {
		t.Errorf("Expected request ID %s, got %s", customID, requestID)
	}
}

func TestRequestID_StoresInContext(t *testing.T) {
	middleware := RequestID()
	var capturedCtx context.Context

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		capturedCtx = c.Request.Context()
	})
	r.ServeHTTP(w, req)

	// Check that request ID was stored in context
	requestID := GetRequestID(capturedCtx)
	if requestID == "" {
		t.Error("Expected request ID to be stored in context")
	}

	// Should match the header
	headerID := w.Header().Get(HeaderXRequestID)
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
	ctx := context.WithValue(context.Background(), requestutil.RequestIDKey{}, 12345) // Wrong type
	requestID := GetRequestID(ctx)

	if requestID != "" {
		t.Errorf("Expected empty request ID for wrong type, got %s", requestID)
	}
}

func TestRequestIDWithOptions_Defaults(t *testing.T) {
	// Test with empty config
	opts := mwopts.RequestIDOptions{}

	middleware := RequestIDWithOptions(opts, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {})
	r.ServeHTTP(w, req)

	// Should use default header
	requestID := w.Header().Get(HeaderXRequestID)
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
		id := requestutil.GenerateRequestID()
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
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware)
		r.GET("/test", func(_ *gin.Context) {})
		r.ServeHTTP(w, req)

		requestID := w.Header().Get(HeaderXRequestID)
		if ids[requestID] {
			t.Errorf("Duplicate request ID generated: %s", requestID)
		}
		ids[requestID] = true
	}
}

func TestRequestIDWithOptions_EmptyHeader(t *testing.T) {
	opts := mwopts.RequestIDOptions{
		Header: "", // Empty should use default
	}

	middleware := RequestIDWithOptions(opts, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.GET("/test", func(_ *gin.Context) {})
	r.ServeHTTP(w, req)

	// Should use default header
	requestID := w.Header().Get(HeaderXRequestID)
	if requestID == "" {
		t.Error("Expected default header to be used when config header is empty")
	}
}
