package main

import (
	"fmt"
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// 演示中间件优先级机制的使用
func main() {
	// 创建一个模拟的路由器（实际使用中由框架提供）
	router := newMockRouter()

	// 创建中间件注册器
	registrar := middleware.NewRegistrar()

	// 注册中间件（故意以错误的顺序注册，但会自动按优先级排序）
	registrar.Register("custom", middleware.PriorityCustom, createMiddleware("custom"))
	registrar.Register("auth", middleware.PriorityAuth, createMiddleware("auth"))
	registrar.Register("logger", middleware.PriorityLogger, createMiddleware("logger"))
	registrar.Register("recovery", middleware.PriorityRecovery, createMiddleware("recovery"))
	registrar.Register("cors", middleware.PriorityCORS, createMiddleware("cors"))

	// 使用条件注册
	enableMetrics := true
	registrar.RegisterIf(enableMetrics, "metrics", middleware.PriorityMetrics, createMiddleware("metrics"))

	// 显示注册的中间件列表（按优先级排序）
	fmt.Println("已注册的中间件（按优先级顺序）:")
	for i, name := range registrar.List() {
		fmt.Printf("%d. %s\n", i+1, name)
	}

	// 应用中间件到路由器
	registrar.Apply(router)

	fmt.Printf("\n已应用 %d 个中间件到路由器\n", registrar.Count())

	// 执行中间件链验证顺序
	fmt.Println("\n中间件执行顺序:")
	handler := func(c transport.Context) {
		fmt.Println("  → 业务逻辑")
	}

	// 从后向前构建中间件链
	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}

	// 执行
	handler(nil)
}

// createMiddleware 创建一个带标记的中间件用于演示
func createMiddleware(name string) transport.MiddlewareFunc {
	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			fmt.Printf("  → %s\n", name)
			next(c)
		}
	}
}

// mockRouter 实现 transport.Router 接口用于演示
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
