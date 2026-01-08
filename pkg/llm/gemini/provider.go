// Package gemini 提供 Google Gemini LLM 供应商实现。
// 支持 Gemini Pro 和 Gemini Pro Vision 模型。
package gemini

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

// ProviderName 是 Gemini 供应商的名称标识符
const ProviderName = "gemini"

func init() {
	llm.RegisterProvider(ProviderName, NewProvider)
}

// Config Gemini 供应商配置。
type Config struct {
	// BaseURL API 基础地址。
	BaseURL string `json:"base_url" mapstructure:"base_url"`

	// APIKey Google AI API 密钥。
	APIKey string `json:"api_key" mapstructure:"api_key"`

	// EmbedModel 用于生成嵌入的模型。
	EmbedModel string `json:"embed_model" mapstructure:"embed_model"`

	// ChatModel 用于对话的模型。
	ChatModel string `json:"chat_model" mapstructure:"chat_model"`

	// Timeout 请求超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// MaxRetries 最大重试次数。
	MaxRetries int `json:"max_retries" mapstructure:"max_retries"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://generativelanguage.googleapis.com/v1beta",
		EmbedModel: "text-embedding-004",
		ChatModel:  "gemini-1.5-flash",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// Provider Gemini 供应商实现。
type Provider struct {
	config *Config
	client *httpclient.Client
}

// NewProvider 从配置 map 创建 Gemini 供应商。
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

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini: api_key 是必需的")
	}

	return NewProviderWithConfig(cfg), nil
}

// NewProviderWithConfig 使用结构化配置创建 Gemini 供应商。
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

// embedRequest Gemini embedding API 请求体。
type embedRequest struct {
	Requests []embedContentRequest `json:"requests"`
}

type embedContentRequest struct {
	Model   string       `json:"model"`
	Content embedContent `json:"content"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

// embedResponse Gemini embedding API 响应体。
type embedResponse struct {
	Embeddings []struct {
		Values []float32 `json:"values"`
	} `json:"embeddings"`
}

// Embed 为多个文本生成向量嵌入。
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Gemini 使用 batchEmbedContents API
	requests := make([]embedContentRequest, len(texts))
	for i, text := range texts {
		requests[i] = embedContentRequest{
			Model: fmt.Sprintf("models/%s", p.config.EmbedModel),
			Content: embedContent{
				Parts: []embedPart{{Text: text}},
			},
		}
	}

	reqBody := embedRequest{Requests: requests}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:batchEmbedContents?key=%s",
		p.config.BaseURL, p.config.EmbedModel, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var embedResp embedResponse
	if err := p.client.DoJSON(req, &embedResp); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(embedResp.Embeddings))
	for i, emb := range embedResp.Embeddings {
		embeddings[i] = emb.Values
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

// chatRequest Gemini generateContent API 请求体。
type chatRequest struct {
	Contents          []chatContent     `json:"contents"`
	SystemInstruction *chatContent      `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

type chatContent struct {
	Role  string     `json:"role,omitempty"`
	Parts []chatPart `json:"parts"`
}

type chatPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// chatResponse Gemini generateContent API 响应体。
type chatResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// Chat 进行多轮对话。
func (p *Provider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	var contents []chatContent
	var systemInstruction *chatContent

	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			systemInstruction = &chatContent{
				Parts: []chatPart{{Text: msg.Content}},
			}
		case llm.RoleUser:
			contents = append(contents, chatContent{
				Role:  "user",
				Parts: []chatPart{{Text: msg.Content}},
			})
		case llm.RoleAssistant:
			contents = append(contents, chatContent{
				Role:  "model",
				Parts: []chatPart{{Text: msg.Content}},
			})
		}
	}

	reqBody := chatRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig: &generationConfig{
			Temperature:     0.7,
			TopP:            0.95,
			TopK:            40,
			MaxOutputTokens: 2048,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		p.config.BaseURL, p.config.ChatModel, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var chatResp chatResponse
	if err := p.client.DoJSON(req, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Candidates) == 0 || len(chatResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("未返回响应内容")
	}

	return chatResp.Candidates[0].Content.Parts[0].Text, nil
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

	var contents []chatContent
	var systemInstruction *chatContent

	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			systemInstruction = &chatContent{
				Parts: []chatPart{{Text: msg.Content}},
			}
		case llm.RoleUser:
			contents = append(contents, chatContent{
				Role:  "user",
				Parts: []chatPart{{Text: msg.Content}},
			})
		case llm.RoleAssistant:
			contents = append(contents, chatContent{
				Role:  "model",
				Parts: []chatPart{{Text: msg.Content}},
			})
		}
	}

	reqBody := chatRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig: &generationConfig{
			Temperature:     0.7,
			TopP:            0.95,
			TopK:            40,
			MaxOutputTokens: 2048,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		p.config.BaseURL, p.config.ChatModel, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var chatResp chatResponse
	if err := p.client.DoJSON(req, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Candidates) == 0 || len(chatResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("未返回响应内容")
	}

	// 构建响应，包含 token 使用情况
	// Gemini 使用不同的字段名称，需要转换
	response := &llm.GenerateResponse{
		Content: chatResp.Candidates[0].Content.Parts[0].Text,
		TokenUsage: &llm.TokenUsage{
			PromptTokens:     chatResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: chatResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      chatResp.UsageMetadata.TotalTokenCount,
		},
	}

	return response, nil
}
