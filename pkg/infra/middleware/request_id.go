package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/requestutil"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// HeaderXRequestID is re-exported from common for backward compatibility.
const HeaderXRequestID = requestutil.HeaderXRequestID

// GetRequestID returns the request ID from the context.
// Returns empty string if not found.
// This is re-exported from common for backward compatibility.
var GetRequestID = requestutil.GetRequestID

// RequestIDGenerator 定义请求 ID 生成器类型。
type RequestIDGenerator func() string

// RequestID returns a middleware that adds a unique request ID to each request with default options.
func RequestID() transport.MiddlewareFunc {
	return RequestIDWithOptions(*mwopts.NewRequestIDOptions(), nil)
}

// RequestIDWithOptions 返回一个使用纯配置选项和运行时依赖注入的 RequestID 中间件。
// 这是推荐的 API，适用于配置中心场景（配置必须可序列化）。
//
// 参数：
//   - opts: 纯配置选项（可 JSON 序列化）
//   - generator: 可选的请求 ID 生成器
//     如果为 nil，默认使用 requestutil.GenerateRequestID（生成 16 字节随机十六进制字符串）
//
// 示例：
//
//	// 使用默认生成器
//	middleware.RequestIDWithOptions(opts, nil)
//
//	// 使用 UUID 生成器
//	middleware.RequestIDWithOptions(opts, func() string {
//	    return uuid.New().String()
//	})
func RequestIDWithOptions(opts mwopts.RequestIDOptions, generator RequestIDGenerator) transport.MiddlewareFunc {
	// Set defaults
	header := opts.Header
	if header == "" {
		header = HeaderXRequestID
	}

	if generator == nil {
		generator = requestutil.GenerateRequestID
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			// Check if request ID already exists in header
			requestID := c.Header(header)
			if requestID == "" {
				requestID = generator()
			}

			// Set request ID in response header
			c.SetHeader(header, requestID)

			// Store request ID in context using common package
			ctx := requestutil.WithRequestID(c.Request(), requestID)
			c.SetRequest(ctx)

			next(c)
		}
	}
}
