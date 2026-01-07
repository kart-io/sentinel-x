package security

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// Security header constants.
const (
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderReferrerPolicy          = "Referrer-Policy"
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
)

// SecurityHeaders returns a middleware that adds security headers with default options.
func SecurityHeaders() gin.HandlerFunc {
	return SecurityHeadersWithOptions(*mwopts.NewSecurityHeadersOptions())
}

// SecurityHeadersWithOptions returns a middleware that adds security headers with custom options.
// 这是推荐的 API，使用纯配置选项。
//
// 参数：
//   - opts: SecurityHeaders 配置选项（纯配置，可 JSON 序列化）
//
// 示例：
//
//	opts := mwopts.NewSecurityHeadersOptions()
//	opts.EnableHSTS = true
//	middleware.SecurityHeadersWithOptions(*opts)
func SecurityHeadersWithOptions(opts mwopts.SecurityHeadersOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add HSTS header
		if opts.EnableHSTS && isHTTPSConnection(c) {
			hsts := fmt.Sprintf("max-age=%d", opts.HSTSMaxAge)
			if opts.HSTSIncludeSubdomains {
				hsts += "; includeSubDomains"
			}
			if opts.HSTSPreload {
				hsts += "; preload"
			}
			c.Header(HeaderStrictTransportSecurity, hsts)
		}

		// Add X-Frame-Options header
		if opts.EnableFrameOptions {
			c.Header(HeaderXFrameOptions, opts.FrameOptionsValue)
		}

		// Add X-Content-Type-Options header
		if opts.EnableContentTypeOptions {
			c.Header(HeaderXContentTypeOptions, "nosniff")
		}

		// Add X-XSS-Protection header
		if opts.EnableXSSProtection {
			c.Header(HeaderXXSSProtection, opts.XSSProtectionValue)
		}

		// Add Content-Security-Policy header
		if opts.ContentSecurityPolicy != "" {
			c.Header(HeaderContentSecurityPolicy, opts.ContentSecurityPolicy)
		}

		// Add Referrer-Policy header
		if opts.ReferrerPolicy != "" {
			c.Header(HeaderReferrerPolicy, opts.ReferrerPolicy)
		}

		c.Next()
	}
}

// isHTTPSConnection checks if the current connection is using HTTPS protocol.
// This function examines multiple indicators to determine HTTPS status:
//   - Direct TLS connection (req.TLS != nil)
//   - X-Forwarded-Proto header (for reverse proxy scenarios)
//
// Returns true if the connection is HTTPS, false otherwise.
func isHTTPSConnection(c *gin.Context) bool {
	req := c.Request

	// Check if TLS is enabled
	if req.TLS != nil {
		return true
	}

	// Check X-Forwarded-Proto header (for reverse proxy scenarios)
	proto := req.Header.Get("X-Forwarded-Proto")
	return strings.ToLower(proto) == "https"
}
