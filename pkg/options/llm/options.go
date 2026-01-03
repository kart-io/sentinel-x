// Package llm provides LLM provider configuration options.
package llm

import (
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

var _ options.IOptions = (*ProviderOptions)(nil)

// ProviderOptions 定义 LLM 供应商配置。
type ProviderOptions struct {
	// Provider 供应商名称（ollama, openai, deepseek 等）。
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

// NewProviderOptions 创建默认 LLM 供应商配置。
func NewProviderOptions() *ProviderOptions {
	return &ProviderOptions{
		Provider:   "ollama",
		BaseURL:    "http://localhost:11434",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// NewEmbeddingOptions 创建默认 Embedding 供应商配置。
func NewEmbeddingOptions() *ProviderOptions {
	opts := NewProviderOptions()
	opts.Model = "nomic-embed-text"
	return opts
}

// NewChatOptions 创建默认 Chat 供应商配置。
func NewChatOptions() *ProviderOptions {
	opts := NewProviderOptions()
	opts.Model = "deepseek-r1:7b"
	return opts
}

// ToConfigMap 转换为配置 map，用于供应商工厂。
func (o *ProviderOptions) ToConfigMap() map[string]any {
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

// AddFlags adds flags for LLM provider options to the specified FlagSet.
func (o *ProviderOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Provider, options.Join(prefixes...)+"llm.provider", o.Provider, "LLM provider (ollama, openai, deepseek).")
	fs.StringVar(&o.BaseURL, options.Join(prefixes...)+"llm.base-url", o.BaseURL, "LLM API base URL.")
	fs.StringVar(&o.APIKey, options.Join(prefixes...)+"llm.api-key", o.APIKey, "LLM API key.")
	fs.StringVar(&o.Model, options.Join(prefixes...)+"llm.model", o.Model, "LLM model name.")
	fs.DurationVar(&o.Timeout, options.Join(prefixes...)+"llm.timeout", o.Timeout, "LLM request timeout.")
	fs.IntVar(&o.MaxRetries, options.Join(prefixes...)+"llm.max-retries", o.MaxRetries, "LLM maximum number of retries.")
	fs.StringVar(&o.Organization, options.Join(prefixes...)+"llm.organization", o.Organization, "LLM organization ID (optional).")
}

// Validate validates the LLM provider options.
func (o *ProviderOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if o.Provider == "" {
		errs = append(errs, fmt.Errorf("provider is required"))
	}
	if o.BaseURL == "" {
		errs = append(errs, fmt.Errorf("base-url is required"))
	}
	if o.Model == "" {
		errs = append(errs, fmt.Errorf("model is required"))
	}
	// OpenAI 供应商需要 API key
	if o.Provider == "openai" && o.APIKey == "" {
		errs = append(errs, fmt.Errorf("api-key is required for openai provider"))
	}
	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("timeout must be positive"))
	}
	return errs
}

// Complete completes the LLM provider options with defaults.
func (o *ProviderOptions) Complete() error {
	if o.MaxRetries <= 0 {
		o.MaxRetries = 3
	}
	return nil
}
