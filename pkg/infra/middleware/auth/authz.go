package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/middleware/internal/pathutil"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// authzOptions 是内部使用的授权选项结构（非导出）。
type authzOptions struct {
	authorizer        authz.Authorizer
	resourceExtractor func(ctx *gin.Context) string
	actionExtractor   func(ctx *gin.Context) string
	subjectExtractor  func(ctx *gin.Context) string
	skipPaths         []string
	skipPathPrefixes  []string
	errorHandler      func(ctx *gin.Context, err error)
}

// AuthzWithOptions 返回一个使用纯配置选项和运行时依赖注入的 Authz 中间件。
// 这是推荐的 API，适用于配置中心场景（配置必须可序列化）。
//
// 参数：
//   - opts: 纯配置选项（可 JSON 序列化）
//   - authorizer: 授权器（运行时依赖）
//   - resourceExtractor: 可选的资源提取器，如果为 nil 使用默认实现
//   - actionExtractor: 可选的动作提取器，如果为 nil 使用默认实现
//   - subjectExtractor: 可选的主体提取器，如果为 nil 使用默认实现
//   - errorHandler: 可选的错误处理器
//
// 示例：
//
//	opts := mwopts.NewAuthzOptions()
//	middleware := AuthzWithOptions(
//	    *opts,
//	    myAuthorizer,
//	    nil,  // 使用默认资源提取器
//	    nil,  // 使用默认动作提取器
//	    nil,  // 使用默认主体提取器
//	    nil,  // 使用默认错误处理
//	)
func AuthzWithOptions(
	opts mwopts.AuthzOptions,
	authorizer authz.Authorizer,
	resourceExtractor func(ctx *gin.Context) string,
	actionExtractor func(ctx *gin.Context) string,
	subjectExtractor func(ctx *gin.Context) string,
	errorHandler func(ctx *gin.Context, err error),
) gin.HandlerFunc {
	// 使用默认提取器
	if resourceExtractor == nil {
		resourceExtractor = defaultResourceExtractor
	}
	if actionExtractor == nil {
		actionExtractor = defaultActionExtractor
	}
	if subjectExtractor == nil {
		subjectExtractor = defaultSubjectExtractor
	}

	o := &authzOptions{
		authorizer:        authorizer,
		resourceExtractor: resourceExtractor,
		actionExtractor:   actionExtractor,
		subjectExtractor:  subjectExtractor,
		skipPaths:         opts.SkipPaths,
		skipPathPrefixes:  opts.SkipPathPrefixes,
		errorHandler:      errorHandler,
	}

	return func(c *gin.Context) {
		// Check if path should be skipped
		path := c.Request.URL.Path
		if shouldSkipAuthz(path, o.skipPaths, o.skipPathPrefixes) {
			c.Next()
			return
		}

		// Check if authorizer is configured
		if o.authorizer == nil {
			handleAuthzError(c, o, errors.ErrInternal.WithMessage("authorizer not configured"))
			return
		}

		// Extract subject
		subject := o.subjectExtractor(c)
		if subject == "" {
			handleAuthzError(c, o, errors.ErrUnauthorized.WithMessage("no subject found"))
			return
		}

		// Extract resource and action
		resource := o.resourceExtractor(c)
		action := o.actionExtractor(c)

		// Check authorization
		allowed, err := o.authorizer.Authorize(c.Request.Context(), subject, resource, action)
		if err != nil {
			// Log authorization error for security audit
			logAuthzFailure(c, subject, resource, action, err)
			handleAuthzError(c, o, err)
			return
		}

		if !allowed {
			authzErr := errors.ErrNoPermission.WithMessagef(
				"access denied: subject=%s, resource=%s, action=%s",
				subject, resource, action)
			// Log authorization denial for security audit
			logAuthzFailure(c, subject, resource, action, authzErr)
			handleAuthzError(c, o, authzErr)
			return
		}

		// Debug: log successful authz
		logger.Infow("authorization successful",
			"subject", subject,
			"resource", resource,
			"action", action,
			"path", path,
		)

		c.Next()
	}
}

// defaultResourceExtractor extracts the resource from the request path.
func defaultResourceExtractor(c *gin.Context) string {
	path := c.Request.URL.Path

	// Remove leading slash and API prefix
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "api/")
	path = strings.TrimPrefix(path, "v1/")
	path = strings.TrimPrefix(path, "v2/")

	// Get the first path segment as resource
	parts := strings.SplitN(path, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}

	return path
}

// defaultActionExtractor maps HTTP method to action.
func defaultActionExtractor(c *gin.Context) string {
	method := c.Request.Method
	switch method {
	case "GET":
		return "read"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// defaultSubjectExtractor extracts the subject from auth claims.
func defaultSubjectExtractor(c *gin.Context) string {
	return auth.SubjectFromContext(c.Request.Context())
}

// shouldSkipAuthz checks if the path should skip authorization.
func shouldSkipAuthz(path string, skipPaths, skipPrefixes []string) bool {
	return pathutil.ShouldSkip(path, skipPaths, skipPrefixes)
}

// handleAuthzError handles authorization errors.
func handleAuthzError(c *gin.Context, o *authzOptions, err error) {
	if o.errorHandler != nil {
		o.errorHandler(c, err)
		return
	}

	// Default error handling
	errno := errors.FromError(err)
	c.JSON(errno.HTTPStatus(), response.Err(errno))
}

// logAuthzFailure logs authorization failures for security audit.
// This helps detect unauthorized access attempts and permission violations.
func logAuthzFailure(c *gin.Context, subject, resource, action string, err error) {
	// Get request information
	req := c.Request
	if req == nil {
		return
	}

	// Log using structured logger with security-relevant fields
	logger.Warnw("authorization denied",
		"subject", subject,
		"resource", resource,
		"action", action,
		"error", err.Error(),
		"remote_addr", req.RemoteAddr,
		"path", req.URL.Path,
		"method", req.Method,
		"user_agent", req.UserAgent(),
	)
}
