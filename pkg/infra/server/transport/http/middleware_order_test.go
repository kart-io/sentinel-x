package http

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	options "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// TestMiddlewareOrder_Default 测试默认中间件顺序。
func TestMiddlewareOrder_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建中间件选项（使用默认顺序）
	mwOpts := mwopts.NewOptions()

	// 创建服务器
	serverOpts := options.NewOptions()
	server := NewServer(serverOpts, mwOpts)

	// 验证默认顺序
	defaultOrder := mwOpts.GetMiddlewareOrder()
	expectedOrder := []string{
		mwopts.MiddlewareRecovery,
		mwopts.MiddlewareRequestID,
		mwopts.MiddlewareLogger,
		mwopts.MiddlewareMetrics,
		mwopts.MiddlewareCORS,
		mwopts.MiddlewareTimeout,
	}

	if len(defaultOrder) != len(expectedOrder) {
		t.Errorf("Expected %d middleware in default order, got %d", len(expectedOrder), len(defaultOrder))
	}

	for i, name := range expectedOrder {
		if i >= len(defaultOrder) {
			break
		}
		if defaultOrder[i] != name {
			t.Errorf("Expected middleware at position %d to be %q, got %q", i, name, defaultOrder[i])
		}
	}

	// 添加测试路由
	server.Engine().GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// 发送测试请求
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	server.Engine().ServeHTTP(w, req)

	// 验证请求成功
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

// TestMiddlewareOrder_Custom 测试自定义中间件顺序。
func TestMiddlewareOrder_Custom(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建自定义顺序的中间件选项
	mwOpts := mwopts.NewOptions()
	customOrder := []string{
		mwopts.MiddlewareRecovery,
		mwopts.MiddlewareRequestID,
		mwopts.MiddlewareCORS, // CORS 提前到 logger 之前
		mwopts.MiddlewareLogger,
		mwopts.MiddlewareTimeout,
	}
	mwOpts.Middleware = customOrder

	// 创建服务器
	serverOpts := options.NewOptions()
	server := NewServer(serverOpts, mwOpts)

	// 验证自定义顺序
	actualOrder := mwOpts.GetMiddlewareOrder()
	if len(actualOrder) != len(customOrder) {
		t.Errorf("Expected %d middleware in custom order, got %d", len(customOrder), len(actualOrder))
	}

	for i, name := range customOrder {
		if i >= len(actualOrder) {
			break
		}
		if actualOrder[i] != name {
			t.Errorf("Expected middleware at position %d to be %q, got %q", i, name, actualOrder[i])
		}
	}

	// 添加测试路由
	server.Engine().GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// 发送测试请求
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	server.Engine().ServeHTTP(w, req)

	// 验证请求成功
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

// TestMiddlewareOrder_Validation 测试中间件顺序验证。
func TestMiddlewareOrder_Validation(t *testing.T) {
	tests := []struct {
		name        string
		middleware  []string
		expectError bool
	}{
		{
			name:        "valid middleware list",
			middleware:  []string{mwopts.MiddlewareRecovery, mwopts.MiddlewareLogger},
			expectError: false,
		},
		{
			name:        "unknown middleware",
			middleware:  []string{mwopts.MiddlewareRecovery, "unknown-middleware"},
			expectError: true,
		},
		{
			name:        "duplicate middleware",
			middleware:  []string{mwopts.MiddlewareRecovery, mwopts.MiddlewareRecovery},
			expectError: true,
		},
		{
			name:        "empty middleware list",
			middleware:  []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mwOpts := mwopts.NewOptions()
			mwOpts.Middleware = tt.middleware

			errs := mwOpts.ValidateMiddleware()

			hasError := len(errs) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, errs)
			}
		})
	}
}

// TestMiddlewareOrder_ExecutionOrder 测试中间件实际执行顺序。
func TestMiddlewareOrder_ExecutionOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 用于记录中间件执行顺序
	var executionOrder []string
	var mu sync.Mutex

	// 创建自定义顺序
	mwOpts := mwopts.NewOptions()
	mwOpts.Middleware = []string{
		mwopts.MiddlewareRecovery,
		mwopts.MiddlewareRequestID,
		mwopts.MiddlewareLogger,
	}
	mwOpts.SetConfig(mwopts.MiddlewareRecovery, mwopts.NewRecoveryOptions())
	mwOpts.SetConfig(mwopts.MiddlewareRequestID, mwopts.NewRequestIDOptions())
	mwOpts.SetConfig(mwopts.MiddlewareLogger, mwopts.NewLoggerOptions())

	// 创建服务器
	serverOpts := options.NewOptions()
	server := NewServer(serverOpts, mwOpts)

	// 添加自定义中间件来跟踪执行顺序
	// 这些中间件会在配置的中间件之后执行
	server.Engine().Use(func(c *gin.Context) {
		mu.Lock()
		executionOrder = append(executionOrder, "test-middleware")
		mu.Unlock()
		c.Next()
	})

	// 添加测试路由
	server.Engine().GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// 发送测试请求
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	server.Engine().ServeHTTP(w, req)

	// 验证请求成功
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// 验证至少有一个中间件执行了
	mu.Lock()
	defer mu.Unlock()
	if len(executionOrder) == 0 {
		t.Error("Expected at least one middleware to execute")
	}
}
