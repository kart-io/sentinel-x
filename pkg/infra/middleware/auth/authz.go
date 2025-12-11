package auth

import (
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	options "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// // AuthzOptions defines authorization middleware options.
// type AuthzOptions struct {
// 	// Authorizer is the authorizer to use.
// 	Authorizer authz.Authorizer

// 	// ResourceExtractor extracts the resource from the request.
// 	// Default: extracts from request path.
// 	ResourceExtractor func(ctx transport.Context) string

// 	// ActionExtractor extracts the action from the request.
// 	// Default: maps HTTP method to action (GET->read, POST->create, etc.).
// 	ActionExtractor func(ctx transport.Context) string

// 	// SubjectExtractor extracts the subject from the request.
// 	// Default: extracts from auth claims in context.
// 	SubjectExtractor func(ctx transport.Context) string

// 	// SkipPaths is a list of paths to skip authorization.
// 	SkipPaths []string

// 	// SkipPathPrefixes is a list of path prefixes to skip authorization.
// 	SkipPathPrefixes []string

// 	// ErrorHandler is called when authorization fails.
// 	ErrorHandler func(ctx transport.Context, err error)
// }

// AuthzOption is a functional option for authz middleware.
type AuthzOption func(*options.AuthzOptions)

// NewAuthzOptions creates default authz options.
func NewAuthzOptions() *options.AuthzOptions {
	return &options.AuthzOptions{
		ResourceExtractor: defaultResourceExtractor,
		ActionExtractor:   defaultActionExtractor,
		SubjectExtractor:  defaultSubjectExtractor,
		SkipPaths:         []string{},
		SkipPathPrefixes:  []string{},
	}
}

// AuthzWithAuthorizer sets the authorizer.
func AuthzWithAuthorizer(a authz.Authorizer) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.Authorizer = a
	}
}

// AuthzWithResourceExtractor sets the resource extractor.
func AuthzWithResourceExtractor(extractor func(ctx transport.Context) string) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.ResourceExtractor = extractor
	}
}

// AuthzWithActionExtractor sets the action extractor.
func AuthzWithActionExtractor(extractor func(ctx transport.Context) string) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.ActionExtractor = extractor
	}
}

// AuthzWithSubjectExtractor sets the subject extractor.
func AuthzWithSubjectExtractor(extractor func(ctx transport.Context) string) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.SubjectExtractor = extractor
	}
}

// AuthzWithSkipPaths sets paths to skip authorization.
func AuthzWithSkipPaths(paths ...string) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.SkipPaths = paths
	}
}

// AuthzWithSkipPathPrefixes sets path prefixes to skip authorization.
func AuthzWithSkipPathPrefixes(prefixes ...string) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.SkipPathPrefixes = prefixes
	}
}

// AuthzWithErrorHandler sets the error handler.
func AuthzWithErrorHandler(handler func(ctx transport.Context, err error)) AuthzOption {
	return func(o *options.AuthzOptions) {
		o.ErrorHandler = handler
	}
}

// Authz creates an authorization middleware.
func Authz(opts ...AuthzOption) transport.MiddlewareFunc {
	options := NewAuthzOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(ctx transport.Context) {
			// Check if path should be skipped
			path := ctx.HTTPRequest().URL.Path
			if shouldSkipAuthz(path, options.SkipPaths, options.SkipPathPrefixes) {
				next(ctx)
				return
			}

			// Check if authorizer is configured
			if options.Authorizer == nil {
				handleAuthzError(ctx, options, errors.ErrInternal.WithMessage("authorizer not configured"))
				return
			}

			// Extract subject
			subject := options.SubjectExtractor(ctx)
			if subject == "" {
				handleAuthzError(ctx, options, errors.ErrUnauthorized.WithMessage("no subject found"))
				return
			}

			// Extract resource and action
			resource := options.ResourceExtractor(ctx)
			action := options.ActionExtractor(ctx)

			// Check authorization
			allowed, err := options.Authorizer.Authorize(ctx.Request(), subject, resource, action)
			if err != nil {
				// Log authorization error for security audit
				logAuthzFailure(ctx, subject, resource, action, err)
				handleAuthzError(ctx, options, err)
				return
			}

			if !allowed {
				authzErr := errors.ErrNoPermission.WithMessagef(
					"access denied: subject=%s, resource=%s, action=%s",
					subject, resource, action)
				// Log authorization denial for security audit
				logAuthzFailure(ctx, subject, resource, action, authzErr)
				handleAuthzError(ctx, options, authzErr)
				return
			}

			next(ctx)
		}
	}
}

// defaultResourceExtractor extracts the resource from the request path.
func defaultResourceExtractor(ctx transport.Context) string {
	path := ctx.HTTPRequest().URL.Path

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
func defaultActionExtractor(ctx transport.Context) string {
	method := ctx.HTTPRequest().Method
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
func defaultSubjectExtractor(ctx transport.Context) string {
	return auth.SubjectFromContext(ctx.Request())
}

// shouldSkipAuthz checks if the path should skip authorization.
func shouldSkipAuthz(path string, skipPaths, skipPrefixes []string) bool {
	// Check exact match
	for _, p := range skipPaths {
		if path == p {
			return true
		}
	}

	// Check prefix match
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// handleAuthzError handles authorization errors.
func handleAuthzError(ctx transport.Context, options *options.AuthzOptions, err error) {
	if options.ErrorHandler != nil {
		options.ErrorHandler(ctx, err)
		return
	}

	// Default error handling
	errno := errors.FromError(err)
	ctx.JSON(errno.HTTPStatus(), response.Err(errno))
}

// ActionMapping defines custom HTTP method to action mapping.
type ActionMapping map[string]string

// DefaultActionMapping is the default HTTP method to action mapping.
var DefaultActionMapping = ActionMapping{
	"GET":     "read",
	"HEAD":    "read",
	"POST":    "create",
	"PUT":     "update",
	"PATCH":   "update",
	"DELETE":  "delete",
	"OPTIONS": "options",
}

// AuthzWithActionMapping creates an action extractor with custom mapping.
func AuthzWithActionMapping(mapping ActionMapping) AuthzOption {
	return AuthzWithActionExtractor(func(ctx transport.Context) string {
		method := ctx.HTTPRequest().Method
		if action, ok := mapping[method]; ok {
			return action
		}
		return strings.ToLower(method)
	})
}

// logAuthzFailure logs authorization failures for security audit.
// This helps detect unauthorized access attempts and permission violations.
func logAuthzFailure(ctx transport.Context, subject, resource, action string, err error) {
	// Get request information
	req := ctx.HTTPRequest()
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
