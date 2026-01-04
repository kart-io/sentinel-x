// Package deepseek 提供 DeepSeek LLM 供应商实现。
// DeepSeek API 兼容 OpenAI 格式，但有自己的特定模型。
//
// 基本用法示例：
//
//	import _ "github.com/kart-io/sentinel-x/pkg/llm/deepseek"
//	import "github.com/kart-io/sentinel-x/pkg/llm"
//
//	// 创建供应商
//	provider, err := llm.NewProvider("deepseek", map[string]any{
//	    "api_key": "your-api-key",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 使用 Chat API
//	response, err := provider.Chat(ctx, []llm.Message{
//	    {Role: llm.RoleUser, Content: "你好"},
//	})
//
// 高级配置示例：
//
//	provider, err := llm.NewProvider("deepseek", map[string]any{
//	    "api_key":           "your-api-key",
//	    "chat_model":        "deepseek-chat",
//	    "temperature":       0.7,                    // 控制随机性
//	    "top_p":             0.9,                    // 核采样
//	    "max_tokens":        2000,                   // 最大生成 token 数
//	    "frequency_penalty": 0.5,                    // 频率惩罚
//	    "presence_penalty":  0.5,                    // 存在惩罚
//	    "stop":              []string{"结束", "END"}, // 停止序列
//	})
package deepseek

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kart-io/sentinel-x/pkg/llm"
	"github.com/kart-io/sentinel-x/pkg/utils/httpclient"
	"github.com/kart-io/sentinel-x/pkg/utils/json"
)

// ProviderName 是 DeepSeek 供应商的名称标识符
const ProviderName = "deepseek"

func init() {
	llm.RegisterProvider(ProviderName, NewProvider)
}

// Config DeepSeek 供应商配置。
type Config struct {
	// BaseURL API 基础地址。
	BaseURL string `json:"base_url" mapstructure:"base_url"`

	// APIKey API 密钥。
	APIKey string `json:"api_key" mapstructure:"api_key"`

	// ChatModel 用于对话的模型。
	ChatModel string `json:"chat_model" mapstructure:"chat_model"`

	// Timeout 请求超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// MaxRetries 最大重试次数。
	MaxRetries int `json:"max_retries" mapstructure:"max_retries"`

	// Temperature 控制生成文本的随机性，范围 0.0-2.0。
	// 较低的值（如 0.2）使输出更确定，较高的值（如 1.8）使输出更随机。
	// 默认值为 0，表示不设置此参数，使用 API 默认值。
	Temperature float64 `json:"temperature" mapstructure:"temperature"`

	// TopP 核采样参数，范围 0.0-1.0。
	// 控制累积概率阈值，默认值为 0，表示不设置此参数。
	TopP float64 `json:"top_p" mapstructure:"top_p"`

	// MaxTokens 最大生成 token 数。
	// 默认值为 0，表示不设置此参数，使用 API 默认值。
	MaxTokens int `json:"max_tokens" mapstructure:"max_tokens"`

	// FrequencyPenalty 频率惩罚系数，范围 -2.0 到 2.0。
	// 正值会根据新 token 在文本中的现有频率来降低其被采样的可能性，减少重复。
	// 默认值为 0，表示不设置此参数。
	FrequencyPenalty float64 `json:"frequency_penalty" mapstructure:"frequency_penalty"`

	// PresencePenalty 存在惩罚系数，范围 -2.0 到 2.0。
	// 正值会根据新 token 是否已出现在文本中来降低其被采样的可能性，鼓励新话题。
	// 默认值为 0，表示不设置此参数。
	PresencePenalty float64 `json:"presence_penalty" mapstructure:"presence_penalty"`

	// Stop 停止序列，最多 4 个字符串。
	// 当生成的文本包含这些序列之一时，API 将停止生成。
	Stop []string `json:"stop" mapstructure:"stop"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://api.deepseek.com",
		ChatModel:  "deepseek-chat",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// Provider DeepSeek 供应商实现。
type Provider struct {
	config *Config
	client *httpclient.Client
}

// NewProvider 从配置 map 创建 DeepSeek 供应商。
func NewProvider(configMap map[string]any) (llm.Provider, error) {
	cfg := DefaultConfig()

	if v, ok := configMap["base_url"].(string); ok && v != "" {
		cfg.BaseURL = v
	}
	if v, ok := configMap["api_key"].(string); ok && v != "" {
		cfg.APIKey = v
	}
	if v, ok := configMap["chat_model"].(string); ok && v != "" {
		cfg.ChatModel = v
	}
	if v, ok := configMap["timeout"].(time.Duration); ok && v > 0 {
		cfg.Timeout = v
	}
	if v, ok := configMap["max_retries"].(int); ok && v > 0 {
		cfg.MaxRetries = v
	}

	// 解析生成参数
	if v, ok := configMap["temperature"].(float64); ok {
		cfg.Temperature = v
	}
	if v, ok := configMap["top_p"].(float64); ok {
		cfg.TopP = v
	}
	if v, ok := configMap["max_tokens"].(int); ok {
		cfg.MaxTokens = v
	}
	if v, ok := configMap["frequency_penalty"].(float64); ok {
		cfg.FrequencyPenalty = v
	}
	if v, ok := configMap["presence_penalty"].(float64); ok {
		cfg.PresencePenalty = v
	}
	if v, ok := configMap["stop"]; ok {
		// 支持 []string 和 []any 类型
		switch stop := v.(type) {
		case []string:
			cfg.Stop = stop
		case []any:
			cfg.Stop = make([]string, len(stop))
			for i, s := range stop {
				if str, ok := s.(string); ok {
					cfg.Stop[i] = str
				}
			}
		}
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("deepseek: api_key 是必需的")
	}

	return NewProviderWithConfig(cfg), nil
}

// NewProviderWithConfig 使用结构化配置创建 DeepSeek 供应商。
func NewProviderWithConfig(cfg *Config) *Provider {
	return &Provider{
		config: cfg,
		client: httpclient.NewClient(cfg.Timeout, cfg.MaxRetries),
	}
}

// Name 返回供应商名称。
func (p *Provider) Name() string {
	return ProviderName
}

// Embed DeepSeek 目前不支持 Embedding API，返回错误。
func (p *Provider) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, fmt.Errorf("deepseek: 不支持 Embedding API，请使用其他供应商")
}

// EmbedSingle DeepSeek 目前不支持 Embedding API，返回错误。
func (p *Provider) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("deepseek: 不支持 Embedding API，请使用其他供应商")
}

// chatRequest DeepSeek chat API 请求体（兼容 OpenAI 格式）。
type chatRequest struct {
	Model            string        `json:"model"`
	Messages         []chatMessage `json:"messages"`
	Stream           bool          `json:"stream"`
	Temperature      float64       `json:"temperature,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
	Stop             []string      `json:"stop,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse DeepSeek chat API 响应体。
type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat 进行多轮对话。
func (p *Provider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	chatMessages := make([]chatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = chatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	reqBody := chatRequest{
		Model:    p.config.ChatModel,
		Messages: chatMessages,
		Stream:   false,
	}

	// 应用配置的生成参数（仅在非零值时设置）
	if p.config.Temperature > 0 {
		reqBody.Temperature = p.config.Temperature
	}
	if p.config.TopP > 0 {
		reqBody.TopP = p.config.TopP
	}
	if p.config.MaxTokens > 0 {
		reqBody.MaxTokens = p.config.MaxTokens
	}
	if p.config.FrequencyPenalty != 0 {
		reqBody.FrequencyPenalty = p.config.FrequencyPenalty
	}
	if p.config.PresencePenalty != 0 {
		reqBody.PresencePenalty = p.config.PresencePenalty
	}
	if len(p.config.Stop) > 0 {
		reqBody.Stop = p.config.Stop
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	p.setHeaders(req)

	var chatResp chatResponse
	if err := p.client.DoJSON(req, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("未返回响应内容")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Generate 根据提示生成文本。
func (p *Provider) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []llm.Message{}
	if systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: systemPrompt,
		})
	}
	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: prompt,
	})

	return p.Chat(ctx, messages)
}

// setHeaders 设置请求头。
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
}
