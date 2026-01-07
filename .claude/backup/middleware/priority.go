package middleware

import (
	"fmt"
	"sort"
	"sync"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// Priority 定义中间件优先级类型。
// 数值越大，优先级越高，越先执行。
type Priority int

// 预定义的中间件优先级常量。
// 优先级从高到低排列，确保中间件按正确顺序执行。
const (
	// PriorityRecovery 最高优先级，必须第一个执行以捕获所有 panic。
	PriorityRecovery Priority = 1000

	// PriorityRequestID 第二优先级，为其他中间件提供唯一请求 ID。
	PriorityRequestID Priority = 900

	// PriorityLogger 依赖 RequestID，记录请求日志。
	PriorityLogger Priority = 800

	// PriorityMetrics 观测性中间件，收集性能指标。
	PriorityMetrics Priority = 700

	// PriorityTracing 分布式追踪中间件。
	PriorityTracing Priority = 650

	// PriorityCORS 跨域资源共享，安全相关。
	PriorityCORS Priority = 600

	// PriorityBodyLimit 请求体大小限制，防止 DoS 攻击。
	PriorityBodyLimit Priority = 550

	// PrioritySecurityHeaders 安全响应头设置。
	PrioritySecurityHeaders Priority = 540

	// PriorityTimeout 请求超时控制，弹性机制。
	PriorityTimeout Priority = 500

	// PriorityAuth 身份认证，必须在业务逻辑前执行。
	PriorityAuth Priority = 400

	// PriorityAuthz 授权检查，在认证后执行。
	PriorityAuthz Priority = 300

	// PriorityCompression 响应压缩，在业务逻辑之后执行。
	PriorityCompression Priority = 200

	// PriorityCustom 自定义中间件的默认优先级。
	PriorityCustom Priority = 100
)

// PrioritizedMiddleware 表示带优先级的中间件。
type PrioritizedMiddleware struct {
	// Name 中间件名称，用于调试和日志。
	Name string

	// Priority 优先级，数值越大越先执行。
	Priority Priority

	// Handler 中间件处理函数。
	Handler transport.MiddlewareFunc

	// order 注册顺序，用于同优先级时的排序。
	order int
}

// Registrar 中间件注册器，管理中间件的注册和应用。
type Registrar struct {
	mu          sync.RWMutex
	middlewares []PrioritizedMiddleware
	counter     int // 注册顺序计数器
}

// NewRegistrar 创建一个新的中间件注册器。
func NewRegistrar() *Registrar {
	return &Registrar{
		middlewares: make([]PrioritizedMiddleware, 0),
		counter:     0,
	}
}

// Register 注册一个中间件。
//
// 参数：
//   - name: 中间件名称，用于调试和日志
//   - priority: 优先级，建议使用预定义的 Priority 常量
//   - handler: 中间件处理函数
//
// 示例：
//
//	registrar.Register("recovery", PriorityRecovery, recoveryMiddleware)
//	registrar.Register("auth", PriorityAuth, authMiddleware)
func (r *Registrar) Register(name string, priority Priority, handler transport.MiddlewareFunc) {
	if handler == nil {
		panic(fmt.Sprintf("middleware handler cannot be nil for %q", name))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.middlewares = append(r.middlewares, PrioritizedMiddleware{
		Name:     name,
		Priority: priority,
		Handler:  handler,
		order:    r.counter,
	})
	r.counter++
}

// RegisterIf 条件注册中间件，仅在条件为 true 时注册。
//
// 参数：
//   - condition: 是否注册的条件
//   - name: 中间件名称
//   - priority: 优先级
//   - handler: 中间件处理函数
//
// 示例：
//
//	registrar.RegisterIf(enableAuth, "auth", PriorityAuth, authMiddleware)
func (r *Registrar) RegisterIf(condition bool, name string, priority Priority, handler transport.MiddlewareFunc) {
	if condition {
		r.Register(name, priority, handler)
	}
}

// Apply 将所有已注册的中间件按优先级顺序应用到路由器。
// 优先级高的中间件先执行，同优先级按注册顺序执行。
//
// 参数：
//   - router: 要应用中间件的路由器
//
// 注意：
//   - 此方法会按优先级排序，仅在应用时执行一次
//   - 排序后的顺序：Recovery -> RequestID -> Logger -> ... -> Custom
func (r *Registrar) Apply(router transport.Router) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 复制中间件列表以避免排序影响原始列表
	sorted := make([]PrioritizedMiddleware, len(r.middlewares))
	copy(sorted, r.middlewares)

	// 按优先级排序（降序），同优先级按注册顺序（升序）
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].order < sorted[j].order
	})

	// 按顺序应用中间件
	for _, mw := range sorted {
		router.Use(mw.Handler)
	}
}

// Count 返回已注册的中间件数量。
func (r *Registrar) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.middlewares)
}

// List 返回按优先级排序的中间件名称列表（用于调试）。
func (r *Registrar) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 复制并排序
	sorted := make([]PrioritizedMiddleware, len(r.middlewares))
	copy(sorted, r.middlewares)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].order < sorted[j].order
	})

	names := make([]string, len(sorted))
	for i, mw := range sorted {
		names[i] = fmt.Sprintf("%s[%d]", mw.Name, mw.Priority)
	}
	return names
}

// Clear 清空所有已注册的中间件（仅用于测试）。
func (r *Registrar) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = make([]PrioritizedMiddleware, 0)
	r.counter = 0
}
