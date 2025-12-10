package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// CORSConfig defines the config for CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a list of origins that may access the resource.
	// Default: ["*"]
	AllowOrigins []string

	// AllowMethods is a list of methods allowed when accessing the resource.
	// Default: ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]
	AllowMethods []string

	// AllowHeaders is a list of headers that can be used when making the request.
	// Default: ["Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"]
	AllowHeaders []string

	// ExposeHeaders is a list of headers that browsers are allowed to access.
	// Default: []
	ExposeHeaders []string

	// AllowCredentials indicates whether credentials are allowed.
	// Default: false
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached.
	// Default: 86400 (24 hours)
	MaxAge int
}

// DefaultCORSConfig is the default CORS middleware config.
// NOTE: AllowOrigins is empty by default and must be explicitly configured.
// This forces developers to consciously configure allowed origins for security.
var DefaultCORSConfig = CORSConfig{
	AllowOrigins: []string{}, // 强制显式配置，不允许空配置运行
	AllowMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	},
	AllowHeaders: []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-Request-ID",
	},
	ExposeHeaders:    []string{},
	AllowCredentials: false,
	MaxAge:           86400,
}

// CORS returns a middleware that adds CORS headers.
func CORS() transport.MiddlewareFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// Validate checks if the CORS configuration is valid.
func (c CORSConfig) Validate() error {
	if len(c.AllowOrigins) == 0 {
		return fmt.Errorf("CORS: AllowOrigins must be explicitly configured, empty list not allowed")
	}

	// 禁止 * 与 credentials 同时使用
	hasWildcard := false
	for _, origin := range c.AllowOrigins {
		if origin == "*" {
			hasWildcard = true
			break
		}
	}
	if hasWildcard && c.AllowCredentials {
		return fmt.Errorf("CORS: cannot use wildcard origin '*' with AllowCredentials=true")
	}

	return nil
}

// CORSWithConfig returns a CORS middleware with custom config.
func CORSWithConfig(config CORSConfig) transport.MiddlewareFunc {
	// Validate configuration
	if err := config.Validate(); err != nil {
		panic(err) // 配置错误应该在启动时失败
	}

	// Set defaults
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = DefaultCORSConfig.AllowHeaders
	}
	if config.MaxAge == 0 {
		config.MaxAge = DefaultCORSConfig.MaxAge
	}

	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			req := c.HTTPRequest()
			origin := req.Header.Get("Origin")

			// Check if origin is allowed
			allowedOrigin := ""
			for _, o := range config.AllowOrigins {
				if o == "*" || o == origin {
					allowedOrigin = o
					break
				}
			}

			if allowedOrigin == "" {
				next(c)
				return
			}

			// Set CORS headers
			c.SetHeader("Access-Control-Allow-Origin", allowedOrigin)

			if config.AllowCredentials {
				c.SetHeader("Access-Control-Allow-Credentials", "true")
			}

			if exposeHeaders != "" {
				c.SetHeader("Access-Control-Expose-Headers", exposeHeaders)
			}

			// Handle preflight request
			if req.Method == http.MethodOptions {
				c.SetHeader("Access-Control-Allow-Methods", allowMethods)
				c.SetHeader("Access-Control-Allow-Headers", allowHeaders)
				c.SetHeader("Access-Control-Max-Age", maxAge)
				c.JSON(http.StatusNoContent, nil)
				return
			}

			next(c)
		}
	}
}
