package resilience

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

func TestBodyLimit_NormalRequest(t *testing.T) {
	maxSize := int64(1024) // 1KB
	middleware := BodyLimit(maxSize)

	handlerCalled := false

	// 创建小于限制的请求体
	body := bytes.NewReader([]byte("small body"))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(len("small body"))
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.POST("/test", func(c *gin.Context) {
		handlerCalled = true
	})

	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called for normal request")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestBodyLimit_LargeRequest_ContentLength(t *testing.T) {
	maxSize := int64(10) // 10 字节
	middleware := BodyLimit(maxSize)

	handlerCalled := false

	// 创建超过限制的请求体
	body := bytes.NewReader([]byte("this is a very large body that exceeds the limit"))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(len("this is a very large body that exceeds the limit"))
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.POST("/test", func(c *gin.Context) {
		handlerCalled = true
	})

	r.ServeHTTP(w, req)

	// 应该被拒绝，不调用处理程序
	if handlerCalled {
		t.Error("Expected handler NOT to be called for large request")
	}

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}

func TestBodyLimitWithOptions_SkipPaths(t *testing.T) {
	opts := mwopts.BodyLimitOptions{
		MaxSize:   10, // 10 字节
		SkipPaths: []string{"/upload", "/webhook"},
	}

	middleware := BodyLimitWithOptions(opts)

	tests := []struct {
		name         string
		path         string
		bodySize     int
		shouldReject bool
	}{
		{
			name:         "skipped path /upload - not rejected",
			path:         "/upload",
			bodySize:     100,
			shouldReject: false,
		},
		{
			name:         "skipped path /webhook - not rejected",
			path:         "/webhook",
			bodySize:     100,
			shouldReject: false,
		},
		{
			name:         "normal path - rejected",
			path:         "/api/test",
			bodySize:     100,
			shouldReject: true,
		},
		{
			name:         "normal path - accepted (small body)",
			path:         "/api/test",
			bodySize:     5,
			shouldReject: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false

			body := bytes.NewReader(bytes.Repeat([]byte("a"), tt.bodySize))
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			req.ContentLength = int64(tt.bodySize)
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.POST(tt.path, func(c *gin.Context) {
				handlerCalled = true
			})

			r.ServeHTTP(w, req)

			if tt.shouldReject {
				if handlerCalled {
					t.Error("Expected handler NOT to be called for large body")
				}
				if w.Code != http.StatusRequestEntityTooLarge {
					t.Errorf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
				}
			} else {
				if !handlerCalled {
					t.Error("Expected handler to be called")
				}
			}
		})
	}
}

func TestBodyLimitWithOptions_SkipPathPrefixes(t *testing.T) {
	opts := mwopts.BodyLimitOptions{
		MaxSize:          10, // 10 字节
		SkipPathPrefixes: []string{"/api/v1/files", "/internal"},
	}

	middleware := BodyLimitWithOptions(opts)

	tests := []struct {
		name         string
		path         string
		bodySize     int
		shouldReject bool
	}{
		{
			name:         "skipped prefix /api/v1/files - not rejected",
			path:         "/api/v1/files/upload",
			bodySize:     100,
			shouldReject: false,
		},
		{
			name:         "skipped prefix /internal - not rejected",
			path:         "/internal/debug",
			bodySize:     100,
			shouldReject: false,
		},
		{
			name:         "non-matching prefix - rejected",
			path:         "/api/v2/data",
			bodySize:     100,
			shouldReject: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false

			body := bytes.NewReader(bytes.Repeat([]byte("a"), tt.bodySize))
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			req.ContentLength = int64(tt.bodySize)
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware)
			r.POST(tt.path, func(c *gin.Context) {
				handlerCalled = true
			})

			r.ServeHTTP(w, req)

			if tt.shouldReject {
				if handlerCalled {
					t.Error("Expected handler NOT to be called")
				}
			} else {
				if !handlerCalled {
					t.Error("Expected handler to be called")
				}
			}
		})
	}
}

func TestBodyLimitWithOptions_DefaultMaxSize(t *testing.T) {
	// 零值应该使用默认的 4MB
	opts := mwopts.BodyLimitOptions{
		MaxSize: 0,
	}

	middleware := BodyLimitWithOptions(opts)

	handlerCalled := false

	// 1MB 请求应该通过（小于默认的 4MB）
	body := bytes.NewReader(bytes.Repeat([]byte("a"), 1024*1024))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = 1024 * 1024
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.POST("/test", func(c *gin.Context) {
		handlerCalled = true
	})

	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called with default config")
	}
}

func TestBodyLimit_NoContentLength(t *testing.T) {
	maxSize := int64(10)
	middleware := BodyLimit(maxSize)

	handlerCalled := false

	// 没有 Content-Length 头的请求
	body := strings.NewReader("test body")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	// 不设置 ContentLength（默认为 -1）
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware)
	r.POST("/test", func(c *gin.Context) {
		handlerCalled = true
	})

	r.ServeHTTP(w, req)

	// 应该允许通过（Content-Length 检查会跳过）
	// 实际的读取限制由 MaxBytesReader 处理
	if !handlerCalled {
		t.Error("Expected handler to be called when Content-Length is missing")
	}
}
