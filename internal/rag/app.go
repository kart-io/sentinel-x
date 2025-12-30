// Package app provides the RAG Service application.
package app

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/rag/biz"
	grpcHandler "github.com/kart-io/sentinel-x/internal/rag/grpc"
	"github.com/kart-io/sentinel-x/internal/rag/handler"
	"github.com/kart-io/sentinel-x/internal/rag/router"
	"github.com/kart-io/sentinel-x/pkg/component/milvus"
	"github.com/kart-io/sentinel-x/pkg/component/ollama"

	// Register adapters
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
)

const (
	appName        = "sentinel-rag"
	appDescription = `Sentinel-X RAG Service

The RAG (Retrieval-Augmented Generation) knowledge base service for Sentinel-X platform.

This server provides:
  - Document indexing with vector embeddings
  - Semantic similarity search
  - RAG-based question answering with LLM`
)

// NewApp creates a new application instance.
func NewApp() *app.App {
	opts := NewOptions()

	return app.NewApp(
		app.WithName(appName),
		app.WithDescription(appDescription),
		app.WithOptions(opts),
		app.WithRunFunc(func() error {
			return Run(opts)
		}),
	)
}

// Run runs the RAG Service with the given options.
func Run(opts *Options) error {
	printBanner(opts)

	// 1. 初始化日志
	opts.Log.AddInitialField("service.name", appName)
	opts.Log.AddInitialField("service.version", app.GetVersion())
	if err := opts.Log.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting RAG service...")

	// 2. 初始化 Milvus 客户端
	milvusClient, err := milvus.New(opts.Milvus)
	if err != nil {
		return fmt.Errorf("failed to initialize milvus: %w", err)
	}
	defer milvusClient.Close(context.Background())
	logger.Info("Milvus client initialized")

	// 3. 初始化 Ollama 客户端
	ollamaClient := ollama.New(opts.Ollama)
	logger.Info("Ollama client initialized")

	// 4. 初始化 Biz 层
	ragConfig := &biz.RAGConfig{
		ChunkSize:    opts.RAG.ChunkSize,
		ChunkOverlap: opts.RAG.ChunkOverlap,
		TopK:         opts.RAG.TopK,
		Collection:   opts.RAG.Collection,
		EmbeddingDim: opts.RAG.EmbeddingDim,
		DataDir:      opts.RAG.DataDir,
		SystemPrompt: opts.RAG.SystemPrompt,
	}
	ragService := biz.NewRAGService(milvusClient, ollamaClient, ragConfig)
	logger.Info("RAG service initialized")

	// 5. 初始化 Handler 层
	ragHandler := handler.NewRAGHandler(ragService)
	ragGRPCHandler := grpcHandler.NewHandler(ragService)
	logger.Info("Handler layer initialized")

	// 6. 初始化服务器
	serverManager := server.NewManager(
		server.WithMode(opts.Server.Mode),
		server.WithHTTPOptions(opts.Server.HTTP),
		server.WithGRPCOptions(opts.Server.GRPC),
		server.WithShutdownTimeout(opts.Server.ShutdownTimeout),
	)

	// 7. 注册路由
	if err := router.Register(serverManager, ragHandler, ragGRPCHandler); err != nil {
		return fmt.Errorf("failed to register routes: %w", err)
	}

	// 8. 启动服务器
	logger.Info("RAG service is ready")
	return serverManager.Run()
}

func printBanner(_ *Options) {
	fmt.Printf("Starting %s...\n", appName)
}
