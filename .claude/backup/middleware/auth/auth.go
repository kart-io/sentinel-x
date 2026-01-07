// Package auth provides authentication middleware.
package auth

import (
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/internal/pathutil"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// options 是内部使用的选项结构（非导出）。
type options struct {
	authenticator    auth.Authenticator
	tokenLookup      string
	authScheme       string
	skipPaths        []string
	skipPathPrefixes []string
	errorHandler     func(ctx transport.Context, err error)
	successHandler   func(ctx transport.Context, claims *auth.Claims)
}

// AuthWithOptions 返回一个使用纯配置选项和运行时依赖注入的 Auth 中间件。
// 这是推荐的 API，适用于配置中心场景（配置必须可序列化）。
//
// 参数：
//   - opts: 纯配置选项（可 JSON 序列化）
//   - authenticator: 身份验证器（运行时依赖）
//   - errorHandler: 可选的错误处理器
//   - successHandler: 可选的成功处理器
//
// 示例：
//
//	opts := mwopts.NewAuthOptions()
//	middleware := AuthWithOptions(
//	    *opts,
//	    myAuthenticator,
//	    nil,  // 使用默认错误处理
//	    func(ctx transport.Context, claims *auth.Claims) {
//	        // 自定义成功处理
//	    },
//	)
func AuthWithOptions(
	opts mwopts.AuthOptions,
	authenticator auth.Authenticator,
	errorHandler func(ctx transport.Context, err error),
	successHandler func(ctx transport.Context, claims *auth.Claims),
) transport.MiddlewareFunc {
	o := &options{
		authenticator:    authenticator,
		tokenLookup:      opts.TokenLookup,
		authScheme:       opts.AuthScheme,
		skipPaths:        opts.SkipPaths,
		skipPathPrefixes: opts.SkipPathPrefixes,
		errorHandler:     errorHandler,
		successHandler:   successHandler,
	}
	return authMiddleware(o)
}

// authMiddleware 是实际的中间件实现逻辑。
func authMiddleware(o *options) transport.MiddlewareFunc {
	// Parse token lookup
	lookup := parseTokenLookup(o.tokenLookup)

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(ctx transport.Context) {
			// Check if path should be skipped
			path := ctx.HTTPRequest().URL.Path
			if shouldSkipAuth(path, o.skipPaths, o.skipPathPrefixes) {
				next(ctx)
				return
			}

			// Check if authenticator is configured
			if o.authenticator == nil {
				handleAuthError(ctx, o, errors.ErrInternal.WithMessage("authenticator not configured"))
				return
			}

			// Extract token
			tokenString := extractToken(ctx, lookup, o.authScheme)
			if tokenString == "" {
				handleAuthError(ctx, o, errors.ErrUnauthorized.WithMessage("missing authentication token"))
				return
			}

			// Verify token
			claims, err := o.authenticator.Verify(ctx.Request(), tokenString)
			if err != nil {
				// Log authentication failure for security audit
				logAuthFailure(ctx, tokenString, err)
				handleAuthError(ctx, o, err)
				return
			}

			// Debug: log successful auth
			logger.Infow("authentication successful",
				"subject", claims.Subject,
				"path", path,
			)

			// Inject claims into context
			newCtx := auth.InjectAuth(ctx.Request(), claims, tokenString)
			ctx.SetRequest(newCtx)

			// Call success handler if set
			if o.successHandler != nil {
				o.successHandler(ctx, claims)
			}

			next(ctx)
		}
	}
}

// tokenLookup represents a token extraction method.
type tokenLookup struct {
	source string // "header", "query", "cookie"
	name   string // name of the header/query/cookie
}

// parseTokenLookup parses the token lookup string.
func parseTokenLookup(lookup string) tokenLookup {
	parts := strings.SplitN(lookup, ":", 2)
	if len(parts) != 2 {
		return tokenLookup{source: "header", name: "Authorization"}
	}
	return tokenLookup{source: parts[0], name: parts[1]}
}

// extractToken extracts the token from the request.
func extractToken(ctx transport.Context, lookup tokenLookup, scheme string) string {
	var token string

	switch lookup.source {
	case "header":
		token = ctx.Header(lookup.name)
		if scheme != "" && strings.HasPrefix(token, scheme+" ") {
			token = strings.TrimPrefix(token, scheme+" ")
		}
	case "query":
		token = ctx.Query(lookup.name)
	case "cookie":
		if cookie, err := ctx.HTTPRequest().Cookie(lookup.name); err == nil {
			token = cookie.Value
		}
	}

	// Sanitize token
	token = strings.ReplaceAll(token, " ", "")
	token = strings.ReplaceAll(token, "+", "-")
	token = strings.ReplaceAll(token, "/", "_")
	token = strings.TrimRight(token, "=")
	return token
}

// shouldSkipAuth checks if the path should skip authentication.
func shouldSkipAuth(path string, skipPaths, skipPrefixes []string) bool {
	return pathutil.ShouldSkip(path, skipPaths, skipPrefixes)
}

// handleAuthError handles authentication errors.
func handleAuthError(ctx transport.Context, o *options, err error) {
	if o.errorHandler != nil {
		o.errorHandler(ctx, err)
		return
	}

	// Default error handling
	errno := errors.FromError(err)
	ctx.JSON(errno.HTTPStatus(), response.Err(errno))
}

// logAuthFailure logs authentication failures for security audit.
// This helps detect brute force attacks, token forgery attempts, and other security issues.
func logAuthFailure(ctx transport.Context, token string, err error) {
	// Get request information
	req := ctx.HTTPRequest()
	if req == nil {
		return
	}

	// Only record token prefix to avoid leaking complete token in logs
	tokenPrefix := ""
	if len(token) > 20 {
		tokenPrefix = token[:20] + "..."
	} else if len(token) > 0 {
		tokenPrefix = token[:len(token)/2] + "..."
	}

	// Log using structured logger with security-relevant fields
	logger.Warnw("authentication failed",
		"error", err.Error(),
		"remote_addr", req.RemoteAddr,
		"token_prefix", tokenPrefix,
		"path", req.URL.Path,
		"method", req.Method,
		"user_agent", req.UserAgent(),
	)
}
