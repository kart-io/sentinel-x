// Package rag provides RAG (Retrieval-Augmented Generation) configuration options.
package rag

import (
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*Options)(nil)

// Options contains RAG-specific configuration.
type Options struct {
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

	// Tree 树形索引配置（Tree/RAPTOR）。
	Tree *TreeOptions `json:"tree" mapstructure:"tree"`
}

// TreeOptions 树形索引配置。
type TreeOptions struct {
	// Enabled 是否启用树形索引。
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// MaxLevel 树的最大层级（0=叶子层，建议 2-3 层）。
	MaxLevel int `json:"max-level" mapstructure:"max-level"`

	// NumClusters 每层的聚类数量（KMeans 聚类参数）。
	NumClusters int `json:"num-clusters" mapstructure:"num-clusters"`

	// SummaryModel 摘要生成使用的模型名称。
	SummaryModel string `json:"summary-model" mapstructure:"summary-model"`

	// SummaryMaxTokens 摘要的最大 token 长度。
	SummaryMaxTokens int `json:"summary-max-tokens" mapstructure:"summary-max-tokens"`
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

// DefaultSystemPrompt is the default system prompt for RAG queries.
const DefaultSystemPrompt = `You are a helpful assistant that answers questions based on the provided context.
Use the following context to answer the question. If you cannot find the answer in the context, say so.
Always cite the source documents when providing information.

Context:
{{context}}

Question: {{question}}

Answer:`

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

// NewTreeOptions 创建默认树配置。
func NewTreeOptions() *TreeOptions {
	return &TreeOptions{
		Enabled:          false, // 默认关闭，POC 功能
		MaxLevel:         3,
		NumClusters:      5,
		SummaryModel:     "deepseek-chat",
		SummaryMaxTokens: 200,
	}
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		ChunkSize:    512,
		ChunkOverlap: 50,
		TopK:         5,
		Collection:   "milvus_docs",
		EmbeddingDim: 768, // nomic-embed-text dimension
		DataDir:      "_output/rag-data",
		SystemPrompt: DefaultSystemPrompt,
		Enhancer:     NewEnhancerOptions(),
		Tree:         NewTreeOptions(),
	}
}

// AddFlags adds flags for RAG options to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.IntVar(&o.ChunkSize, options.Join(prefixes...)+"rag.chunk-size", o.ChunkSize, "Size of text chunks.")
	fs.IntVar(&o.ChunkOverlap, options.Join(prefixes...)+"rag.chunk-overlap", o.ChunkOverlap, "Overlap between chunks.")
	fs.IntVar(&o.TopK, options.Join(prefixes...)+"rag.top-k", o.TopK, "Number of results from similarity search.")
	fs.StringVar(&o.Collection, options.Join(prefixes...)+"rag.collection", o.Collection, "Milvus collection name.")
	fs.IntVar(&o.EmbeddingDim, options.Join(prefixes...)+"rag.embedding-dim", o.EmbeddingDim, "Embedding vector dimension.")
	fs.StringVar(&o.DataDir, options.Join(prefixes...)+"rag.data-dir", o.DataDir, "Directory for storing documents.")

	// 增强器配置
	if o.Enhancer == nil {
		o.Enhancer = NewEnhancerOptions()
	}
	fs.BoolVar(&o.Enhancer.EnableQueryRewrite, options.Join(prefixes...)+"rag.enhancer.enable-query-rewrite", o.Enhancer.EnableQueryRewrite, "Enable query rewriting.")
	fs.BoolVar(&o.Enhancer.EnableHyDE, options.Join(prefixes...)+"rag.enhancer.enable-hyde", o.Enhancer.EnableHyDE, "Enable HyDE (Hypothetical Document Embeddings).")
	fs.BoolVar(&o.Enhancer.EnableRerank, options.Join(prefixes...)+"rag.enhancer.enable-rerank", o.Enhancer.EnableRerank, "Enable result reranking.")
	fs.BoolVar(&o.Enhancer.EnableRepacking, options.Join(prefixes...)+"rag.enhancer.enable-repacking", o.Enhancer.EnableRepacking, "Enable document repacking.")
	fs.IntVar(&o.Enhancer.RerankTopK, options.Join(prefixes...)+"rag.enhancer.rerank-top-k", o.Enhancer.RerankTopK, "Number of documents to keep after reranking.")

	// 树形索引配置
	if o.Tree == nil {
		o.Tree = NewTreeOptions()
	}
	fs.BoolVar(&o.Tree.Enabled, options.Join(prefixes...)+"rag.tree.enabled", o.Tree.Enabled, "Enable tree-based indexing (RAPTOR).")
	fs.IntVar(&o.Tree.MaxLevel, options.Join(prefixes...)+"rag.tree.max-level", o.Tree.MaxLevel, "Maximum tree level (0=leaf).")
	fs.IntVar(&o.Tree.NumClusters, options.Join(prefixes...)+"rag.tree.num-clusters", o.Tree.NumClusters, "Number of clusters per level.")
	fs.StringVar(&o.Tree.SummaryModel, options.Join(prefixes...)+"rag.tree.summary-model", o.Tree.SummaryModel, "Model for summary generation.")
	fs.IntVar(&o.Tree.SummaryMaxTokens, options.Join(prefixes...)+"rag.tree.summary-max-tokens", o.Tree.SummaryMaxTokens, "Max tokens for summaries.")
}

// Validate validates the RAG options.
func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if o.ChunkSize <= 0 {
		errs = append(errs, fmt.Errorf("chunk-size must be positive"))
	}
	if o.TopK <= 0 {
		errs = append(errs, fmt.Errorf("top-k must be positive"))
	}
	if o.EmbeddingDim <= 0 {
		errs = append(errs, fmt.Errorf("embedding-dim must be positive"))
	}
	return errs
}

// Complete completes the RAG options with defaults.
func (o *Options) Complete() error {
	if o.Enhancer == nil {
		o.Enhancer = NewEnhancerOptions()
	}
	if o.Tree == nil {
		o.Tree = NewTreeOptions()
	}
	if o.SystemPrompt == "" {
		o.SystemPrompt = DefaultSystemPrompt
	}
	return nil
}
