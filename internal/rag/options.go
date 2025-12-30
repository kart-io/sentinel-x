// Package app provides the RAG Service application.
package app

import (
	"fmt"
	"time"

	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	milvusopts "github.com/kart-io/sentinel-x/pkg/options/milvus"
	ollamaopts "github.com/kart-io/sentinel-x/pkg/options/ollama"
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

	// Ollama contains Ollama API configuration.
	Ollama *ollamaopts.Options `json:"ollama" mapstructure:"ollama"`

	// RAG contains RAG-specific configuration.
	RAG *RAGOptions `json:"rag" mapstructure:"rag"`
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
	}
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	serverOpts := serveropts.NewOptions()
	serverOpts.HTTP.Addr = ":8082"
	serverOpts.GRPC.Addr = ":8102"

	return &Options{
		Server: serverOpts,
		Log:    logopts.NewOptions(),
		Milvus: milvusopts.NewOptions(),
		Ollama: ollamaopts.NewOptions(),
		RAG:    NewRAGOptions(),
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.Server.AddFlags(fs)
	o.Log.AddFlags(fs)
	o.Milvus.AddFlags(fs, "milvus.")
	o.Ollama.AddFlags(fs, "ollama.")
	o.addRAGFlags(fs)
}

func (o *Options) addRAGFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.RAG.ChunkSize, "rag.chunk-size", o.RAG.ChunkSize, "Size of text chunks")
	fs.IntVar(&o.RAG.ChunkOverlap, "rag.chunk-overlap", o.RAG.ChunkOverlap, "Overlap between chunks")
	fs.IntVar(&o.RAG.TopK, "rag.top-k", o.RAG.TopK, "Number of results from similarity search")
	fs.StringVar(&o.RAG.Collection, "rag.collection", o.RAG.Collection, "Milvus collection name")
	fs.IntVar(&o.RAG.EmbeddingDim, "rag.embedding-dim", o.RAG.EmbeddingDim, "Embedding vector dimension")
	fs.StringVar(&o.RAG.DataDir, "rag.data-dir", o.RAG.DataDir, "Directory for storing documents")
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
	if err := o.Ollama.Validate(); err != nil {
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

// Complete completes the options.
func (o *Options) Complete() error {
	if err := o.Server.Complete(); err != nil {
		return err
	}
	return nil
}

// GetTimeout returns a reasonable timeout for RAG operations.
func (o *Options) GetTimeout() time.Duration {
	return o.Ollama.Timeout
}
