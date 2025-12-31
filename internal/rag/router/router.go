// Package router provides RAG service routing.
package router

import (
	"github.com/kart-io/logger"
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
		router := httpServer.Router()

		// RAG API Routes
		v1 := router.Group("/v1")
		{
			rag := v1.Group("/rag")
			{
				// Index endpoints
				rag.Handle("POST", "/index", ragHandler.Index)
				rag.Handle("POST", "/index/directory", ragHandler.IndexDirectory)
				rag.Handle("POST", "/index/url", ragHandler.IndexFromURL)

				// Query endpoint
				rag.Handle("POST", "/query", ragHandler.Query)

				// Stats endpoint
				rag.Handle("GET", "/stats", ragHandler.Stats)

				// Evaluation endpoints (Ragas metrics)
				rag.Handle("POST", "/evaluate", ragHandler.Evaluate)
				rag.Handle("POST", "/query-evaluate", ragHandler.QueryAndEvaluate)
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
