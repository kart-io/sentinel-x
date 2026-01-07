// Package security provides security middleware.
package security

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// CORS returns a middleware that adds CORS headers.
func CORS() gin.HandlerFunc {
	return CORSWithOptions(*mwopts.NewCORSOptions())
}

// validateCORSOptions validates CORS options.
// 提取验证逻辑以便 CORSConfig.Validate() 和 CORSWithOptions 共用。
func validateCORSOptions(opts mwopts.CORSOptions) error {
	if len(opts.AllowOrigins) == 0 {
		return fmt.Errorf("CORS: AllowOrigins must be explicitly configured, empty list not allowed")
	}

	// Check for wildcard and credentials conflict
	hasWildcard := false
	for _, origin := range opts.AllowOrigins {
		if origin == "*" {
			hasWildcard = true
		}

		// Validate origin format (if not wildcard)
		if origin != "*" {
			if err := validateOriginFormat(origin); err != nil {
				return fmt.Errorf("CORS: invalid origin format '%s': %w", origin, err)
			}
		}
	}

	// Wildcard cannot be used with credentials
	if hasWildcard && opts.AllowCredentials {
		return fmt.Errorf("CORS: cannot use wildcard origin '*' with AllowCredentials=true (RFC6454 security requirement)")
	}

	return nil
}

// validateOriginFormat validates that an origin follows the correct URL format.
// Origins must be in the format: scheme://host[:port]
func validateOriginFormat(origin string) error {
	if origin == "" {
		return fmt.Errorf("origin cannot be empty")
	}

	// Check for scheme
	if !strings.Contains(origin, "://") {
		return fmt.Errorf("origin must include scheme (http:// or https://)")
	}

	// Check for path, query, or fragment (origins should not include these)
	schemeEnd := strings.Index(origin, "://") + 3
	if schemeEnd < len(origin) {
		remainder := origin[schemeEnd:]
		if strings.ContainsAny(remainder, "/?#") {
			return fmt.Errorf("origin should not include path, query, or fragment")
		}
	}

	return nil
}

// CORSWithOptions returns a CORS middleware with CORSOptions.
// 这是推荐的构造函数，直接使用 pkg/options/middleware.CORSOptions。
func CORSWithOptions(opts mwopts.CORSOptions) gin.HandlerFunc {
	// Validate configuration
	if err := validateCORSOptions(opts); err != nil {
		panic(err) // 配置错误应该在启动时失败
	}

	// Set defaults (NewCORSOptions 已经设置了默认值，这里只做兜底)
	if len(opts.AllowOrigins) == 0 {
		opts.AllowOrigins = []string{"*"}
	}
	if len(opts.AllowMethods) == 0 {
		opts.AllowMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		}
	}
	if len(opts.AllowHeaders) == 0 {
		opts.AllowHeaders = []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
		}
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 86400
	}

	allowMethods := strings.Join(opts.AllowMethods, ", ")
	allowHeaders := strings.Join(opts.AllowHeaders, ", ")
	exposeHeaders := strings.Join(opts.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(opts.MaxAge)

	return func(c *gin.Context) {
		req := c.Request
		origin := req.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := ""
		for _, o := range opts.AllowOrigins {
			if o == "*" || o == origin {
				allowedOrigin = o
				break
			}
		}

		if allowedOrigin == "" {
			c.Next()
			return
		}

		// Set CORS headers
		c.Header("Access-Control-Allow-Origin", allowedOrigin)

		if opts.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if exposeHeaders != "" {
			c.Header("Access-Control-Expose-Headers", exposeHeaders)
		}

		// Handle preflight request
		if req.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)
			c.Header("Access-Control-Max-Age", maxAge)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
