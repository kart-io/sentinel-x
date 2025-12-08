package handler

import (
	"context"

	"google.golang.org/grpc/status"

	v1 "github.com/kart-io/sentinel-x/example/server/example/api/hello/v1"
	"github.com/kart-io/sentinel-x/example/server/example/service/helloservice"
	"github.com/kart-io/sentinel-x/pkg/errors"
)

// HelloGRPCHandler handles gRPC requests for HelloService.
// It delegates all business logic to the underlying service.
// NO business logic should exist here.
type HelloGRPCHandler struct {
	v1.UnimplementedHelloServiceServer
	svc *helloservice.Service
}

// NewHelloGRPCHandler creates a new gRPC handler.
func NewHelloGRPCHandler(svc *helloservice.Service) *HelloGRPCHandler {
	return &HelloGRPCHandler{svc: svc}
}

// SayHello implements the gRPC SayHello method.
// It delegates to the service layer - NO business logic here.
func (h *HelloGRPCHandler) SayHello(ctx context.Context, req *v1.HelloRequest) (*v1.HelloResponse, error) {
	name := req.Name
	if name == "" {
		name = "World"
	}

	// Delegate to service layer - NO business logic here
	message, err := h.svc.SayHello(ctx, name)
	if err != nil {
		// Convert Errno to gRPC status error
		return nil, toGRPCError(err)
	}

	return &v1.HelloResponse{Message: message}, nil
}

// toGRPCError converts an error to a gRPC status error.
// If the error is an Errno, it uses the appropriate gRPC code.
// Otherwise, it returns an Internal error.
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Convert to Errno if possible
	errno := errors.FromError(err)
	return status.Error(errno.GRPCCode, errno.MessageEN)
}

// Ensure HelloGRPCHandler implements HelloServiceServer.
var _ v1.HelloServiceServer = (*HelloGRPCHandler)(nil)
