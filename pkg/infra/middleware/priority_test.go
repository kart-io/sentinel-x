package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestNewRegistrar 测试创建注册器。
func TestNewRegistrar(t *testing.T) {
	r := NewRegistrar()
	if r == nil {
		t.Fatal("NewRegistrar() returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("NewRegistrar() count = %d, want 0", r.Count())
	}
}

// TestRegister 测试注册中间件。
func TestRegister(t *testing.T) {
	r := NewRegistrar()

	// 创建测试中间件
	middleware1 := func(c *gin.Context) { c.Next() }
	middleware2 := func(c *gin.Context) { c.Next() }

	// 注册中间件
	r.Register("test1", PriorityCustom, middleware1)
	r.Register("test2", PriorityAuth, middleware2)

	if r.Count() != 2 {
		t.Errorf("Register() count = %d, want 2", r.Count())
	}
}

// TestRegisterNilHandler 测试注册 nil 处理器应该 panic。
func TestRegisterNilHandler(t *testing.T) {
	r := NewRegistrar()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Register(nil) should panic")
		}
	}()

	r.Register("test", PriorityCustom, nil)
}

// TestRegisterIf 测试条件注册。
func TestRegisterIf(t *testing.T) {
	r := NewRegistrar()

	middleware := func(c *gin.Context) { c.Next() }

	// 条件为 true，应该注册
	r.RegisterIf(true, "test1", PriorityCustom, middleware)
	if r.Count() != 1 {
		t.Errorf("RegisterIf(true) count = %d, want 1", r.Count())
	}

	// 条件为 false，不应该注册
	r.RegisterIf(false, "test2", PriorityAuth, middleware)
	if r.Count() != 1 {
		t.Errorf("RegisterIf(false) count = %d, want 1", r.Count())
	}
}

// TestApplyPriority 测试按优先级应用中间件。
func TestApplyPriority(t *testing.T) {
	r := NewRegistrar()
	// Use a real gin engine to capture execution order
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 记录中间件执行顺序
	var executionOrder []string

	// 创建带标记的中间件
	createMiddleware := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			executionOrder = append(executionOrder, name)
			c.Next()
		}
	}

	// 故意以错误的顺序注册（低优先级先注册）
	r.Register("custom", PriorityCustom, createMiddleware("custom"))
	r.Register("auth", PriorityAuth, createMiddleware("auth"))
	r.Register("logger", PriorityLogger, createMiddleware("logger"))
	r.Register("recovery", PriorityRecovery, createMiddleware("recovery"))

	// 应用中间件
	r.Apply(router)

	// Add a final handler
	router.GET("/", func(c *gin.Context) {})

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证执行顺序（应该按优先级从高到低）
	expected := []string{"recovery", "logger", "auth", "custom"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("executionOrder length = %d, want %d", len(executionOrder), len(expected))
	}

	for i, name := range expected {
		if executionOrder[i] != name {
			t.Errorf("executionOrder[%d] = %s, want %s", i, executionOrder[i], name)
		}
	}
}

// TestApplySamePriority 测试同优先级按注册顺序执行。
func TestApplySamePriority(t *testing.T) {
	r := NewRegistrar()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var executionOrder []string

	createMiddleware := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			executionOrder = append(executionOrder, name)
			c.Next()
		}
	}

	// 注册三个同优先级的中间件
	r.Register("custom1", PriorityCustom, createMiddleware("custom1"))
	r.Register("custom2", PriorityCustom, createMiddleware("custom2"))
	r.Register("custom3", PriorityCustom, createMiddleware("custom3"))

	r.Apply(router)

	// Add handler
	router.GET("/", func(c *gin.Context) {})

	// 执行
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证按注册顺序执行
	expected := []string{"custom1", "custom2", "custom3"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("executionOrder length = %d, want %d", len(executionOrder), len(expected))
	}

	for i, name := range expected {
		if executionOrder[i] != name {
			t.Errorf("executionOrder[%d] = %s, want %s", i, executionOrder[i], name)
		}
	}
}

// TestList 测试列出中间件。
func TestList(t *testing.T) {
	r := NewRegistrar()

	middleware := func(c *gin.Context) { c.Next() }

	r.Register("recovery", PriorityRecovery, middleware)
	r.Register("auth", PriorityAuth, middleware)
	r.Register("custom", PriorityCustom, middleware)

	names := r.List()
	if len(names) != 3 {
		t.Fatalf("List() length = %d, want 3", len(names))
	}

	// 验证顺序（按优先级降序）
	expected := []string{"recovery[1000]", "auth[400]", "custom[100]"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("List()[%d] = %s, want %s", i, names[i], name)
		}
	}
}

// TestClear 测试清空注册器。
func TestClear(t *testing.T) {
	r := NewRegistrar()

	middleware := func(c *gin.Context) { c.Next() }

	r.Register("test", PriorityCustom, middleware)
	if r.Count() != 1 {
		t.Fatalf("Register() count = %d, want 1", r.Count())
	}

	r.Clear()
	if r.Count() != 0 {
		t.Errorf("Clear() count = %d, want 0", r.Count())
	}
}

// TestComplexPriorityOrder 测试复杂优先级排序场景。
func TestComplexPriorityOrder(t *testing.T) {
	r := NewRegistrar()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var executionOrder []string

	createMiddleware := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			executionOrder = append(executionOrder, name)
			c.Next()
		}
	}

	// 混合注册不同优先级和同优先级的中间件
	r.Register("auth1", PriorityAuth, createMiddleware("auth1"))
	r.Register("recovery", PriorityRecovery, createMiddleware("recovery"))
	r.Register("auth2", PriorityAuth, createMiddleware("auth2"))
	r.Register("logger", PriorityLogger, createMiddleware("logger"))
	r.Register("custom", PriorityCustom, createMiddleware("custom"))
	r.Register("cors", PriorityCORS, createMiddleware("cors"))

	r.Apply(router)

	router.GET("/", func(c *gin.Context) {})

	// 执行
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 预期顺序：
	// 1. recovery (1000)
	// 2. logger (800)
	// 3. cors (600)
	// 4. auth1 (400, 注册序号0)
	// 5. auth2 (400, 注册序号2)
	// 6. custom (100)
	expected := []string{"recovery", "logger", "cors", "auth1", "auth2", "custom"}

	if len(executionOrder) != len(expected) {
		t.Fatalf("executionOrder length = %d, want %d", len(executionOrder), len(expected))
	}

	for i, name := range expected {
		if executionOrder[i] != name {
			t.Errorf("executionOrder[%d] = %s, want %s", i, executionOrder[i], name)
		}
	}
}

// TestEmptyRegistrar 测试空注册器应用。
func TestEmptyRegistrar(t *testing.T) {
	r := NewRegistrar()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	r.Apply(router)
	// We can't easily check internal state of gin, but we can verify it doesn't panic
}

// BenchmarkRegister 性能测试：注册中间件。
func BenchmarkRegister(b *testing.B) {
	middleware := func(c *gin.Context) { c.Next() }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewRegistrar()
		r.Register("test", PriorityCustom, middleware)
	}
}

// BenchmarkApply 性能测试：应用中间件。
func BenchmarkApply(b *testing.B) {
	middleware := func(c *gin.Context) { c.Next() }

	r := NewRegistrar()
	for i := 0; i < 10; i++ {
		r.Register("test", Priority(i*100), middleware)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Apply(router)
	}
}
