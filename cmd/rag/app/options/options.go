// Package options contains flags and options for initializing the RAG server.
package options

import (
	"fmt"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	ragsvc "github.com/kart-io/sentinel-x/internal/rag"
	cliflag "github.com/kart-io/sentinel-x/pkg/app/cliflag"
	cacheopts "github.com/kart-io/sentinel-x/pkg/options/cache"
	llmopts "github.com/kart-io/sentinel-x/pkg/options/llm"
	logopts "github.com/kart-io/sentinel-x/pkg/options/logger"
	middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	milvusopts "github.com/kart-io/sentinel-x/pkg/options/milvus"
	ragopts "github.com/kart-io/sentinel-x/pkg/options/rag"
	grpcopts "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// ServerOptions contains the configuration options for the server.
type ServerOptions struct {
	// HTTPOptions contains HTTP server configuration.
	HTTPOptions *httpopts.Options `json:"http" mapstructure:"http"`

	// GRPCOptions contains gRPC server configuration.
	GRPCOptions *grpcopts.Options `json:"grpc" mapstructure:"grpc"`

	// LogOptions contains logger configuration.
	LogOptions *logopts.Options `json:"log" mapstructure:"log"`

	// MilvusOptions contains Milvus database configuration.
	MilvusOptions *milvusopts.Options `json:"milvus" mapstructure:"milvus"`

	// EmbeddingOptions contains embedding provider configuration.
	EmbeddingOptions *llmopts.ProviderOptions `json:"embedding" mapstructure:"embedding"`

	// ChatOptions contains chat provider configuration.
	ChatOptions *llmopts.ProviderOptions `json:"chat" mapstructure:"chat"`

	// RAGOptions contains RAG-specific configuration.
	RAGOptions *ragopts.Options `json:"rag" mapstructure:"rag"`

	// CacheOptions contains cache configuration.
	CacheOptions *cacheopts.Options `json:"cache" mapstructure:"cache"`

	// RecoveryOptions contains recovery middleware configuration.
	RecoveryOptions *middlewareopts.RecoveryOptions `json:"recovery" mapstructure:"recovery"`

	// RequestIDOptions contains request ID middleware configuration.
	RequestIDOptions *middlewareopts.RequestIDOptions `json:"request-id" mapstructure:"request-id"`

	// LoggerOptions contains logger middleware configuration.
	LoggerOptions *middlewareopts.LoggerOptions `json:"logger" mapstructure:"logger"`

	// CORSOptions contains CORS middleware configuration.
	CORSOptions *middlewareopts.CORSOptions `json:"cors" mapstructure:"cors"`

	// TimeoutOptions contains timeout middleware configuration.
	TimeoutOptions *middlewareopts.TimeoutOptions `json:"timeout" mapstructure:"timeout"`

	// HealthOptions contains health check configuration.
	HealthOptions *middlewareopts.HealthOptions `json:"health" mapstructure:"health"`

	// MetricsOptions contains metrics configuration.
	MetricsOptions *middlewareopts.MetricsOptions `json:"metrics" mapstructure:"metrics"`

	// PprofOptions contains pprof configuration.
	PprofOptions *middlewareopts.PprofOptions `json:"pprof" mapstructure:"pprof"`

	// ShutdownTimeout is the timeout for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" mapstructure:"shutdown-timeout"`
}

// NewServerOptions creates a ServerOptions instance with default values.
func NewServerOptions() *ServerOptions {
	httpOpts := httpopts.NewOptions()
	httpOpts.Addr = ":8082"

	grpcOpts := grpcopts.NewOptions()
	grpcOpts.Addr = ":8102"

	return &ServerOptions{
		HTTPOptions:      httpOpts,
		GRPCOptions:      grpcOpts,
		LogOptions:       logopts.NewOptions(),
		MilvusOptions:    milvusopts.NewOptions(),
		EmbeddingOptions: llmopts.NewEmbeddingOptions(),
		ChatOptions:      llmopts.NewChatOptions(),
		RAGOptions:       ragopts.NewOptions(),
		CacheOptions:     cacheopts.NewOptions(),
		RecoveryOptions:  middlewareopts.NewRecoveryOptions(),
		RequestIDOptions: middlewareopts.NewRequestIDOptions(),
		LoggerOptions:    middlewareopts.NewLoggerOptions(),
		HealthOptions:    middlewareopts.NewHealthOptions(),
		MetricsOptions:   middlewareopts.NewMetricsOptions(),
		ShutdownTimeout:  30 * time.Second,
		// CORSOptions, TimeoutOptions, PprofOptions 默认禁用（nil）
	}
}

// Flags returns flags for a specific server by section name.
func (o *ServerOptions) Flags() (fss cliflag.NamedFlagSets) {
	o.HTTPOptions.AddFlags(fss.FlagSet("http"))
	o.GRPCOptions.AddFlags(fss.FlagSet("grpc"))
	o.LogOptions.AddFlags(fss.FlagSet("log"))
	o.MilvusOptions.AddFlags(fss.FlagSet("milvus"), "milvus.")
	o.EmbeddingOptions.AddFlags(fss.FlagSet("embedding"), "embedding.")
	o.ChatOptions.AddFlags(fss.FlagSet("chat"), "chat.")
	o.RAGOptions.AddFlags(fss.FlagSet("rag"), "rag.")
	o.CacheOptions.AddFlags(fss.FlagSet("cache"), "cache.")

	// misc flags
	fs := fss.FlagSet("misc")
	fs.DurationVar(&o.ShutdownTimeout, "shutdown-timeout", o.ShutdownTimeout, "Graceful shutdown timeout")

	return fss
}

// Complete completes all the required options.
func (o *ServerOptions) Complete() error {
	if err := o.HTTPOptions.Complete(); err != nil {
		return err
	}
	if err := o.GRPCOptions.Complete(); err != nil {
		return err
	}
	if err := o.EmbeddingOptions.Complete(); err != nil {
		return fmt.Errorf("embedding: %w", err)
	}
	if err := o.ChatOptions.Complete(); err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	if err := o.RAGOptions.Complete(); err != nil {
		return fmt.Errorf("rag: %w", err)
	}
	if err := o.CacheOptions.Complete(); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	return nil
}

// Validate checks whether the options in ServerOptions are valid.
func (o *ServerOptions) Validate() error {
	errs := []error{}

	errs = append(errs, o.HTTPOptions.Validate()...)
	errs = append(errs, o.GRPCOptions.Validate()...)
	errs = append(errs, o.LogOptions.Validate()...)
	errs = append(errs, o.MilvusOptions.Validate()...)
	errs = append(errs, o.EmbeddingOptions.Validate()...)
	errs = append(errs, o.ChatOptions.Validate()...)
	errs = append(errs, o.RAGOptions.Validate()...)
	errs = append(errs, o.CacheOptions.Validate()...)

	return utilerrors.NewAggregate(errs)
}

// Config builds a ragsvc.Config based on ServerOptions.
func (o *ServerOptions) Config() (*ragsvc.Config, error) {
	return &ragsvc.Config{
		HTTPOptions:      o.HTTPOptions,
		GRPCOptions:      o.GRPCOptions,
		LogOptions:       o.LogOptions,
		MilvusOptions:    o.MilvusOptions,
		EmbeddingOptions: o.EmbeddingOptions,
		ChatOptions:      o.ChatOptions,
		RAGOptions:       o.RAGOptions,
		CacheOptions:     o.CacheOptions,
		RecoveryOptions:  o.RecoveryOptions,
		RequestIDOptions: o.RequestIDOptions,
		LoggerOptions:    o.LoggerOptions,
		CORSOptions:      o.CORSOptions,
		TimeoutOptions:   o.TimeoutOptions,
		HealthOptions:    o.HealthOptions,
		MetricsOptions:   o.MetricsOptions,
		PprofOptions:     o.PprofOptions,
		ShutdownTimeout:  o.ShutdownTimeout,
	}, nil
}
