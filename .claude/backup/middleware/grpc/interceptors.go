// Package grpc provides gRPC interceptors for authentication and authorization.
//
// This package implements gRPC unary and stream interceptors that integrate
// with the auth and authz packages for unified security across HTTP and gRPC.
//
// Token is extracted from gRPC metadata (Authorization header) and validated
// using the configured Authenticator. Claims are then injected into the context.
//
// Usage:
//
//	// Create JWT authenticator
//	jwtAuth, _ := jwt.New(jwt.WithKey("secret-key"))
//
//	// Create RBAC authorizer
//	rbacAuthz := rbac.New()
//
//	// Create gRPC server with interceptors
//	server := grpc.NewServer(
//	    grpc.UnaryInterceptor(grpcmw.ChainUnaryInterceptors(
//	        grpcmw.UnaryAuthInterceptor(jwtAuth),
//	        grpcmw.UnaryAuthzInterceptor(rbacAuthz),
//	    )),
//	    grpc.StreamInterceptor(grpcmw.ChainStreamInterceptors(
//	        grpcmw.StreamAuthInterceptor(jwtAuth),
//	        grpcmw.StreamAuthzInterceptor(rbacAuthz),
//	    )),
//	)
package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

const (
	// AuthorizationHeader is the metadata key for authorization token.
	AuthorizationHeader = "authorization"

	// BearerScheme is the bearer token scheme.
	BearerScheme = "bearer"
)

// AuthInterceptorOptions configures the auth interceptor.
type AuthInterceptorOptions struct {
	// Authenticator is the authenticator to use.
	Authenticator auth.Authenticator

	// SkipMethods is a list of full method names to skip authentication.
	// Format: "/package.Service/Method"
	SkipMethods []string

	// MetadataKey is the metadata key to read the token from.
	// Default: "authorization"
	MetadataKey string

	// TokenScheme is the expected token scheme.
	// Default: "bearer"
	TokenScheme string
}

// AuthInterceptorOption is a functional option for auth interceptor.
type AuthInterceptorOption func(*AuthInterceptorOptions)

// NewAuthInterceptorOptions creates default auth interceptor options.
func NewAuthInterceptorOptions() *AuthInterceptorOptions {
	return &AuthInterceptorOptions{
		MetadataKey: AuthorizationHeader,
		TokenScheme: BearerScheme,
		SkipMethods: []string{},
	}
}

// WithAuthenticator sets the authenticator.
func WithAuthenticator(a auth.Authenticator) AuthInterceptorOption {
	return func(o *AuthInterceptorOptions) {
		o.Authenticator = a
	}
}

// WithSkipMethods sets methods to skip authentication.
func WithSkipMethods(methods ...string) AuthInterceptorOption {
	return func(o *AuthInterceptorOptions) {
		o.SkipMethods = methods
	}
}

// WithMetadataKey sets the metadata key for token extraction.
func WithMetadataKey(key string) AuthInterceptorOption {
	return func(o *AuthInterceptorOptions) {
		o.MetadataKey = key
	}
}

// WithTokenScheme sets the expected token scheme.
func WithTokenScheme(scheme string) AuthInterceptorOption {
	return func(o *AuthInterceptorOptions) {
		o.TokenScheme = scheme
	}
}

// UnaryAuthInterceptor creates a unary authentication interceptor.
func UnaryAuthInterceptor(opts ...AuthInterceptorOption) grpc.UnaryServerInterceptor {
	options := NewAuthInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	skipMap := make(map[string]struct{})
	for _, method := range options.SkipMethods {
		skipMap[method] = struct{}{}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method should be skipped
		if _, skip := skipMap[info.FullMethod]; skip {
			return handler(ctx, req)
		}

		// Authenticate
		newCtx, err := authenticate(ctx, options)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// StreamAuthInterceptor creates a stream authentication interceptor.
func StreamAuthInterceptor(opts ...AuthInterceptorOption) grpc.StreamServerInterceptor {
	options := NewAuthInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	skipMap := make(map[string]struct{})
	for _, method := range options.SkipMethods {
		skipMap[method] = struct{}{}
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method should be skipped
		if _, skip := skipMap[info.FullMethod]; skip {
			return handler(srv, ss)
		}

		// Authenticate
		ctx, err := authenticate(ss.Context(), options)
		if err != nil {
			return err
		}

		// Wrap stream with new context
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrapped)
	}
}

// authenticate performs authentication and returns a new context with claims.
func authenticate(ctx context.Context, opts *AuthInterceptorOptions) (context.Context, error) {
	if opts.Authenticator == nil {
		return nil, status.Error(codes.Internal, "authenticator not configured")
	}

	// Extract token from metadata
	tokenString, err := extractTokenFromMetadata(ctx, opts.MetadataKey, opts.TokenScheme)
	if err != nil {
		return nil, err
	}

	// Verify token
	claims, err := opts.Authenticator.Verify(ctx, tokenString)
	if err != nil {
		return nil, mapAuthError(err)
	}

	// Inject claims into context
	return auth.InjectAuth(ctx, claims, tokenString), nil
}

// extractTokenFromMetadata extracts the token from gRPC metadata.
func extractTokenFromMetadata(ctx context.Context, key, scheme string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get(key)
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}

	token := values[0]

	// Remove scheme prefix if present
	if scheme != "" {
		prefix := scheme + " "
		if strings.HasPrefix(strings.ToLower(token), strings.ToLower(prefix)) {
			token = token[len(prefix):]
		}
	}

	return strings.TrimSpace(token), nil
}

// mapAuthError maps auth errors to gRPC status.
func mapAuthError(err error) error {
	if errno, ok := err.(*errors.Errno); ok {
		return status.Error(errno.GRPCStatus(), errno.MessageEN)
	}
	return status.Error(codes.Unauthenticated, err.Error())
}

// AuthzInterceptorOptions configures the authz interceptor.
type AuthzInterceptorOptions struct {
	// Authorizer is the authorizer to use.
	Authorizer authz.Authorizer

	// SkipMethods is a list of full method names to skip authorization.
	SkipMethods []string

	// ResourceExtractor extracts the resource from the request.
	// Default: extracts service name from method.
	ResourceExtractor func(fullMethod string, req interface{}) string

	// ActionExtractor extracts the action from the request.
	// Default: extracts method name from method.
	ActionExtractor func(fullMethod string, req interface{}) string
}

// AuthzInterceptorOption is a functional option for authz interceptor.
type AuthzInterceptorOption func(*AuthzInterceptorOptions)

// NewAuthzInterceptorOptions creates default authz interceptor options.
func NewAuthzInterceptorOptions() *AuthzInterceptorOptions {
	return &AuthzInterceptorOptions{
		SkipMethods:       []string{},
		ResourceExtractor: defaultGRPCResourceExtractor,
		ActionExtractor:   defaultGRPCActionExtractor,
	}
}

// WithAuthorizer sets the authorizer.
func WithAuthorizer(a authz.Authorizer) AuthzInterceptorOption {
	return func(o *AuthzInterceptorOptions) {
		o.Authorizer = a
	}
}

// WithAuthzSkipMethods sets methods to skip authorization.
func WithAuthzSkipMethods(methods ...string) AuthzInterceptorOption {
	return func(o *AuthzInterceptorOptions) {
		o.SkipMethods = methods
	}
}

// WithResourceExtractor sets the resource extractor.
func WithResourceExtractor(extractor func(fullMethod string, req interface{}) string) AuthzInterceptorOption {
	return func(o *AuthzInterceptorOptions) {
		o.ResourceExtractor = extractor
	}
}

// WithActionExtractor sets the action extractor.
func WithActionExtractor(extractor func(fullMethod string, req interface{}) string) AuthzInterceptorOption {
	return func(o *AuthzInterceptorOptions) {
		o.ActionExtractor = extractor
	}
}

// UnaryAuthzInterceptor creates a unary authorization interceptor.
func UnaryAuthzInterceptor(opts ...AuthzInterceptorOption) grpc.UnaryServerInterceptor {
	options := NewAuthzInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	skipMap := make(map[string]struct{})
	for _, method := range options.SkipMethods {
		skipMap[method] = struct{}{}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method should be skipped
		if _, skip := skipMap[info.FullMethod]; skip {
			return handler(ctx, req)
		}

		// Authorize
		if err := authorize(ctx, info.FullMethod, req, options); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// StreamAuthzInterceptor creates a stream authorization interceptor.
func StreamAuthzInterceptor(opts ...AuthzInterceptorOption) grpc.StreamServerInterceptor {
	options := NewAuthzInterceptorOptions()
	for _, opt := range opts {
		opt(options)
	}

	skipMap := make(map[string]struct{})
	for _, method := range options.SkipMethods {
		skipMap[method] = struct{}{}
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method should be skipped
		if _, skip := skipMap[info.FullMethod]; skip {
			return handler(srv, ss)
		}

		// Authorize (use nil req for stream)
		if err := authorize(ss.Context(), info.FullMethod, nil, options); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}

// authorize performs authorization check.
func authorize(ctx context.Context, fullMethod string, req interface{}, opts *AuthzInterceptorOptions) error {
	if opts.Authorizer == nil {
		return status.Error(codes.Internal, "authorizer not configured")
	}

	// Get subject from context
	subject := auth.SubjectFromContext(ctx)
	if subject == "" {
		return status.Error(codes.Unauthenticated, "no subject found in context")
	}

	// Extract resource and action
	resource := opts.ResourceExtractor(fullMethod, req)
	action := opts.ActionExtractor(fullMethod, req)

	// Check authorization
	allowed, err := opts.Authorizer.Authorize(ctx, subject, resource, action)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if !allowed {
		return status.Errorf(codes.PermissionDenied,
			"access denied: subject=%s, resource=%s, action=%s",
			subject, resource, action)
	}

	return nil
}

// defaultGRPCResourceExtractor extracts the service name from the full method.
func defaultGRPCResourceExtractor(fullMethod string, _ interface{}) string {
	// Full method format: "/package.Service/Method"
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 2 {
		serviceParts := strings.Split(parts[1], ".")
		if len(serviceParts) > 0 {
			return serviceParts[len(serviceParts)-1]
		}
		return parts[1]
	}
	return fullMethod
}

// defaultGRPCActionExtractor extracts the method name from the full method.
func defaultGRPCActionExtractor(fullMethod string, _ interface{}) string {
	// Full method format: "/package.Service/Method"
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return fullMethod
}

// wrappedStream wraps a grpc.ServerStream with a custom context.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context.
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// ChainUnaryInterceptors chains multiple unary interceptors into one.
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return current(currentCtx, currentReq, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(ctx, req)
	}
}

// ChainStreamInterceptors chains multiple stream interceptors into one.
func ChainStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		buildChain := func(current grpc.StreamServerInterceptor, next grpc.StreamHandler) grpc.StreamHandler {
			return func(currentSrv interface{}, currentStream grpc.ServerStream) error {
				return current(currentSrv, currentStream, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(srv, ss)
	}
}
