package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GrpcMiddleware returns a gRPC unary interceptor for authorization
// subFn: function to extract subject from context
// objFn: function to extract object from full method name
// actFn: function to extract action from full method name (or constant)
func (a *Authorizer) GrpcUnaryInterceptor(
	subFn func(context.Context) string,
	objFn func(fullMethod string) string,
	actFn func(fullMethod string) string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		sub := subFn(ctx)
		obj := objFn(info.FullMethod)
		act := actFn(info.FullMethod)

		allowed, err := a.service.Enforce(sub, obj, act)
		if err != nil {
			return nil, status.Error(codes.Internal, "authorization error")
		}

		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		}

		return handler(ctx, req)
	}
}

// GrpcStreamInterceptor returns a gRPC stream interceptor for authorization
func (a *Authorizer) GrpcStreamInterceptor(
	subFn func(context.Context) string,
	objFn func(fullMethod string) string,
	actFn func(fullMethod string) string,
) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()
		sub := subFn(ctx)
		obj := objFn(info.FullMethod)
		act := actFn(info.FullMethod)

		allowed, err := a.service.Enforce(sub, obj, act)
		if err != nil {
			return status.Error(codes.Internal, "authorization error")
		}

		if !allowed {
			return status.Error(codes.PermissionDenied, "permission denied")
		}

		return handler(srv, ss)
	}
}
