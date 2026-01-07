package resilience

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// PanicHandler 定义 panic 处理器类型。
// 参数：
//   - ctx: 请求上下文
//   - err: panic 值
//   - stack: 堆栈跟踪信息
type PanicHandler func(ctx *gin.Context, err interface{}, stack []byte)

// Recovery returns a middleware that recovers from panics with default options.
func Recovery() gin.HandlerFunc {
	return RecoveryWithOptions(*mwopts.NewRecoveryOptions(), nil)
}

// RecoveryWithOptions 返回一个使用纯配置选项和运行时依赖注入的 Recovery 中间件。
// 这是推荐的 API，适用于配置中心场景（配置必须可序列化）。
//
// 参数：
//   - opts: 纯配置选项（可 JSON 序列化）
//   - onPanic: 可选的 panic 处理器，用于自定义日志或告警逻辑
//     如果为 nil，则不执行额外处理（仅记录日志和返回错误响应）
//
// 示例：
//
//	// 使用默认行为
//	middleware.RecoveryWithOptions(opts, nil)
//
//	// 自定义 panic 处理器
//	middleware.RecoveryWithOptions(opts, func(ctx *gin.Context, err interface{}, stack []byte) {
//	    // 发送告警到监控系统
//	    alerting.SendPanicAlert(err, stack)
//	})
func RecoveryWithOptions(opts mwopts.RecoveryOptions, onPanic PanicHandler) gin.HandlerFunc {
	// Check if running in production environment
	isProd := isProductionEnvironment()

	// Validate and adjust config for production environment
	shouldReturnStackToClient := validateStackTraceConfig(opts.EnableStackTrace, isProd)

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				// Always log full stack trace to logs
				logPanicWithStackTrace(c, r, stack)

				// Call panic handler if configured
				if onPanic != nil {
					onPanic(c, r, stack)
				}

				// Build client error response
				err := buildClientErrorResponse(r, stack, shouldReturnStackToClient)

				response.Fail(c, err)
			}
		}()
		c.Next()
	}
}

// isProductionEnvironment checks if the application is running in production environment.
// It checks APP_ENV or GO_ENV environment variables.
func isProductionEnvironment() bool {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}

	// Convert to lowercase for case-insensitive comparison
	switch env {
	case "production", "prod", "PRODUCTION", "PROD":
		return true
	default:
		return false
	}
}

// validateStackTraceConfig validates and adjusts EnableStackTrace config based on environment.
// In production, it enforces disabling stack trace in client responses with a warning.
func validateStackTraceConfig(enableStackTrace bool, isProd bool) bool {
	if isProd && enableStackTrace {
		logger.Warn("Stack trace is enabled but running in production environment. " +
			"Stack trace will NOT be returned to clients for security reasons. " +
			"Full stack trace will still be logged.")
		return false
	}
	return enableStackTrace
}

// logPanicWithStackTrace logs the panic with full stack trace information.
// This ensures complete debugging information is always available in logs.
func logPanicWithStackTrace(c *gin.Context, panicValue interface{}, stack []byte) {
	logger.Errorw("panic recovered",
		"panic", panicValue,
		"stack_trace", string(stack),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)
}

// buildClientErrorResponse builds the error response to be returned to the client.
// Stack trace is only included in non-production environments when explicitly enabled.
func buildClientErrorResponse(panicValue interface{}, stack []byte, includeStackTrace bool) *errors.Errno {
	if includeStackTrace {
		// Development environment: include full stack trace for debugging
		return errors.ErrPanic.WithMessage(fmt.Sprintf("panic: %v\n%s", panicValue, string(stack)))
	}

	// Production environment: return generic error message only
	return errors.ErrPanic.WithMessage(fmt.Sprintf("panic: %v", panicValue))
}
