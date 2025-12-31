// Package app provides the RAG Service application.
package app

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/enhancer"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/evaluator"
	"github.com/kart-io/sentinel-x/internal/rag/biz"
	grpcHandler "github.com/kart-io/sentinel-x/internal/rag/grpc"
	"github.com/kart-io/sentinel-x/internal/rag/handler"
	"github.com/kart-io/sentinel-x/internal/rag/router"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/component/milvus"
	"github.com/kart-io/sentinel-x/pkg/llm"

	// Register adapters
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"

	// Register LLM providers
	_ "github.com/kart-io/sentinel-x/pkg/llm/deepseek"
	_ "github.com/kart-io/sentinel-x/pkg/llm/gemini"
	_ "github.com/kart-io/sentinel-x/pkg/llm/huggingface"
	_ "github.com/kart-io/sentinel-x/pkg/llm/ollama"
	_ "github.com/kart-io/sentinel-x/pkg/llm/openai"

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
  - RAG-based question answering with LLM
  - Support for multiple LLM providers (Ollama, OpenAI, DeepSeek, HuggingFace, Gemini)`
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

	// 3. 初始化 Store 层
	vectorStore := store.NewMilvusStore(milvusClient)
	logger.Info("Vector store initialized")

	// 4. 初始化 LLM 供应商
	embedProvider, err := llm.NewEmbeddingProvider(opts.Embedding.Provider, opts.Embedding.ToConfigMap())
	if err != nil {
		return fmt.Errorf("failed to initialize embedding provider: %w", err)
	}
	logger.Infow("Embedding provider initialized",
		"provider", opts.Embedding.Provider,
		"model", opts.Embedding.Model,
	)

	chatProvider, err := llm.NewChatProvider(opts.Chat.Provider, opts.Chat.ToConfigMap())
	if err != nil {
		return fmt.Errorf("failed to initialize chat provider: %w", err)
	}
	logger.Infow("Chat provider initialized",
		"provider", opts.Chat.Provider,
		"model", opts.Chat.Model,
	)

	// 5. 初始化 Biz 层
	serviceConfig := &biz.ServiceConfig{
		IndexerConfig: &biz.IndexerConfig{
			ChunkSize:    opts.RAG.ChunkSize,
			ChunkOverlap: opts.RAG.ChunkOverlap,
			Collection:   opts.RAG.Collection,
			EmbeddingDim: opts.RAG.EmbeddingDim,
			DataDir:      opts.RAG.DataDir,
		},
		RetrieverConfig: &biz.RetrieverConfig{
			TopK:       opts.RAG.TopK,
			Collection: opts.RAG.Collection,
			Enhancer: enhancer.Config{
				EnableQueryRewrite: opts.RAG.Enhancer.EnableQueryRewrite,
				EnableHyDE:         opts.RAG.Enhancer.EnableHyDE,
				EnableRerank:       opts.RAG.Enhancer.EnableRerank,
				EnableRepacking:    opts.RAG.Enhancer.EnableRepacking,
				RerankTopK:         opts.RAG.Enhancer.RerankTopK,
			},
		},
		GeneratorConfig: &biz.GeneratorConfig{
			SystemPrompt: opts.RAG.SystemPrompt,
		},
	}
	ragService := biz.NewRAGService(vectorStore, embedProvider, chatProvider, serviceConfig)
	logger.Infow("RAG service initialized",
		"enhancer.query_rewrite", opts.RAG.Enhancer.EnableQueryRewrite,
		"enhancer.hyde", opts.RAG.Enhancer.EnableHyDE,
		"enhancer.rerank", opts.RAG.Enhancer.EnableRerank,
		"enhancer.repacking", opts.RAG.Enhancer.EnableRepacking,
	)

	// 6. 初始化评估器
	ragEvaluator := evaluator.New(chatProvider, embedProvider)
	logger.Info("RAG evaluator initialized")

	// 7. 初始化 Handler 层
	ragHandler := handler.NewRAGHandler(ragService, ragEvaluator)
	ragGRPCHandler := grpcHandler.NewHandler(ragService)
	logger.Info("Handler layer initialized")

	// 8. 初始化服务器
	serverManager := server.NewManager(
		server.WithMode(opts.Server.Mode),
		server.WithHTTPOptions(opts.Server.HTTP),
		server.WithGRPCOptions(opts.Server.GRPC),
		server.WithShutdownTimeout(opts.Server.ShutdownTimeout),
	)

	// 9. 注册路由
	if err := router.Register(serverManager, ragHandler, ragGRPCHandler); err != nil {
		return fmt.Errorf("failed to register routes: %w", err)
	}

	// 10. 启动服务器
	logger.Info("RAG service is ready")
	return serverManager.Run()
}

func printBanner(opts *Options) {
	fmt.Printf("Starting %s...\n", appName)
	fmt.Printf("  Embedding: %s (%s)\n", opts.Embedding.Provider, opts.Embedding.Model)
	fmt.Printf("  Chat: %s (%s)\n", opts.Chat.Provider, opts.Chat.Model)
}
