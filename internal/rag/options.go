// Package app provides the RAG Service application.
package app

import (
	"fmt"
	"time"

	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
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

	// Milvus contains Milvus database configuration.
	Milvus *milvusopts.Options `json:"milvus" mapstructure:"milvus"`

	// Embedding contains embedding provider configuration.
	Embedding *LLMProviderOptions `json:"embedding" mapstructure:"embedding"`

	// Chat contains chat provider configuration.
	Chat *LLMProviderOptions `json:"chat" mapstructure:"chat"`

	// RAG contains RAG-specific configuration.
	RAG *RAGOptions `json:"rag" mapstructure:"rag"`
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
		Milvus:    milvusopts.NewOptions(),
		Embedding: embeddingOpts,
		Chat:      chatOpts,
		RAG:       NewRAGOptions(),
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
