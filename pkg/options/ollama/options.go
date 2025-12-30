// Package ollamaopts provides options for Ollama client configuration.
package ollamaopts

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

// Options contains Ollama client configuration.
type Options struct {
	// BaseURL is the Ollama API base URL.
	BaseURL string `json:"base-url" mapstructure:"base-url"`

	// EmbedModel is the model for generating embeddings.
	EmbedModel string `json:"embed-model" mapstructure:"embed-model"`

	// ChatModel is the model for chat completion.
	ChatModel string `json:"chat-model" mapstructure:"chat-model"`

	// Timeout for API requests.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// MaxRetries is the maximum number of retries for failed requests.
	MaxRetries int `json:"max-retries" mapstructure:"max-retries"`
}

// NewOptions creates new Options with defaults.
func NewOptions() *Options {
	return &Options{
		BaseURL:    "http://localhost:11434",
		EmbedModel: "nomic-embed-text",
		ChatModel:  "deepseek-r1:7b",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// AddFlags adds flags to the flagset.
func (o *Options) AddFlags(fs *pflag.FlagSet, prefix string) {
	fs.StringVar(&o.BaseURL, prefix+"base-url", o.BaseURL, "Ollama API base URL")
	fs.StringVar(&o.EmbedModel, prefix+"embed-model", o.EmbedModel, "Model for embeddings")
	fs.StringVar(&o.ChatModel, prefix+"chat-model", o.ChatModel, "Model for chat completion")
	fs.DurationVar(&o.Timeout, prefix+"timeout", o.Timeout, "Request timeout")
	fs.IntVar(&o.MaxRetries, prefix+"max-retries", o.MaxRetries, "Max retries for failed requests")
}

// Validate validates the options.
func (o *Options) Validate() error {
	if o.BaseURL == "" {
		return fmt.Errorf("ollama base-url is required")
	}
	if o.EmbedModel == "" {
		return fmt.Errorf("ollama embed-model is required")
	}
	if o.ChatModel == "" {
		return fmt.Errorf("ollama chat-model is required")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("ollama timeout must be positive")
	}
	return nil
}
