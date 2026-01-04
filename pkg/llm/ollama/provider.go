// Package ollama 提供 Ollama LLM 供应商实现。
package ollama

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

// ProviderName 是 Ollama 供应商的名称标识符
const ProviderName = "ollama"

func init() {
	llm.RegisterProvider(ProviderName, NewProvider)
}

// Config Ollama 供应商配置。
type Config struct {
	BaseURL    string        `json:"base_url" mapstructure:"base_url"`
	EmbedModel string        `json:"embed_model" mapstructure:"embed_model"`
	ChatModel  string        `json:"chat_model" mapstructure:"chat_model"`
	Timeout    time.Duration `json:"timeout" mapstructure:"timeout"`
	MaxRetries int           `json:"max_retries" mapstructure:"max_retries"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "http://localhost:11434",
		EmbedModel: "nomic-embed-text",
		ChatModel:  "deepseek-r1:7b",
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}

// Provider Ollama 供应商实现。
type Provider struct {
	config *Config
	client *httpclient.Client
}

// NewProvider 从配置 map 创建 Ollama 供应商。
func NewProvider(configMap map[string]any) (llm.Provider, error) {
	cfg := DefaultConfig()

	if v, ok := configMap["base_url"].(string); ok && v != "" {
		cfg.BaseURL = v
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

	return NewProviderWithConfig(cfg), nil
}

// NewProviderWithConfig 使用结构化配置创建 Ollama 供应商。
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

// embedRequest Ollama embed API 请求体。
type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embedResponse Ollama embed API 响应体。
type embedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed 为多个文本生成向量嵌入。
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := embedRequest{
		Model: p.config.EmbedModel,
		Input: texts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var embedResp embedResponse
	if err := p.client.DoJSON(req, &embedResp); err != nil {
		return nil, err
	}

	return embedResp.Embeddings, nil
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

// chatRequest Ollama chat API 请求体。
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse Ollama chat API 响应体。
type chatResponse struct {
	Model   string      `json:"model"`
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
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

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var chatResp chatResponse
	if err := p.client.DoJSON(req, &chatResp); err != nil {
		return "", err
	}

	return chatResp.Message.Content, nil
}

// generateRequest Ollama generate API 请求体。
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	System string `json:"system,omitempty"`
}

// generateResponse Ollama generate API 响应体。
type generateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate 根据提示生成文本。
func (p *Provider) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	reqBody := generateRequest{
		Model:  p.config.ChatModel,
		Prompt: prompt,
		Stream: false,
		System: systemPrompt,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var genResp generateResponse
	if err := p.client.DoJSON(req, &genResp); err != nil {
		return "", err
	}

	return genResp.Response, nil
}

// Ping 检查 Ollama 服务是否可用。
func (p *Provider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := p.client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务不可用，状态码 %d", resp.StatusCode)
	}

	return nil
}

// ListModels 列出可用模型。
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := p.client.DoJSON(req, &result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}
