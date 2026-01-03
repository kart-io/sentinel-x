// Package app provides the RAG Service application.
package app

import (
	"fmt"
	"time"

	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	milvusopts "github.com/kart-io/sentinel-x/pkg/options/milvus"
	serveropts "github.com/kart-io/sentinel-x/pkg/options/server"
	"github.com/spf13/pflag"
)

// Options contains all RAG Service options.
type Options struct {
	// Server contains server configuration (HTTP/gRPC).
	Server *serveropts.Options `json:"server" mapstructure:"server"`

	// Log contains logger configuration.
	Log *logopts.Options `json:"log" mapstructure:"log"`

	// Metrics contains top-level metrics configuration.
	Metrics *middlewareopts.MetricsOptions `json:"metrics" mapstructure:"metrics"`

	// Health contains top-level health check configuration.
	Health *middlewareopts.HealthOptions `json:"health" mapstructure:"health"`

	// Pprof contains top-level pprof configuration.
	Pprof *middlewareopts.PprofOptions `json:"pprof" mapstructure:"pprof"`

	// Recovery contains top-level recovery configuration.
	Recovery *middlewareopts.RecoveryOptions `json:"recovery" mapstructure:"recovery"`

	// Logger contains top-level logger middleware configuration.
	Logger *middlewareopts.LoggerOptions `json:"logger" mapstructure:"logger"`

	// CORS contains top-level CORS middleware configuration.
	CORS *middlewareopts.CORSOptions `json:"cors" mapstructure:"cors"`

	// Timeout contains top-level timeout middleware configuration.
	Timeout *middlewareopts.TimeoutOptions `json:"timeout" mapstructure:"timeout"`

	// RequestID contains top-level request ID middleware configuration.
	RequestID *middlewareopts.RequestIDOptions `json:"request-id" mapstructure:"request-id"`

	// Milvus contains Milvus database configuration.
	Milvus *milvusopts.Options `json:"milvus" mapstructure:"milvus"`

	// Embedding contains embedding provider configuration.
	Embedding *LLMProviderOptions `json:"embedding" mapstructure:"embedding"`

	// Chat contains chat provider configuration.
	Chat *LLMProviderOptions `json:"chat" mapstructure:"chat"`

	// RAG contains RAG-specific configuration.
	RAG *RAGOptions `json:"rag" mapstructure:"rag"`

	// Cache contains cache configuration.
	Cache *CacheOptions `json:"cache" mapstructure:"cache"`
}

// LLMProviderOptions 定义 LLM 供应商配置。
type LLMProviderOptions struct {
	// Provider 供应商名称（ollama, openai 等）。
	Provider string `json:"provider" mapstructure:"provider"`

	// BaseURL API 基础地址。
	BaseURL string `json:"base-url" mapstructure:"base-url"`

	// APIKey API 密钥（OpenAI 等需要）。
	APIKey string `json:"api-key" mapstructure:"api-key"`

	// Model 使用的模型名称。
	Model string `json:"model" mapstructure:"model"`

	// Timeout 请求超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// MaxRetries 最大重试次数。
	MaxRetries int `json:"max-retries" mapstructure:"max-retries"`

	// Organization 组织 ID（OpenAI 可选）。
	Organization string `json:"organization" mapstructure:"organization"`
}

// NewLLMProviderOptions 创建默认 LLM 供应商配置。
func NewLLMProviderOptions() *LLMProviderOptions {
	return &LLMProviderOptions{
		Provider:   "ollama",
		BaseURL:    "http://localhost:11434",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// ToConfigMap 转换为配置 map，用于供应商工厂。
func (o *LLMProviderOptions) ToConfigMap() map[string]any {
	return map[string]any{
		"base_url":     o.BaseURL,
		"api_key":      o.APIKey,
		"embed_model":  o.Model,
		"chat_model":   o.Model,
		"timeout":      o.Timeout,
		"max_retries":  o.MaxRetries,
		"organization": o.Organization,
	}
}

// RAGOptions contains RAG-specific configuration.
type RAGOptions struct {
	// ChunkSize is the size of text chunks.
	ChunkSize int `json:"chunk-size" mapstructure:"chunk-size"`

	// ChunkOverlap is the overlap between chunks.
	ChunkOverlap int `json:"chunk-overlap" mapstructure:"chunk-overlap"`

	// TopK is the number of results to return from similarity search.
	TopK int `json:"top-k" mapstructure:"top-k"`

	// Collection is the name of the Milvus collection.
	Collection string `json:"collection" mapstructure:"collection"`

	// EmbeddingDim is the dimension of embedding vectors.
	EmbeddingDim int `json:"embedding-dim" mapstructure:"embedding-dim"`

	// DataDir is the directory for storing downloaded documents.
	DataDir string `json:"data-dir" mapstructure:"data-dir"`

	// SystemPrompt is the system prompt for RAG queries.
	SystemPrompt string `json:"system-prompt" mapstructure:"system-prompt"`

	// Enhancer 增强器配置。
	Enhancer *EnhancerOptions `json:"enhancer" mapstructure:"enhancer"`
}

// EnhancerOptions RAG 增强器配置。
type EnhancerOptions struct {
	// EnableQueryRewrite 是否启用查询重写。
	EnableQueryRewrite bool `json:"enable-query-rewrite" mapstructure:"enable-query-rewrite"`

	// EnableHyDE 是否启用 HyDE（假设文档嵌入）。
	EnableHyDE bool `json:"enable-hyde" mapstructure:"enable-hyde"`

	// EnableRerank 是否启用重排序。
	EnableRerank bool `json:"enable-rerank" mapstructure:"enable-rerank"`

	// EnableRepacking 是否启用文档重组。
	EnableRepacking bool `json:"enable-repacking" mapstructure:"enable-repacking"`

	// RerankTopK 重排序后保留的文档数量。
	RerankTopK int `json:"rerank-top-k" mapstructure:"rerank-top-k"`
}

// CacheOptions 查询缓存配置。
type CacheOptions struct {
	// Enabled 是否启用缓存。
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// TTL 缓存过期时间。
	TTL time.Duration `json:"ttl" mapstructure:"ttl"`

	// KeyPrefix 缓存键前缀。
	KeyPrefix string `json:"key-prefix" mapstructure:"key-prefix"`

	// Redis Redis 连接配置。
	Redis *RedisOptions `json:"redis" mapstructure:"redis"`
}

// RedisOptions Redis 配置。
type RedisOptions struct {
	// Host Redis 主机地址。
	Host string `json:"host" mapstructure:"host"`

	// Port Redis 端口。
	Port int `json:"port" mapstructure:"port"`

	// Password Redis 密码。
	Password string `json:"password" mapstructure:"password"`

	// Database Redis 数据库编号。
	Database int `json:"database" mapstructure:"database"`

	// MaxRetries 最大重试次数。
	MaxRetries int `json:"max-retries" mapstructure:"max-retries"`

	// PoolSize 连接池大小。
	PoolSize int `json:"pool-size" mapstructure:"pool-size"`

	// MinIdleConns 最小空闲连接数。
	MinIdleConns int `json:"min-idle-conns" mapstructure:"min-idle-conns"`
}

// NewCacheOptions 创建默认缓存配置。
func NewCacheOptions() *CacheOptions {
	return &CacheOptions{
		Enabled:   true,
		TTL:       1 * time.Hour,
		KeyPrefix: "rag:query:",
		Redis:     NewRedisOptions(),
	}
}

// NewRedisOptions 创建默认 Redis 配置。
func NewRedisOptions() *RedisOptions {
	return &RedisOptions{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		Database:     0,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 5,
	}
}

// NewEnhancerOptions 创建默认增强器配置。
func NewEnhancerOptions() *EnhancerOptions {
	return &EnhancerOptions{
		EnableQueryRewrite: true,
		EnableHyDE:         false, // HyDE 增加延迟，默认关闭
		EnableRerank:       true,
		EnableRepacking:    true,
		RerankTopK:         5,
	}
}

// NewRAGOptions creates new RAGOptions with defaults.
func NewRAGOptions() *RAGOptions {
	return &RAGOptions{
		ChunkSize:    512,
		ChunkOverlap: 50,
		TopK:         5,
		Collection:   "milvus_docs",
		EmbeddingDim: 768, // nomic-embed-text dimension
		DataDir:      "_output/rag-data",
		SystemPrompt: `You are a helpful assistant that answers questions based on the provided context.
Use the following context to answer the question. If you cannot find the answer in the context, say so.
Always cite the source documents when providing information.

Context:
{{context}}

Question: {{question}}

Answer:`,
		Enhancer: NewEnhancerOptions(),
	}
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	serverOpts := serveropts.NewOptions()
	serverOpts.HTTP.Addr = ":8082"
	serverOpts.GRPC.Addr = ":8102"

	// 默认 embedding 配置
	embeddingOpts := NewLLMProviderOptions()
	embeddingOpts.Model = "nomic-embed-text"

	// 默认 chat 配置
	chatOpts := NewLLMProviderOptions()
	chatOpts.Model = "deepseek-r1:7b"

	return &Options{
		Server:    serverOpts,
		Log:       logopts.NewOptions(),
		Metrics:   middlewareopts.NewMetricsOptions(),
		Health:    middlewareopts.NewHealthOptions(),
		Pprof:     middlewareopts.NewPprofOptions(),
		Recovery:  middlewareopts.NewRecoveryOptions(),
		Logger:    middlewareopts.NewLoggerOptions(),
		CORS:      middlewareopts.NewCORSOptions(),
		Timeout:   middlewareopts.NewTimeoutOptions(),
		RequestID: middlewareopts.NewRequestIDOptions(),
		Milvus:    milvusopts.NewOptions(),
		Embedding: embeddingOpts,
		Chat:      chatOpts,
		RAG:       NewRAGOptions(),
		Cache:     NewCacheOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
	o.Log.AddFlags(fs)
	o.Milvus.AddFlags(fs, "milvus.")
	o.addEmbeddingFlags(fs)
	o.addChatFlags(fs)
	o.addRAGFlags(fs)
	o.addCacheFlags(fs)
}

func (o *Options) addEmbeddingFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Embedding.Provider, "embedding.provider", o.Embedding.Provider, "Embedding provider (ollama, openai)")
	fs.StringVar(&o.Embedding.BaseURL, "embedding.base-url", o.Embedding.BaseURL, "Embedding API base URL")
	fs.StringVar(&o.Embedding.APIKey, "embedding.api-key", o.Embedding.APIKey, "Embedding API key (for OpenAI)")
	fs.StringVar(&o.Embedding.Model, "embedding.model", o.Embedding.Model, "Embedding model name")
	fs.DurationVar(&o.Embedding.Timeout, "embedding.timeout", o.Embedding.Timeout, "Embedding request timeout")
	fs.IntVar(&o.Embedding.MaxRetries, "embedding.max-retries", o.Embedding.MaxRetries, "Embedding max retries")
}

func (o *Options) addChatFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Chat.Provider, "chat.provider", o.Chat.Provider, "Chat provider (ollama, openai)")
	fs.StringVar(&o.Chat.BaseURL, "chat.base-url", o.Chat.BaseURL, "Chat API base URL")
	fs.StringVar(&o.Chat.APIKey, "chat.api-key", o.Chat.APIKey, "Chat API key (for OpenAI)")
	fs.StringVar(&o.Chat.Model, "chat.model", o.Chat.Model, "Chat model name")
	fs.DurationVar(&o.Chat.Timeout, "chat.timeout", o.Chat.Timeout, "Chat request timeout")
	fs.IntVar(&o.Chat.MaxRetries, "chat.max-retries", o.Chat.MaxRetries, "Chat max retries")
}

func (o *Options) addRAGFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.RAG.ChunkSize, "rag.chunk-size", o.RAG.ChunkSize, "Size of text chunks")
	fs.IntVar(&o.RAG.ChunkOverlap, "rag.chunk-overlap", o.RAG.ChunkOverlap, "Overlap between chunks")
	fs.IntVar(&o.RAG.TopK, "rag.top-k", o.RAG.TopK, "Number of results from similarity search")
	fs.StringVar(&o.RAG.Collection, "rag.collection", o.RAG.Collection, "Milvus collection name")
	fs.IntVar(&o.RAG.EmbeddingDim, "rag.embedding-dim", o.RAG.EmbeddingDim, "Embedding vector dimension")
	fs.StringVar(&o.RAG.DataDir, "rag.data-dir", o.RAG.DataDir, "Directory for storing documents")

	// 增强器配置
	fs.BoolVar(&o.RAG.Enhancer.EnableQueryRewrite, "rag.enhancer.enable-query-rewrite", o.RAG.Enhancer.EnableQueryRewrite, "Enable query rewriting")
	fs.BoolVar(&o.RAG.Enhancer.EnableHyDE, "rag.enhancer.enable-hyde", o.RAG.Enhancer.EnableHyDE, "Enable HyDE (Hypothetical Document Embeddings)")
	fs.BoolVar(&o.RAG.Enhancer.EnableRerank, "rag.enhancer.enable-rerank", o.RAG.Enhancer.EnableRerank, "Enable result reranking")
	fs.BoolVar(&o.RAG.Enhancer.EnableRepacking, "rag.enhancer.enable-repacking", o.RAG.Enhancer.EnableRepacking, "Enable document repacking")
	fs.IntVar(&o.RAG.Enhancer.RerankTopK, "rag.enhancer.rerank-top-k", o.RAG.Enhancer.RerankTopK, "Number of documents to keep after reranking")
}

func (o *Options) addCacheFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Cache.Enabled, "cache.enabled", o.Cache.Enabled, "Enable query result cache")
	fs.DurationVar(&o.Cache.TTL, "cache.ttl", o.Cache.TTL, "Cache TTL duration")
	fs.StringVar(&o.Cache.KeyPrefix, "cache.key-prefix", o.Cache.KeyPrefix, "Cache key prefix")
	fs.StringVar(&o.Cache.Redis.Host, "cache.redis.host", o.Cache.Redis.Host, "Redis host")
	fs.IntVar(&o.Cache.Redis.Port, "cache.redis.port", o.Cache.Redis.Port, "Redis port")
	fs.StringVar(&o.Cache.Redis.Password, "cache.redis.password", o.Cache.Redis.Password, "Redis password")
	fs.IntVar(&o.Cache.Redis.Database, "cache.redis.database", o.Cache.Redis.Database, "Redis database number")
	fs.IntVar(&o.Cache.Redis.MaxRetries, "cache.redis.max-retries", o.Cache.Redis.MaxRetries, "Redis max retries")
	fs.IntVar(&o.Cache.Redis.PoolSize, "cache.redis.pool-size", o.Cache.Redis.PoolSize, "Redis connection pool size")
	fs.IntVar(&o.Cache.Redis.MinIdleConns, "cache.redis.min-idle-conns", o.Cache.Redis.MinIdleConns, "Redis minimum idle connections")
}

// Validate validates the options.
func (o *Options) Validate() error {
	if err := o.Log.Validate(); err != nil {
		return err
	}
	if err := o.Server.Validate(); err != nil {
		return err
	}
	if err := o.Milvus.Validate(); err != nil {
		return err
	}
	if err := o.validateLLMProvider(o.Embedding, "embedding"); err != nil {
		return err
	}
	if err := o.validateLLMProvider(o.Chat, "chat"); err != nil {
		return err
	}
	if o.RAG.ChunkSize <= 0 {
		return fmt.Errorf("rag.chunk-size must be positive")
	}
	if o.RAG.TopK <= 0 {
		return fmt.Errorf("rag.top-k must be positive")
	}
	return nil
}

func (o *Options) validateLLMProvider(opts *LLMProviderOptions, prefix string) error {
	if opts.Provider == "" {
		return fmt.Errorf("%s.provider is required", prefix)
	}
	if opts.BaseURL == "" {
		return fmt.Errorf("%s.base-url is required", prefix)
	}
	if opts.Model == "" {
		return fmt.Errorf("%s.model is required", prefix)
	}
	// OpenAI 供应商需要 API key
	if opts.Provider == "openai" && opts.APIKey == "" {
		return fmt.Errorf("%s.api-key is required for openai provider", prefix)
	}
	if opts.Timeout <= 0 {
		return fmt.Errorf("%s.timeout must be positive", prefix)
	}
	return nil
}

// Complete completes the options.
func (o *Options) Complete() error {
	// 将顶层 Metrics 配置应用到 server.http.middleware.metrics
	if o.Metrics != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Metrics == nil {
			o.Server.HTTP.Middleware.Metrics = o.Metrics
		} else {
			// 合并配置，顶层配置优先
			if o.Metrics.Path != "" {
				o.Server.HTTP.Middleware.Metrics.Path = o.Metrics.Path
			}
			if o.Metrics.Namespace != "" {
				o.Server.HTTP.Middleware.Metrics.Namespace = o.Metrics.Namespace
			}
			if o.Metrics.Subsystem != "" {
				o.Server.HTTP.Middleware.Metrics.Subsystem = o.Metrics.Subsystem
			}
		}
	}

	// 将顶层 Health 配置应用到 server.http.middleware.health
	if o.Health != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Health == nil {
			o.Server.HTTP.Middleware.Health = o.Health
		} else {
			// 合并配置，顶层配置优先
			if o.Health.Path != "" {
				o.Server.HTTP.Middleware.Health.Path = o.Health.Path
			}
			if o.Health.LivenessPath != "" {
				o.Server.HTTP.Middleware.Health.LivenessPath = o.Health.LivenessPath
			}
			if o.Health.ReadinessPath != "" {
				o.Server.HTTP.Middleware.Health.ReadinessPath = o.Health.ReadinessPath
			}
		}
	}

	// 将顶层 Pprof 配置应用到 server.http.middleware.pprof
	if o.Pprof != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Pprof == nil {
			o.Server.HTTP.Middleware.Pprof = o.Pprof
		} else {
			// 合并配置，顶层配置优先
			if o.Pprof.Prefix != "" {
				o.Server.HTTP.Middleware.Pprof.Prefix = o.Pprof.Prefix
			}
			o.Server.HTTP.Middleware.Pprof.EnableCmdline = o.Pprof.EnableCmdline
			o.Server.HTTP.Middleware.Pprof.EnableProfile = o.Pprof.EnableProfile
			o.Server.HTTP.Middleware.Pprof.EnableSymbol = o.Pprof.EnableSymbol
			o.Server.HTTP.Middleware.Pprof.EnableTrace = o.Pprof.EnableTrace
			if o.Pprof.BlockProfileRate != 0 {
				o.Server.HTTP.Middleware.Pprof.BlockProfileRate = o.Pprof.BlockProfileRate
			}
			if o.Pprof.MutexProfileFraction != 0 {
				o.Server.HTTP.Middleware.Pprof.MutexProfileFraction = o.Pprof.MutexProfileFraction
			}
		}
	}

	// 将顶层 Recovery 配置应用到 server.http.middleware.recovery
	if o.Recovery != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Recovery == nil {
			o.Server.HTTP.Middleware.Recovery = o.Recovery
		} else {
			// 合并配置，顶层配置优先
			o.Server.HTTP.Middleware.Recovery.EnableStackTrace = o.Recovery.EnableStackTrace
			if o.Recovery.OnPanic != nil {
				o.Server.HTTP.Middleware.Recovery.OnPanic = o.Recovery.OnPanic
			}
		}
	}

	// 将顶层 Logger 配置应用到 server.http.middleware.logger
	if o.Logger != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Logger == nil {
			o.Server.HTTP.Middleware.Logger = o.Logger
		} else {
			// 合并配置，顶层配置优先
			if len(o.Logger.SkipPaths) > 0 {
				o.Server.HTTP.Middleware.Logger.SkipPaths = o.Logger.SkipPaths
			}
			o.Server.HTTP.Middleware.Logger.UseStructuredLogger = o.Logger.UseStructuredLogger
			if o.Logger.Output != nil {
				o.Server.HTTP.Middleware.Logger.Output = o.Logger.Output
			}
		}
	}

	// 将顶层 CORS 配置应用到 server.http.middleware.cors
	if o.CORS != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.CORS == nil {
			o.Server.HTTP.Middleware.CORS = o.CORS
		} else {
			// 合并配置，顶层配置优先
			if len(o.CORS.AllowOrigins) > 0 {
				o.Server.HTTP.Middleware.CORS.AllowOrigins = o.CORS.AllowOrigins
			}
			if len(o.CORS.AllowMethods) > 0 {
				o.Server.HTTP.Middleware.CORS.AllowMethods = o.CORS.AllowMethods
			}
			if len(o.CORS.AllowHeaders) > 0 {
				o.Server.HTTP.Middleware.CORS.AllowHeaders = o.CORS.AllowHeaders
			}
			if len(o.CORS.ExposeHeaders) > 0 {
				o.Server.HTTP.Middleware.CORS.ExposeHeaders = o.CORS.ExposeHeaders
			}
			o.Server.HTTP.Middleware.CORS.AllowCredentials = o.CORS.AllowCredentials
			if o.CORS.MaxAge != 0 {
				o.Server.HTTP.Middleware.CORS.MaxAge = o.CORS.MaxAge
			}
		}
	}

	// 将顶层 Timeout 配置应用到 server.http.middleware.timeout
	if o.Timeout != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.Timeout == nil {
			o.Server.HTTP.Middleware.Timeout = o.Timeout
		} else {
			// 合并配置，顶层配置优先
			if o.Timeout.Timeout > 0 {
				o.Server.HTTP.Middleware.Timeout.Timeout = o.Timeout.Timeout
			}
			if len(o.Timeout.SkipPaths) > 0 {
				o.Server.HTTP.Middleware.Timeout.SkipPaths = o.Timeout.SkipPaths
			}
		}
	}

	// 将顶层 RequestID 配置应用到 server.http.middleware.request-id
	if o.RequestID != nil && o.Server != nil && o.Server.HTTP != nil && o.Server.HTTP.Middleware != nil {
		if o.Server.HTTP.Middleware.RequestID == nil {
			o.Server.HTTP.Middleware.RequestID = o.RequestID
		} else {
			// 合并配置，顶层配置优先
			if o.RequestID.Header != "" {
				o.Server.HTTP.Middleware.RequestID.Header = o.RequestID.Header
			}
			if o.RequestID.Generator != nil {
				o.Server.HTTP.Middleware.RequestID.Generator = o.RequestID.Generator
			}
		}
	}

	if err := o.Server.Complete(); err != nil {
		return err
	}
	return nil
}

// GetTimeout returns a reasonable timeout for RAG operations.
func (o *Options) GetTimeout() time.Duration {
	// 使用较长的超时时间
	if o.Chat.Timeout > o.Embedding.Timeout {
		return o.Chat.Timeout
	}
	return o.Embedding.Timeout
}
