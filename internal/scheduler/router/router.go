package router

import (
	"github.com/kart-io/sentinel-x/internal/scheduler/handler"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	pb "github.com/kart-io/sentinel-x/pkg/api/scheduler/v1"
)

// Register registers the scheduler service routes.
func Register(mgr *server.Manager, h *handler.SchedulerHandler) error {
	// gRPC Server
	if grpcServer := mgr.GRPCServer(); grpcServer != nil {
		pb.RegisterSchedulerServiceServer(grpcServer.Server(), h)
	}

	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		// engine := httpServer.Engine()
		// v1 := engine.Group("/v1")
		// ... register HTTP/REST routes here
	}

	return nil
}
