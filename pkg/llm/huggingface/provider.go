// Package huggingface 提供 HuggingFace Inference API 供应商实现。
// 支持 HuggingFace Hub 上的模型进行 Embedding 和 Text Generation。
package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kart-io/sentinel-x/pkg/llm"
)

const ProviderName = "huggingface"

func init() {
	llm.RegisterProvider(ProviderName, NewProvider)
}

// Config HuggingFace 供应商配置。
type Config struct {
	// BaseURL API 基础地址。
	BaseURL string `json:"base_url" mapstructure:"base_url"`

	// APIKey HuggingFace API Token。
	APIKey string `json:"api_key" mapstructure:"api_key"`

	// EmbedModel 用于生成嵌入的模型 ID。
	EmbedModel string `json:"embed_model" mapstructure:"embed_model"`

	// ChatModel 用于对话的模型 ID。
	ChatModel string `json:"chat_model" mapstructure:"chat_model"`

	// Timeout 请求超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// MaxRetries 最大重试次数。
	MaxRetries int `json:"max_retries" mapstructure:"max_retries"`

	// WaitForModel 如果模型正在加载，是否等待。
	WaitForModel bool `json:"wait_for_model" mapstructure:"wait_for_model"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		BaseURL:      "https://api-inference.huggingface.co",
		EmbedModel:   "sentence-transformers/all-MiniLM-L6-v2",
		ChatModel:    "mistralai/Mistral-7B-Instruct-v0.2",
		Timeout:      120 * time.Second,
		MaxRetries:   3,
		WaitForModel: true,
	}
}

// Provider HuggingFace 供应商实现。
type Provider struct {
	config     *Config
	httpClient *http.Client
}

// NewProvider 从配置 map 创建 HuggingFace 供应商。
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
	if v, ok := configMap["wait_for_model"].(bool); ok {
		cfg.WaitForModel = v
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("huggingface: api_key 是必需的")
	}

	return NewProviderWithConfig(cfg), nil
}

// NewProviderWithConfig 使用结构化配置创建 HuggingFace 供应商。
func NewProviderWithConfig(cfg *Config) *Provider {
	return &Provider{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Name 返回供应商名称。
func (p *Provider) Name() string {
	return ProviderName
}

// embeddingRequest HuggingFace Feature Extraction API 请求体。
type embeddingRequest struct {
	Inputs  []string          `json:"inputs"`
	Options *embeddingOptions `json:"options,omitempty"`
}

type embeddingOptions struct {
	WaitForModel bool `json:"wait_for_model,omitempty"`
}

// Embed 为多个文本生成向量嵌入。
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := embeddingRequest{
		Inputs: texts,
	}
	if p.config.WaitForModel {
		reqBody.Options = &embeddingOptions{WaitForModel: true}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/pipeline/feature-extraction/%s", p.config.BaseURL, p.config.EmbedModel)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.doRequestWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败，状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// HuggingFace 返回 [][]float32 或 [][][]float32（需要取平均）
	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		// 尝试解析为 3D 数组（某些模型返回 token 级别的嵌入）
		var tokenEmbeddings [][][]float32
		if err2 := json.NewDecoder(resp.Body).Decode(&tokenEmbeddings); err2 != nil {
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		// 对 token 嵌入取平均
		embeddings = make([][]float32, len(tokenEmbeddings))
		for i, tokens := range tokenEmbeddings {
			if len(tokens) == 0 {
				continue
			}
			dim := len(tokens[0])
			embeddings[i] = make([]float32, dim)
			for _, token := range tokens {
				for j, v := range token {
					embeddings[i][j] += v
				}
			}
			for j := range embeddings[i] {
				embeddings[i][j] /= float32(len(tokens))
			}
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

// chatRequest HuggingFace Text Generation API 请求体。
type chatRequest struct {
	Inputs     string       `json:"inputs"`
	Parameters *chatParams  `json:"parameters,omitempty"`
	Options    *chatOptions `json:"options,omitempty"`
}

type chatParams struct {
	MaxNewTokens   int     `json:"max_new_tokens,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	TopP           float64 `json:"top_p,omitempty"`
	DoSample       bool    `json:"do_sample,omitempty"`
	ReturnFullText bool    `json:"return_full_text,omitempty"`
}

type chatOptions struct {
	WaitForModel bool `json:"wait_for_model,omitempty"`
}

// chatResponse HuggingFace Text Generation API 响应体。
type chatResponse struct {
	GeneratedText string `json:"generated_text"`
}

// Chat 进行多轮对话。
func (p *Provider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	// 将消息格式化为对话模板
	prompt := formatMessages(messages)
	return p.generate(ctx, prompt)
}

// Generate 根据提示生成文本。
func (p *Provider) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	fullPrompt := prompt
	if systemPrompt != "" {
		fullPrompt = fmt.Sprintf("[INST] %s [/INST]\n\n%s", systemPrompt, prompt)
	}
	return p.generate(ctx, fullPrompt)
}

func (p *Provider) generate(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Inputs: prompt,
		Parameters: &chatParams{
			MaxNewTokens:   1024,
			Temperature:    0.7,
			TopP:           0.95,
			DoSample:       true,
			ReturnFullText: false,
		},
	}
	if p.config.WaitForModel {
		reqBody.Options = &chatOptions{WaitForModel: true}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s", p.config.BaseURL, p.config.ChatModel)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.doRequestWithRetry(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("请求失败，状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var responses []chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(responses) == 0 {
		return "", fmt.Errorf("未返回响应内容")
	}

	return responses[0].GeneratedText, nil
}

// formatMessages 将消息格式化为 Mistral 对话模板。
func formatMessages(messages []llm.Message) string {
	var result string
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			result += fmt.Sprintf("[INST] %s [/INST]\n", msg.Content)
		case llm.RoleUser:
			result += fmt.Sprintf("[INST] %s [/INST]\n", msg.Content)
		case llm.RoleAssistant:
			result += msg.Content + "\n"
		}
	}
	return result
}

// setHeaders 设置请求头。
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
}

// doRequestWithRetry 带重试的请求执行。
func (p *Provider) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error
	for i := 0; i <= p.config.MaxRetries; i++ {
		resp, err := p.httpClient.Do(req)
		if err == nil {
			// 503 表示模型正在加载，可以重试
			if resp.StatusCode < 500 || resp.StatusCode == 503 && i < p.config.MaxRetries {
				if resp.StatusCode == 503 {
					resp.Body.Close()
					time.Sleep(time.Duration(i+1) * 2 * time.Second) // 模型加载需要更长时间
					continue
				}
				return resp, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("服务器错误，状态码 %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if i < p.config.MaxRetries {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	return nil, lastErr
}
