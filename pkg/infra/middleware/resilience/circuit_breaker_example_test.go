package resilience_test

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/infra/middleware/resilience"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// ExampleCircuitBreaker_basic 演示熔断器中间件的基本使用。
//
// 使用场景:
//   - 防止级联失败
//   - 保护下游服务
//   - 快速失败，避免资源浪费
func ExampleCircuitBreaker_basic() {
	// 创建熔断器中间件
	// 参数: maxFailures=5, timeout=60秒, halfOpenMaxCalls=1
	_ = resilience.CircuitBreaker(5, 60, 1)

	// 在 Gin 路由中使用:
	// router := gin.Default()
	// router.Use(middleware)
	// router.GET("/api", func(c *gin.Context) {
	//     c.JSON(http.StatusOK, map[string]string{"message": "请求成功"})
	// })

	fmt.Println("熔断器中间件已启动")
	fmt.Println("配置: 5次失败后熔断，60秒后尝试恢复")

	// 模拟请求
	// ... handler(mockContext) ...

	// Output:
	// 熔断器中间件已启动
	// 配置: 5次失败后熔断，60秒后尝试恢复
}

// ExampleCircuitBreakerWithOptions_advanced 演示带配置选项的熔断器。
//
// 使用场景:
//   - 微服务间调用保护
//   - 自定义错误阈值（如 4xx 也触发熔断）
//   - 跳过健康检查和监控端点
func ExampleCircuitBreakerWithOptions_advanced() {
	// 配置熔断器选项
	opts := mwopts.CircuitBreakerOptions{
		MaxFailures:      5,
		Timeout:          60, // 60 秒
		HalfOpenMaxCalls: 1,
		SkipPaths:        []string{"/health", "/metrics"},
		SkipPathPrefixes: []string{"/static/"},
		ErrorThreshold:   500, // 5xx 错误触发熔断
	}

	// 创建中间件
	_ = resilience.CircuitBreakerWithOptions(opts)

	// 在 Gin 路由中使用:
	// router := gin.Default()
	// router.Use(middleware)
	// router.GET("/api", func(c *gin.Context) {
	//     c.JSON(http.StatusOK, map[string]string{"message": "请求成功"})
	// })

	fmt.Println("熔断器中间件已配置")
	fmt.Println("跳过路径: /health, /metrics")
	fmt.Println("跳过前缀: /static/")
	fmt.Println("错误阈值: >= 500 (5xx 错误)")

	// 模拟请求
	// ... handler(mockContext) ...

	// Output:
	// 熔断器中间件已配置
	// 跳过路径: /health, /metrics
	// 跳过前缀: /static/
	// 错误阈值: >= 500 (5xx 错误)
}

// ExampleCircuitBreaker_stateMachine 演示熔断器的状态机转换。
func ExampleCircuitBreaker_stateMachine() {
	fmt.Println("=== 熔断器状态机 ===")
	fmt.Println()

	fmt.Println("【状态1: Closed (关闭)】")
	fmt.Println("描述: 正常处理所有请求")
	fmt.Println("条件: 失败次数 < MaxFailures")
	fmt.Println("行为: 记录失败次数，成功时重置计数")
	fmt.Println()

	fmt.Println("【状态2: Open (打开)】")
	fmt.Println("描述: 拒绝所有请求")
	fmt.Println("条件: 失败次数 >= MaxFailures")
	fmt.Println("行为: 立即返回 503 Service Unavailable")
	fmt.Println("持续时间: Timeout 配置的时间")
	fmt.Println()

	fmt.Println("【状态3: Half-Open (半开)】")
	fmt.Println("描述: 允许少量请求探测")
	fmt.Println("条件: Open 状态超时后")
	fmt.Println("行为:")
	fmt.Println("  - 允许 HalfOpenMaxCalls 个请求通过")
	fmt.Println("  - 全部成功 → 转为 Closed")
	fmt.Println("  - 任一失败 → 转为 Open")
	fmt.Println()

	fmt.Println("【状态转换】")
	fmt.Println("Closed --[失败次数>=MaxFailures]--> Open")
	fmt.Println("Open --[超时Timeout]--> Half-Open")
	fmt.Println("Half-Open --[全部成功]--> Closed")
	fmt.Println("Half-Open --[任一失败]--> Open")

	// Output:
	// === 熔断器状态机 ===
	//
	// 【状态1: Closed (关闭)】
	// 描述: 正常处理所有请求
	// 条件: 失败次数 < MaxFailures
	// 行为: 记录失败次数，成功时重置计数
	//
	// 【状态2: Open (打开)】
	// 描述: 拒绝所有请求
	// 条件: 失败次数 >= MaxFailures
	// 行为: 立即返回 503 Service Unavailable
	// 持续时间: Timeout 配置的时间
	//
	// 【状态3: Half-Open (半开)】
	// 描述: 允许少量请求探测
	// 条件: Open 状态超时后
	// 行为:
	//   - 允许 HalfOpenMaxCalls 个请求通过
	//   - 全部成功 → 转为 Closed
	//   - 任一失败 → 转为 Open
	//
	// 【状态转换】
	// Closed --[失败次数>=MaxFailures]--> Open
	// Open --[超时Timeout]--> Half-Open
	// Half-Open --[全部成功]--> Closed
	// Half-Open --[任一失败]--> Open
}

// ExampleCircuitBreaker_microservices 演示在微服务中使用熔断器。
func ExampleCircuitBreaker_microservices() {
	fmt.Println("=== 微服务熔断器配置 ===")
	fmt.Println()

	fmt.Println("【场景1: API 网关】")
	fmt.Println("配置:")
	fmt.Println("  MaxFailures: 10        # 网关流量大，阈值更高")
	fmt.Println("  Timeout: 30s           # 快速恢复尝试")
	fmt.Println("  ErrorThreshold: 500    # 仅 5xx 错误触发")
	fmt.Println()

	fmt.Println("【场景2: 后端服务】")
	fmt.Println("配置:")
	fmt.Println("  MaxFailures: 5         # 敏感度更高")
	fmt.Println("  Timeout: 60s           # 给下游足够恢复时间")
	fmt.Println("  ErrorThreshold: 500")
	fmt.Println()

	fmt.Println("【场景3: 第三方 API 调用】")
	fmt.Println("配置:")
	fmt.Println("  MaxFailures: 3         # 快速熔断")
	fmt.Println("  Timeout: 120s          # 外部服务恢复可能较慢")
	fmt.Println("  ErrorThreshold: 400    # 4xx 也可能是下游问题")
	fmt.Println()

	fmt.Println("【最佳实践】")
	fmt.Println("1. 根据服务重要性调整参数")
	fmt.Println("2. 监控熔断器状态和触发次数")
	fmt.Println("3. 配合降级策略使用")
	fmt.Println("4. 定期评估和调整阈值")

	// Output:
	// === 微服务熔断器配置 ===
	//
	// 【场景1: API 网关】
	// 配置:
	//   MaxFailures: 10        # 网关流量大，阈值更高
	//   Timeout: 30s           # 快速恢复尝试
	//   ErrorThreshold: 500    # 仅 5xx 错误触发
	//
	// 【场景2: 后端服务】
	// 配置:
	//   MaxFailures: 5         # 敏感度更高
	//   Timeout: 60s           # 给下游足够恢复时间
	//   ErrorThreshold: 500
	//
	// 【场景3: 第三方 API 调用】
	// 配置:
	//   MaxFailures: 3         # 快速熔断
	//   Timeout: 120s          # 外部服务恢复可能较慢
	//   ErrorThreshold: 400    # 4xx 也可能是下游问题
	//
	// 【最佳实践】
	// 1. 根据服务重要性调整参数
	// 2. 监控熔断器状态和触发次数
	// 3. 配合降级策略使用
	// 4. 定期评估和调整阈值
}

// ExampleCircuitBreaker_withFallback 演示熔断器与降级策略结合使用。
func ExampleCircuitBreaker_withFallback() {
	ctx := context.Background()

	fmt.Println("=== 熔断器 + 降级策略 ===")
	fmt.Println()

	// 模拟业务逻辑
	callDownstreamService := func() (string, error) {
		// 实际会调用下游服务
		// 这里模拟熔断器打开的情况
		return "", fmt.Errorf("circuit breaker is open")
	}

	// 执行请求
	result, err := callDownstreamService()
	if err != nil {
		// 降级策略：根据实际场景选择合适的策略
		// 这里演示策略3（返回降级功能）
		fmt.Println("策略: 返回降级功能")
		result = "degraded service available"
	}

	_ = ctx
	_ = result

	fmt.Println()
	fmt.Println("降级策略建议:")
	fmt.Println("1. 缓存: 使用最近成功的响应")
	fmt.Println("2. 默认值: 返回安全的默认数据")
	fmt.Println("3. 部分功能: 提供有限但可用的服务")
	fmt.Println("4. 友好提示: 告知用户服务暂时不可用")

	// Output:
	// === 熔断器 + 降级策略 ===
	//
	// 策略1: 使用缓存数据
	// 策略2: 使用默认响应
	// 策略3: 返回降级功能
	//
	// 降级策略建议:
	// 1. 缓存: 使用最近成功的响应
	// 2. 默认值: 返回安全的默认数据
	// 3. 部分功能: 提供有限但可用的服务
	// 4. 友好提示: 告知用户服务暂时不可用
}

// ExampleCircuitBreaker_monitoring 演示熔断器监控和告警。
func ExampleCircuitBreaker_monitoring() {
	fmt.Println("=== 熔断器监控指标 ===")
	fmt.Println()

	fmt.Println("【关键指标】")
	fmt.Println("1. circuit_breaker_state")
	fmt.Println("   - closed: 0")
	fmt.Println("   - open: 1")
	fmt.Println("   - half-open: 2")
	fmt.Println()

	fmt.Println("2. circuit_breaker_failures")
	fmt.Println("   - 当前失败次数")
	fmt.Println()

	fmt.Println("3. circuit_breaker_requests_total")
	fmt.Println("   - 总请求数（按状态分类）")
	fmt.Println()

	fmt.Println("4. circuit_breaker_rejected_total")
	fmt.Println("   - 被拒绝的请求数")
	fmt.Println()

	fmt.Println("【告警规则示例】")
	fmt.Println("# 熔断器打开超过5分钟")
	fmt.Println("circuit_breaker_state == 1 for 5m")
	fmt.Println()
	fmt.Println("# 10分钟内拒绝请求超过100次")
	fmt.Println("rate(circuit_breaker_rejected_total[10m]) > 100")
	fmt.Println()
	fmt.Println("# 失败率超过20%")
	fmt.Println("circuit_breaker_failures / circuit_breaker_requests_total > 0.2")

	// Output:
	// === 熔断器监控指标 ===
	//
	// 【关键指标】
	// 1. circuit_breaker_state
	//    - closed: 0
	//    - open: 1
	//    - half-open: 2
	//
	// 2. circuit_breaker_failures
	//    - 当前失败次数
	//
	// 3. circuit_breaker_requests_total
	//    - 总请求数（按状态分类）
	//
	// 4. circuit_breaker_rejected_total
	//    - 被拒绝的请求数
	//
	// 【告警规则示例】
	// # 熔断器打开超过5分钟
	// circuit_breaker_state == 1 for 5m
	//
	// # 10分钟内拒绝请求超过100次
	// rate(circuit_breaker_rejected_total[10m]) > 100
	//
	// # 失败率超过20%
	// circuit_breaker_failures / circuit_breaker_requests_total > 0.2
}

// ExampleCircuitBreaker_comparison 对比熔断器与其他弹性模式。
func ExampleCircuitBreaker_comparison() {
	fmt.Println("=== 弹性模式对比 ===")
	fmt.Println()

	fmt.Println("【熔断器 (Circuit Breaker)】")
	fmt.Println("目的: 防止级联失败")
	fmt.Println("触发: 失败次数达到阈值")
	fmt.Println("行为: 快速失败，拒绝请求")
	fmt.Println("恢复: 超时后半开状态探测")
	fmt.Println("适用: 下游服务故障")
	fmt.Println()

	fmt.Println("【重试 (Retry)】")
	fmt.Println("目的: 处理临时性故障")
	fmt.Println("触发: 请求失败")
	fmt.Println("行为: 等待后重新尝试")
	fmt.Println("恢复: 成功后立即恢复")
	fmt.Println("适用: 网络抖动、临时超时")
	fmt.Println()

	fmt.Println("【超时 (Timeout)】")
	fmt.Println("目的: 防止资源长时间占用")
	fmt.Println("触发: 请求时间超过阈值")
	fmt.Println("行为: 取消请求，释放资源")
	fmt.Println("恢复: 每次请求独立超时")
	fmt.Println("适用: 慢查询、阻塞调用")
	fmt.Println()

	fmt.Println("【限流 (Rate Limit)】")
	fmt.Println("目的: 保护服务不过载")
	fmt.Println("触发: 请求速率超过配额")
	fmt.Println("行为: 拒绝超额请求")
	fmt.Println("恢复: 时间窗口滑动")
	fmt.Println("适用: 流量突刺、恶意攻击")
	fmt.Println()

	fmt.Println("【组合使用建议】")
	fmt.Println("Timeout → Retry → Circuit Breaker → Rate Limit")
	fmt.Println("1. 先设置超时，避免永久阻塞")
	fmt.Println("2. 临时失败时重试")
	fmt.Println("3. 持续失败时熔断")
	fmt.Println("4. 最外层限流保护")

	// Output:
	// === 弹性模式对比 ===
	//
	// 【熔断器 (Circuit Breaker)】
	// 目的: 防止级联失败
	// 触发: 失败次数达到阈值
	// 行为: 快速失败，拒绝请求
	// 恢复: 超时后半开状态探测
	// 适用: 下游服务故障
	//
	// 【重试 (Retry)】
	// 目的: 处理临时性故障
	// 触发: 请求失败
	// 行为: 等待后重新尝试
	// 恢复: 成功后立即恢复
	// 适用: 网络抖动、临时超时
	//
	// 【超时 (Timeout)】
	// 目的: 防止资源长时间占用
	// 触发: 请求时间超过阈值
	// 行为: 取消请求，释放资源
	// 恢复: 每次请求独立超时
	// 适用: 慢查询、阻塞调用
	//
	// 【限流 (Rate Limit)】
	// 目的: 保护服务不过载
	// 触发: 请求速率超过配额
	// 行为: 拒绝超额请求
	// 恢复: 时间窗口滑动
	// 适用: 流量突刺、恶意攻击
	//
	// 【组合使用建议】
	// Timeout → Retry → Circuit Breaker → Rate Limit
	// 1. 先设置超时，避免永久阻塞
	// 2. 临时失败时重试
	// 3. 持续失败时熔断
	// 4. 最外层限流保护
}
