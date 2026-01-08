// Package router provides RAG service routing.
package router

import (
	"github.com/kart-io/logger"
	// Import swagger docs for API documentation
	_ "github.com/kart-io/sentinel-x/api/swagger/rag"
	grpcHandler "github.com/kart-io/sentinel-x/internal/rag/grpc"
	"github.com/kart-io/sentinel-x/internal/rag/handler"
	ragv1 "github.com/kart-io/sentinel-x/pkg/api/rag/v1"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
)

// Register registers the RAG service routes.
func Register(mgr *server.Manager, ragHandler *handler.RAGHandler, ragGRPCHandler *grpcHandler.Handler) error {
	logger.Info("Registering RAG routes...")

	// HTTP Server
	if httpServer := mgr.HTTPServer(); httpServer != nil {
		engine := httpServer.Engine()

		// RAG API Routes
		v1 := engine.Group("/v1")
		{
			rag := v1.Group("/rag")
			{
				// Index endpoints
				rag.POST("/index", ragHandler.Index)
				rag.POST("/index/directory", ragHandler.IndexDirectory)
				rag.POST("/index/url", ragHandler.IndexFromURL)

				// Query endpoint
				rag.POST("/query", ragHandler.Query)

				// Stats endpoint
				rag.GET("/stats", ragHandler.Stats)

				// Collections endpoint
				rag.GET("/collections", ragHandler.ListCollections)

				// Evaluation endpoints (Ragas metrics)
				rag.POST("/evaluate", ragHandler.Evaluate)
				rag.POST("/query-evaluate", ragHandler.QueryAndEvaluate)

				// Prometheus metrics endpoint
				rag.GET("/metrics", ragHandler.Metrics)
			}
		}

		logger.Info("HTTP routes registered")
	}

	// gRPC Server
	if grpcServer := mgr.GRPCServer(); grpcServer != nil {
		ragv1.RegisterRAGServiceServer(grpcServer.Server(), ragGRPCHandler)
		logger.Info("gRPC routes registered")
	}

	return nil
}
