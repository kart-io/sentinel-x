package middleware

import (
	"strings"

	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/kart-io/sentinel-x/pkg/authz"
	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/response"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// AuthzOptions defines authorization middleware options.
type AuthzOptions struct {
	// Authorizer is the authorizer to use.
	Authorizer authz.Authorizer

	// ResourceExtractor extracts the resource from the request.
	// Default: extracts from request path.
	ResourceExtractor func(ctx transport.Context) string

	// ActionExtractor extracts the action from the request.
	// Default: maps HTTP method to action (GET->read, POST->create, etc.).
	ActionExtractor func(ctx transport.Context) string

	// SubjectExtractor extracts the subject from the request.
	// Default: extracts from auth claims in context.
	SubjectExtractor func(ctx transport.Context) string

	// SkipPaths is a list of paths to skip authorization.
	SkipPaths []string

	// SkipPathPrefixes is a list of path prefixes to skip authorization.
	SkipPathPrefixes []string

	// ErrorHandler is called when authorization fails.
	ErrorHandler func(ctx transport.Context, err error)
}

// AuthzOption is a functional option for authz middleware.
type AuthzOption func(*AuthzOptions)

// NewAuthzOptions creates default authz options.
func NewAuthzOptions() *AuthzOptions {
	return &AuthzOptions{
		ResourceExtractor: defaultResourceExtractor,
		ActionExtractor:   defaultActionExtractor,
		SubjectExtractor:  defaultSubjectExtractor,
		SkipPaths:         []string{},
		SkipPathPrefixes:  []string{},
	}
}

// AuthzWithAuthorizer sets the authorizer.
func AuthzWithAuthorizer(a authz.Authorizer) AuthzOption {
	return func(o *AuthzOptions) {
		o.Authorizer = a
	}
}

// AuthzWithResourceExtractor sets the resource extractor.
func AuthzWithResourceExtractor(extractor func(ctx transport.Context) string) AuthzOption {
	return func(o *AuthzOptions) {
		o.ResourceExtractor = extractor
	}
}

// AuthzWithActionExtractor sets the action extractor.
func AuthzWithActionExtractor(extractor func(ctx transport.Context) string) AuthzOption {
	return func(o *AuthzOptions) {
		o.ActionExtractor = extractor
	}
}

// AuthzWithSubjectExtractor sets the subject extractor.
func AuthzWithSubjectExtractor(extractor func(ctx transport.Context) string) AuthzOption {
	return func(o *AuthzOptions) {
		o.SubjectExtractor = extractor
	}
}

// AuthzWithSkipPaths sets paths to skip authorization.
func AuthzWithSkipPaths(paths ...string) AuthzOption {
	return func(o *AuthzOptions) {
		o.SkipPaths = paths
	}
}

// AuthzWithSkipPathPrefixes sets path prefixes to skip authorization.
func AuthzWithSkipPathPrefixes(prefixes ...string) AuthzOption {
	return func(o *AuthzOptions) {
		o.SkipPathPrefixes = prefixes
	}
}

// AuthzWithErrorHandler sets the error handler.
func AuthzWithErrorHandler(handler func(ctx transport.Context, err error)) AuthzOption {
	return func(o *AuthzOptions) {
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
				handleAuthzError(ctx, options, err)
				return
			}

			if !allowed {
				handleAuthzError(ctx, options, errors.ErrNoPermission.WithMessagef(
					"access denied: subject=%s, resource=%s, action=%s",
					subject, resource, action))
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
func handleAuthzError(ctx transport.Context, options *AuthzOptions, err error) {
	if options.ErrorHandler != nil {
		options.ErrorHandler(ctx, err)
		return
	}

	// Default error handling
	errno := errors.FromError(err)
	ctx.JSON(errno.HTTPStatus(), response.Err(errno))
}

// WithAuthz configures and enables authz middleware in Options.
func WithAuthz(opts ...AuthzOption) Option {
	return func(o *Options) {
		o.DisableAuthz = false
		for _, opt := range opts {
			opt(&o.Authz)
		}
	}
}

// WithoutAuthz disables authz middleware.
func WithoutAuthz() Option {
	return func(o *Options) {
		o.DisableAuthz = true
	}
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
