// Package observability provides observability middleware.
package observability

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/internal/pathutil"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/requestutil"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// fieldsPool is a sync.Pool for reusing fields slices to reduce heap allocations.
var fieldsPool = sync.Pool{
	New: func() interface{} {
		s := make([]interface{}, 0, 16)
		return &s
	},
}

// acquireFields retrieves a fields slice from the pool.
func acquireFields() *[]interface{} {
	return fieldsPool.Get().(*[]interface{})
}

// releaseFields resets and returns the fields slice to the pool.
func releaseFields(fields *[]interface{}) {
	// Reset slice to zero length but keep capacity
	*fields = (*fields)[:0]
	fieldsPool.Put(fields)
}

// Logger returns a middleware that logs HTTP requests with default options.
func Logger() gin.HandlerFunc {
	return LoggerWithOptions(*mwopts.NewLoggerOptions(), nil)
}

// LoggerWithOptions 返回一个使用纯配置选项和运行时依赖注入的 Logger 中间件。
// 这是推荐的 API，适用于配置中心场景（配置必须可序列化）。
//
// 参数：
//   - opts: 纯配置选项（可 JSON 序列化）
//   - output: 可选的日志输出函数（仅在 UseStructuredLogger=false 时使用）
//     如果为 nil，默认使用 log.Printf
//
// 示例：
//
//	// 使用默认输出
//	middleware.LoggerWithOptions(opts, nil)
//
//	// 自定义输出
//	middleware.LoggerWithOptions(opts, func(format string, args ...interface{}) {
//	    myLogger.Printf(format, args...)
//	})
func LoggerWithOptions(opts mwopts.LoggerOptions, output func(format string, args ...interface{})) gin.HandlerFunc {
	// Set defaults
	if output == nil {
		output = log.Printf
	}

	// 创建路径匹配器（优化性能）
	pathMatcher := pathutil.NewPathMatcher(opts.SkipPaths, nil)

	return func(c *gin.Context) {
		// Get request info
		req := c.Request
		path := req.URL.Path

		// Skip logging for certain paths
		if pathMatcher(path) {
			c.Next()
			return
		}

		// Record start time
		start := time.Now()

		// Get request ID if available
		requestID := requestutil.GetRequestID(c.Request.Context())

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Log the request
		if opts.UseStructuredLogger {
			// Acquire fields slice from pool
			fields := acquireFields()
			defer releaseFields(fields)

			// Populate fields
			*fields = append(*fields,
				"method", req.Method,
				"path", path,
				"remote_addr", req.RemoteAddr,
				"latency", latency.String(),
				"latency_ms", latency.Milliseconds(),
			)
			if requestID != "" {
				*fields = append(*fields, "request_id", requestID)
			}
			logger.Infow("HTTP Request", (*fields)...)
		} else {
			// Use legacy printf-style logging
			if requestID != "" {
				output("[%s] %s %s %s %v",
					requestID,
					req.Method,
					path,
					req.RemoteAddr,
					latency,
				)
			} else {
				output("%s %s %s %v",
					req.Method,
					path,
					req.RemoteAddr,
					latency,
				)
			}
		}
	}
}
