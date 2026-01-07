package resilience

import (
	"net/http"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/internal/pathutil"
	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// BodyLimit 返回一个请求体大小限制中间件。
// 用于防止恶意客户端发送超大请求体导致服务器资源耗尽（DoS 攻击）。
//
// 参数：
//   - maxSize: 最大请求体大小（字节）
//
// 示例：
//
//	router.Use(BodyLimit(10 * 1024 * 1024)) // 限制 10MB
func BodyLimit(maxSize int64) gin.HandlerFunc {
	return BodyLimitWithOptions(mwopts.BodyLimitOptions{
		MaxSize: maxSize,
	})
}

// BodyLimitWithOptions 返回一个带配置选项的请求体大小限制中间件。
// 这是推荐的构造函数，直接使用 pkg/options/middleware.BodyLimitOptions。
//
// 工作原理：
//  1. 检查 Content-Length 头，如果超过限制立即拒绝
//  2. 使用 http.MaxBytesReader 限制实际读取的字节数
//  3. 支持跳过特定路径（如文件上传路径）
//
// 注意事项：
//   - 必须在读取请求体的中间件之前执行
//   - 对于文件上传路径，建议配置到 SkipPaths 中并单独处理
//   - Content-Length 头可能被客户端伪造，实际读取时仍会限制
func BodyLimitWithOptions(opts mwopts.BodyLimitOptions) gin.HandlerFunc {
	// 设置默认值
	if opts.MaxSize <= 0 {
		opts.MaxSize = 4 * 1024 * 1024 // 默认 4MB
	}

	// 创建路径匹配器
	pathMatcher := pathutil.NewPathMatcher(opts.SkipPaths, opts.SkipPathPrefixes)
	return func(c *gin.Context) {
			req := c.Request

			// 检查是否跳过此路径（精确匹配）
			if pathMatcher(req.URL.Path) {
				c.Next()
				return
			}

			// 早期检查：如果 Content-Length 头已经超过限制，立即拒绝
			// 这样可以避免读取任何数据，节省资源
			if req.ContentLength > opts.MaxSize {
				logger.Warnw("request body too large (Content-Length check)",
					"path", req.URL.Path,
					"content_length", req.ContentLength,
					"max_size", opts.MaxSize,
				)
				response.Fail(c, errors.ErrRequestTooLarge)
				return
			}

			// 使用 http.MaxBytesReader 限制实际读取的字节数
			// 即使客户端没有设置 Content-Length 或设置了错误的值，
			// 在读取超过 MaxSize 字节时也会返回错误
			req.Body = http.MaxBytesReader(c.Writer, req.Body, opts.MaxSize)

			// 继续处理请求
			c.Next()
		}
}
