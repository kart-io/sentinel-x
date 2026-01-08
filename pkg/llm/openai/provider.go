// Package openai 提供 OpenAI LLM 供应商实现。
// 同时支持 OpenAI API 和兼容 OpenAI API 的服务（如 Azure OpenAI、LocalAI 等）。
//
// 基本用法示例：
//
//	import _ "github.com/kart-io/sentinel-x/pkg/llm/openai"
//	import "github.com/kart-io/sentinel-x/pkg/llm"
//
//	// 创建供应商
//	provider, err := llm.NewProvider("openai", map[string]any{
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
//	// 使用 Embedding API
//	embeddings, err := provider.Embed(ctx, []string{"文本1", "文本2"})
//
// 高级配置示例：
//
//	provider, err := llm.NewProvider("openai", map[string]any{
//	    "api_key":           "your-api-key",
//	    "chat_model":        "gpt-4o",                    // 可选：使用 GPT-4o 模型
//	    "embed_model":       "text-embedding-3-large",    // 可选：使用更大的 Embedding 模型
//	    "temperature":       0.7,                         // 可选：控制随机性
//	    "top_p":             0.9,                         // 可选：核采样参数
//	    "max_tokens":        2000,                        // 可选：最大生成 token 数
//	    "frequency_penalty": 0.5,                         // 可选：频率惩罚
//	    "presence_penalty":  0.5,                         // 可选：存在惩罚
//	    "stop":              []string{"\n\n", "END"},     // 可选：停止序列
//	})
package openai

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

// ProviderName 是 OpenAI 供应商的名称标识符
const ProviderName = "openai"

func init() {
	llm.RegisterProvider(ProviderName, NewProvider)
}

// Config OpenAI 供应商配置。
type Config struct {
	// BaseURL API 基础地址，默认为 OpenAI 官方地址。
	// 可设置为兼容 API 地址（如 Azure OpenAI、LocalAI 等）。
	BaseURL string `json:"base_url" mapstructure:"base_url"`

	// APIKey API 密钥。
	APIKey string `json:"api_key" mapstructure:"api_key"`

	// EmbedModel 用于生成嵌入的模型。
	EmbedModel string `json:"embed_model" mapstructure:"embed_model"`

	// ChatModel 用于对话的模型。
	ChatModel string `json:"chat_model" mapstructure:"chat_model"`

	// Timeout 请求超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// MaxRetries 最大重试次数。
	MaxRetries int `json:"max_retries" mapstructure:"max_retries"`

	// Organization 组织 ID（可选）。
	Organization string `json:"organization" mapstructure:"organization"`

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

	// FrequencyPenalty 频率惩罚系数，范围 -2.0-2.0。
	// 正值会根据新 token 在文本中的现有频率来惩罚它们，降低重复相同内容的可能性。
	// 默认值为 0，表示不设置此参数。
	FrequencyPenalty float64 `json:"frequency_penalty" mapstructure:"frequency_penalty"`

	// PresencePenalty 存在惩罚系数，范围 -2.0-2.0。
	// 正值会根据新 token 是否出现在文本中来惩罚它们，增加谈论新话题的可能性。
	// 默认值为 0，表示不设置此参数。
	PresencePenalty float64 `json:"presence_penalty" mapstructure:"presence_penalty"`

	// Stop 停止序列列表。
	// API 遇到这些字符串时会停止生成更多 token。
	// 默认值为 nil，表示不设置此参数。
	Stop []string `json:"stop" mapstructure:"stop"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://api.openai.com/v1",
		EmbedModel: "text-embedding-3-small",
		ChatModel:  "gpt-4o-mini",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// Provider OpenAI 供应商实现。
type Provider struct {
	config *Config
	client *httpclient.Client
}

// NewProvider 从配置 map 创建 OpenAI 供应商。
func NewProvider(configMap map[string]any) (llm.Provider, error) {
	cfg := DefaultConfig()

	if v, ok := configMap["base_url"].(string); ok && v != "" {
		cfg.BaseURL = v
	}
	if v, ok := configMap["api_key"].(string); ok && v != "" {
		cfg.APIKey = v
	}
	if v, ok := configMap["embed_model"].(string); ok && v != "" {
		cfg.EmbedModel = v
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
	if v, ok := configMap["organization"].(string); ok && v != "" {
		cfg.Organization = v
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
		// 支持 []string 和 []interface{} 两种类型
		switch val := v.(type) {
		case []string:
			cfg.Stop = val
		case []interface{}:
			stop := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					stop = append(stop, s)
				}
			}
			cfg.Stop = stop
		}
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: api_key 是必需的")
	}

	return NewProviderWithConfig(cfg), nil
}

// NewProviderWithConfig 使用结构化配置创建 OpenAI 供应商。
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

// embeddingRequest OpenAI embedding API 请求体。
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingResponse OpenAI embedding API 响应体。
type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed 为多个文本生成向量嵌入。
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := embeddingRequest{
		Model: p.config.EmbedModel,
		Input: texts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	p.setHeaders(req)

	var embedResp embeddingResponse
	if err := p.client.DoJSON(req, &embedResp); err != nil {
		return nil, err
	}

	// 按 index 排序确保顺序正确
	embeddings := make([][]float32, len(texts))
	for _, data := range embedResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

// EmbedSingle 为单个文本生成向量嵌入。
func (p *Provider) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未返回向量嵌入")
	}
	return embeddings[0], nil
}

// chatRequest OpenAI chat API 请求体。
type chatRequest struct {
	Model            string        `json:"model"`
	Messages         []chatMessage `json:"messages"`
	Stream           bool          `json:"stream"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	Temperature      float64       `json:"temperature,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
	Stop             []string      `json:"stop,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse OpenAI chat API 响应体。
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
	if p.config.MaxTokens > 0 {
		reqBody.MaxTokens = p.config.MaxTokens
	}
	if p.config.Temperature > 0 {
		reqBody.Temperature = p.config.Temperature
	}
	if p.config.TopP > 0 {
		reqBody.TopP = p.config.TopP
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
func (p *Provider) Generate(ctx context.Context, prompt string, systemPrompt string) (*llm.GenerateResponse, error) {
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
	if p.config.MaxTokens > 0 {
		reqBody.MaxTokens = p.config.MaxTokens
	}
	if p.config.Temperature > 0 {
		reqBody.Temperature = p.config.Temperature
	}
	if p.config.TopP > 0 {
		reqBody.TopP = p.config.TopP
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
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	p.setHeaders(req)

	var chatResp chatResponse
	if err := p.client.DoJSON(req, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("未返回响应内容")
	}

	// 构建响应，包含 token 使用情况
	response := &llm.GenerateResponse{
		Content: chatResp.Choices[0].Message.Content,
		TokenUsage: &llm.TokenUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// ListModels 列出可用模型。
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.BaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	p.setHeaders(req)

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := p.client.DoJSON(req, &result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}

	return models, nil
}

// setHeaders 设置请求头。
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", p.config.Organization)
	}
}
