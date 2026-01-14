// Package ragsvc provides the RAG Service server implementation.
package ragsvc

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/enhancer"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/evaluator"
	"github.com/kart-io/sentinel-x/internal/rag/biz"
	grpcHandler "github.com/kart-io/sentinel-x/internal/rag/grpc"
	"github.com/kart-io/sentinel-x/internal/rag/handler"
	"github.com/kart-io/sentinel-x/internal/rag/router"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/component/milvus"
	// 导入适配器以注册 HTTP 框架支持
	"github.com/kart-io/sentinel-x/pkg/infra/app"
	"github.com/kart-io/sentinel-x/pkg/infra/server"
	"github.com/kart-io/sentinel-x/pkg/llm"
	// 导入 LLM 供应商以自动注册
	_ "github.com/kart-io/sentinel-x/pkg/llm/deepseek"
	_ "github.com/kart-io/sentinel-x/pkg/llm/gemini"
	_ "github.com/kart-io/sentinel-x/pkg/llm/huggingface"
	_ "github.com/kart-io/sentinel-x/pkg/llm/ollama"
	_ "github.com/kart-io/sentinel-x/pkg/llm/openai"
	cacheopts "github.com/kart-io/sentinel-x/pkg/options/cache"
	llmopts "github.com/kart-io/sentinel-x/pkg/options/llm"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	milvusopts "github.com/kart-io/sentinel-x/pkg/options/milvus"
	ragopts "github.com/kart-io/sentinel-x/pkg/options/rag"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
	// Register LLM providers
	goredis "github.com/redis/go-redis/v9"
)

// Name is the name of the application.
const Name = "sentinel-rag"

// Config contains application-related configurations.
type Config struct {
	HTTPOptions      *httpopts.Options
	GRPCOptions      *grpcopts.Options
	LogOptions       *logopts.Options
	MilvusOptions    *milvusopts.Options
	EmbeddingOptions *llmopts.ProviderOptions
	ChatOptions      *llmopts.ProviderOptions
	RAGOptions       *ragopts.Options
	CacheOptions     *cacheopts.Options
	RecoveryOptions  *middlewareopts.RecoveryOptions
	RequestIDOptions *middlewareopts.RequestIDOptions
	LoggerOptions    *middlewareopts.LoggerOptions
	CORSOptions      *middlewareopts.CORSOptions
	TimeoutOptions   *middlewareopts.TimeoutOptions
	HealthOptions    *middlewareopts.HealthOptions
	MetricsOptions   *middlewareopts.MetricsOptions
	PprofOptions     *middlewareopts.PprofOptions
	VersionOptions   *middlewareopts.VersionOptions
	ShutdownTimeout  time.Duration
}

// Server represents the RAG server.
type Server struct {
	srv         *server.Manager
	milvusClose func()
	redisClose  func()
}

// NewServer initializes and returns a new Server instance.
func (cfg *Config) NewServer(_ context.Context) (*Server, error) {
	printBanner(cfg)

	// 1. 初始化日志
	cfg.LogOptions.AddInitialField("service.name", Name)
	cfg.LogOptions.AddInitialField("service.version", app.GetVersion())
	if err := cfg.LogOptions.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger.Info("Starting RAG service...")

	// 2. 初始化 Milvus 客户端
	milvusClient, err := milvus.New(cfg.MilvusOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize milvus: %w", err)
	}
	logger.Info("Milvus client initialized")

	// 3. 初始化 Store 层
	vectorStore := store.NewMilvusStore(milvusClient)
	logger.Info("Vector store initialized")

	// 4. 初始化 Redis 客户端（用于缓存）
	var redisClient *goredis.Client
	var queryCache *biz.QueryCache
	var redisClose func()
	if cfg.CacheOptions.Enabled {
		redisOpts := cfg.CacheOptions.Redis
		if redisOpts == nil {
			logger.Warn("Cache is enabled but no Redis configuration provided in CacheOptions")
		} else {

			redisClient = goredis.NewClient(&goredis.Options{
				Addr:         fmt.Sprintf("%s:%d", redisOpts.Host, redisOpts.Port),
				Password:     redisOpts.Password,
				DB:           redisOpts.Database,
				MaxRetries:   redisOpts.MaxRetries,
				PoolSize:     redisOpts.PoolSize,
				MinIdleConns: redisOpts.MinIdleConns,
			})

			// 测试 Redis 连接
			if err := redisClient.Ping(context.Background()).Err(); err != nil {
				logger.Warnw("failed to connect to redis, cache will be disabled", "error", err.Error())
				_ = redisClient.Close()
				redisClient = nil
			} else {
				queryCache = biz.NewQueryCache(redisClient, &biz.QueryCacheConfig{
					Enabled:   true,
					TTL:       cfg.CacheOptions.TTL,
					KeyPrefix: cfg.CacheOptions.KeyPrefix,
				})
				redisClose = func() { _ = redisClient.Close() }
				logger.Infow("Redis cache initialized",
					"host", redisOpts.Host,
					"port", redisOpts.Port,
					"ttl", cfg.CacheOptions.TTL,
				)
			}
		}
	} else {
		logger.Info("Cache is disabled")
	}

	// 5. 初始化 LLM 供应商
	embedProvider, err := llm.NewEmbeddingProvider(cfg.EmbeddingOptions.Provider, cfg.EmbeddingOptions.ToConfigMap())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedding provider: %w", err)
	}
	logger.Infow("Embedding provider initialized",
		"provider", cfg.EmbeddingOptions.Provider,
		"model", cfg.EmbeddingOptions.Model,
	)

	chatProvider, err := llm.NewChatProvider(cfg.ChatOptions.Provider, cfg.ChatOptions.ToConfigMap())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chat provider: %w", err)
	}
	logger.Infow("Chat provider initialized",
		"provider", cfg.ChatOptions.Provider,
		"model", cfg.ChatOptions.Model,
	)

	// 6. 初始化 Biz 层
	serviceConfig := &biz.ServiceConfig{
		IndexerConfig: &biz.IndexerConfig{
			ChunkSize:    cfg.RAGOptions.ChunkSize,
			ChunkOverlap: cfg.RAGOptions.ChunkOverlap,
			Collection:   cfg.RAGOptions.Collection,
			EmbeddingDim: cfg.RAGOptions.EmbeddingDim,
			DataDir:      cfg.RAGOptions.DataDir,
		},
		RetrieverConfig: &biz.RetrieverConfig{
			TopK:       cfg.RAGOptions.TopK,
			Collection: cfg.RAGOptions.Collection,
			Enhancer: enhancer.Config{
				EnableQueryRewrite: cfg.RAGOptions.Enhancer.EnableQueryRewrite,
				EnableHyDE:         cfg.RAGOptions.Enhancer.EnableHyDE,
				EnableRerank:       cfg.RAGOptions.Enhancer.EnableRerank,
				EnableRepacking:    cfg.RAGOptions.Enhancer.EnableRepacking,
				RerankTopK:         cfg.RAGOptions.Enhancer.RerankTopK,
			},
		},
		GeneratorConfig: &biz.GeneratorConfig{
			SystemPrompt: cfg.RAGOptions.SystemPrompt,
		},
		QueryCacheConfig: &biz.QueryCacheConfig{
			Enabled:   cfg.CacheOptions.Enabled,
			TTL:       cfg.CacheOptions.TTL,
			KeyPrefix: cfg.CacheOptions.KeyPrefix,
		},
	}
	ragService := biz.NewRAGService(vectorStore, embedProvider, chatProvider, queryCache, serviceConfig)
	logger.Infow("RAG service initialized",
		"cache.enabled", cfg.CacheOptions.Enabled,
		"enhancer.query_rewrite", cfg.RAGOptions.Enhancer.EnableQueryRewrite,
		"enhancer.hyde", cfg.RAGOptions.Enhancer.EnableHyDE,
		"enhancer.rerank", cfg.RAGOptions.Enhancer.EnableRerank,
		"enhancer.repacking", cfg.RAGOptions.Enhancer.EnableRepacking,
	)

	// 7. 初始化评估器
	ragEvaluator := evaluator.New(chatProvider, embedProvider)
	logger.Info("RAG evaluator initialized")

	// 8. 初始化 Handler 层
	ragHandler := handler.NewRAGHandler(ragService, ragEvaluator)
	ragGRPCHandler := grpcHandler.NewHandler(ragService)
	logger.Info("Handler layer initialized")

	// 9. 初始化服务器
	serverManager := server.NewManager(
		server.WithHTTPOptions(cfg.HTTPOptions),
		server.WithGRPCOptions(cfg.GRPCOptions),
		server.WithMiddleware(cfg.GetMiddlewareOptions()),
		server.WithShutdownTimeout(cfg.ShutdownTimeout),
	)

	// 10. 注册路由
	if err := router.Register(serverManager, ragHandler, ragGRPCHandler); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	logger.Info("RAG service is ready")
	return &Server{
		srv:         serverManager,
		milvusClose: func() { _ = milvusClient.Close(context.Background()) },
		redisClose:  redisClose,
	}, nil
}

// Run starts the server and listens for termination signals.
func (s *Server) Run(_ context.Context) error {
	defer func() {
		if s.milvusClose != nil {
			s.milvusClose()
		}
		if s.redisClose != nil {
			s.redisClose()
		}
	}()
	return s.srv.Run()
}

// GetMiddlewareOptions 从各个配置构建中间件选项。
func (cfg *Config) GetMiddlewareOptions() *middlewareopts.Options {
	opts := middlewareopts.NewOptions()

	if cfg.RecoveryOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareRecovery, cfg.RecoveryOptions)
	}
	if cfg.RequestIDOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareRequestID, cfg.RequestIDOptions)
	}
	if cfg.LoggerOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareLogger, cfg.LoggerOptions)
	}
	if cfg.CORSOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareCORS, cfg.CORSOptions)
	}
	if cfg.TimeoutOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareTimeout, cfg.TimeoutOptions)
	}
	if cfg.HealthOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareHealth, cfg.HealthOptions)
	}
	if cfg.MetricsOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareMetrics, cfg.MetricsOptions)
	}
	if cfg.PprofOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewarePprof, cfg.PprofOptions)
	}
	if cfg.VersionOptions != nil {
		opts.SetConfig(middlewareopts.MiddlewareVersion, cfg.VersionOptions)
	}

	return opts
}

func printBanner(cfg *Config) {
	fmt.Printf("Starting %s...\n", Name)
	fmt.Printf("  Embedding: %s (%s)\n", cfg.EmbeddingOptions.Provider, cfg.EmbeddingOptions.Model)
	fmt.Printf("  Chat: %s (%s)\n", cfg.ChatOptions.Provider, cfg.ChatOptions.Model)

	// 打印中间件配置
	mw := cfg.GetMiddlewareOptions()
	if mw != nil {
		fmt.Printf("  Enabled Middlewares: %v\n", mw.GetEnabledMiddlewares())
	}
}
