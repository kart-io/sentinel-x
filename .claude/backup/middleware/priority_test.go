package middleware

import (
	"net/http"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// mockRouter 实现 transport.Router 接口用于测试。
type mockRouter struct {
	middlewares []transport.MiddlewareFunc
}

func newMockRouter() *mockRouter {
	return &mockRouter{
		middlewares: make([]transport.MiddlewareFunc, 0),
	}
}

func (m *mockRouter) Handle(method, path string, handler transport.HandlerFunc) {}
func (m *mockRouter) Group(prefix string) transport.Router                      { return m }
func (m *mockRouter) Static(prefix, root string)                                {}
func (m *mockRouter) Mount(prefix string, handler http.Handler)                 {}

func (m *mockRouter) Use(middleware ...transport.MiddlewareFunc) {
	m.middlewares = append(m.middlewares, middleware...)
}

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
	middleware1 := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

	middleware2 := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

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

	middleware := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

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
	router := newMockRouter()

	// 记录中间件执行顺序
	var executionOrder []string

	// 创建带标记的中间件
	createMiddleware := func(name string) transport.MiddlewareFunc {
		return func(next transport.HandlerFunc) transport.HandlerFunc {
			return func(c transport.Context) {
				executionOrder = append(executionOrder, name)
				next(c)
			}
		}
	}

	// 故意以错误的顺序注册（低优先级先注册）
	r.Register("custom", PriorityCustom, createMiddleware("custom"))
	r.Register("auth", PriorityAuth, createMiddleware("auth"))
	r.Register("logger", PriorityLogger, createMiddleware("logger"))
	r.Register("recovery", PriorityRecovery, createMiddleware("recovery"))

	// 应用中间件
	r.Apply(router)

	// 验证注册了4个中间件
	if len(router.middlewares) != 4 {
		t.Errorf("Apply() registered %d middlewares, want 4", len(router.middlewares))
	}

	// 执行中间件链验证顺序
	req := &http.Request{}
	ctx := newMockContext(req, nil)
	handler := func(c transport.Context) {}

	// 从后向前构建中间件链（符合 Chain 的逻辑）
	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}

	// 执行
	handler(ctx)

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
	router := newMockRouter()

	var executionOrder []string

	createMiddleware := func(name string) transport.MiddlewareFunc {
		return func(next transport.HandlerFunc) transport.HandlerFunc {
			return func(c transport.Context) {
				executionOrder = append(executionOrder, name)
				next(c)
			}
		}
	}

	// 注册三个同优先级的中间件
	r.Register("custom1", PriorityCustom, createMiddleware("custom1"))
	r.Register("custom2", PriorityCustom, createMiddleware("custom2"))
	r.Register("custom3", PriorityCustom, createMiddleware("custom3"))

	r.Apply(router)

	// 执行中间件链
	req := &http.Request{}
	ctx := newMockContext(req, nil)
	handler := func(c transport.Context) {}

	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}
	handler(ctx)

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

	middleware := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

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

	middleware := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

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
	router := newMockRouter()

	var executionOrder []string

	createMiddleware := func(name string) transport.MiddlewareFunc {
		return func(next transport.HandlerFunc) transport.HandlerFunc {
			return func(c transport.Context) {
				executionOrder = append(executionOrder, name)
				next(c)
			}
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

	// 执行中间件链
	req := &http.Request{}
	ctx := newMockContext(req, nil)
	handler := func(c transport.Context) {}

	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}
	handler(ctx)

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
	router := newMockRouter()

	r.Apply(router)

	if len(router.middlewares) != 0 {
		t.Errorf("Apply() on empty registrar registered %d middlewares, want 0", len(router.middlewares))
	}
}

// BenchmarkRegister 性能测试：注册中间件。
func BenchmarkRegister(b *testing.B) {
	middleware := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewRegistrar()
		r.Register("test", PriorityCustom, middleware)
	}
}

// BenchmarkApply 性能测试：应用中间件。
func BenchmarkApply(b *testing.B) {
	middleware := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) { next(c) }
	}

	r := NewRegistrar()
	for i := 0; i < 10; i++ {
		r.Register("test", Priority(i*100), middleware)
	}

	router := newMockRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Apply(router)
	}
}
