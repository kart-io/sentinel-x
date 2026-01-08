package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware"
)

// 演示中间件优先级机制的使用
func main() {
	// 创建一个路由器
	gin.SetMode(gin.TestMode)
	router := gin.New()

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
	router.GET("/", func(c *gin.Context) {
		fmt.Println("  → 业务逻辑")
	})

	// 模拟请求
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

// createMiddleware 创建一个带标记的中间件用于演示
func createMiddleware(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("  → %s\n", name)
		c.Next()
	}
}
