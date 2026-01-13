package resilience

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/internal/pathutil"
	llmresilience "github.com/kart-io/sentinel-x/pkg/llm/resilience"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// CircuitBreaker 返回一个熔断器中间件。
//
// 熔断器用于防止级联失败：当下游服务持续出错时,自动熔断请求,
// 避免资源浪费并给下游服务恢复时间。
//
// 参数:
//   - maxFailures: 触发熔断的最大失败次数
//   - timeout: 熔断器打开后的超时时间(秒)
//   - halfOpenMaxCalls: 半开状态允许的最大调用次数
//
// 示例:
//
//	router.Use(CircuitBreaker(5, 60, 1))
func CircuitBreaker(maxFailures int, timeout, halfOpenMaxCalls int) gin.HandlerFunc {
	return CircuitBreakerWithOptions(mwopts.CircuitBreakerOptions{
		MaxFailures:      maxFailures,
		Timeout:          timeout,
		HalfOpenMaxCalls: halfOpenMaxCalls,
		ErrorThreshold:   500, // 默认 5xx 错误触发熔断
	})
}

// CircuitBreakerWithOptions 返回一个带配置选项的熔断器中间件。
// 这是推荐的构造函数,直接使用 pkg/options/middleware.CircuitBreakerOptions。
//
// 工作原理:
//  1. 关闭状态(Closed): 正常处理请求,记录失败次数
//  2. 打开状态(Open): 失败次数达到阈值,拒绝所有请求
//  3. 半开状态(Half-Open): 超时后允许少量请求探测,成功则关闭,失败则重新打开
//
// HTTP 状态码判定:
//   - >= ErrorThreshold 视为失败(默认 500,即 5xx 错误)
//   - < ErrorThreshold 视为成功
//
// 注意事项:
//   - 熔断器状态是单实例的,不跨实例共享
//   - 跳过的路径(SkipPaths)不会影响熔断器状态
//   - 熔断器打开时返回 503 Service Unavailable
func CircuitBreakerWithOptions(opts mwopts.CircuitBreakerOptions) gin.HandlerFunc {
	// 创建路径匹配器
	pathMatcher := pathutil.NewPathMatcher(opts.SkipPaths, opts.SkipPathPrefixes)

	// 创建熔断器实例
	breaker := llmresilience.NewCircuitBreaker(&llmresilience.CircuitBreakerConfig{
		MaxFailures:      opts.MaxFailures,
		Timeout:          opts.GetTimeout(),
		HalfOpenMaxCalls: opts.HalfOpenMaxCalls,
	})

	return func(c *gin.Context) {
		req := c.Request

		// 检查是否跳过此路径
		if pathMatcher(req.URL.Path) {
			c.Next()
			return
		}

		// 通过熔断器执行请求
		err := breaker.Execute(func() (execErr error) {
			// 捕获 panic 并将其视为失败
			defer func() {
				if r := recover(); r != nil {
					// 记录 panic 为错误，以便触发熔断
					execErr = errors.ErrInternal
					logger.Errorw("circuit breaker caught panic",
						"panic", r,
						"path", req.URL.Path,
					)
					// 重新抛出 panic，让 Recovery 中间件处理
					// 注意：这里必须重新 panic，否则 Recovery 无法捕获堆栈信息
					// 而 execErr 的赋值已经确保熔断器记录了这次失败
					panic(r)
				}
			}()

			// 调用下一个处理器
			c.Next()

			// 获取响应状态码
			// gin.ResponseWriter 确保有 Status() 方法,直接调用是安全的
			statusCode := c.Writer.Status()
			if statusCode == 0 {
				// 如果未写入响应,默认为 200
				statusCode = http.StatusOK
			}

			// 根据 HTTP 状态码判断是否失败
			if statusCode >= opts.ErrorThreshold {
				logger.Debugw("circuit breaker detected error response",
					"path", req.URL.Path,
					"status_code", statusCode,
					"threshold", opts.ErrorThreshold,
				)
				return errors.ErrInternal
			}
			return nil
		})

		// 如果熔断器打开,返回 503
		if err == llmresilience.ErrCircuitBreakerOpen {
			logger.Warnw("circuit breaker open, rejecting request",
				"path", req.URL.Path,
				"state", breaker.State().String(),
				"stats", breaker.Stats(),
			)
			response.Fail(c, errors.ErrServiceUnavailable)
			return
		}
	}
}
