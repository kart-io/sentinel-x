package siliconflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/llm"
)

const testAPIKey = "test-key"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseURL != "https://api.siliconflow.cn/v1" {
		t.Errorf("expected BaseURL https://api.siliconflow.cn/v1, got %s", cfg.BaseURL)
	}
	if cfg.EmbedModel != "BAAI/bge-m3" {
		t.Errorf("expected EmbedModel BAAI/bge-m3, got %s", cfg.EmbedModel)
	}
	if cfg.ChatModel != "Qwen/Qwen2.5-7B-Instruct" {
		t.Errorf("expected ChatModel Qwen/Qwen2.5-7B-Instruct, got %s", cfg.ChatModel)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected Timeout 120s, got %v", cfg.Timeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]any
		wantError bool
	}{
		{
			name: "valid config",
			config: map[string]any{
				"api_key": testAPIKey,
			},
			wantError: false,
		},
		{
			name: "custom config",
			config: map[string]any{
				"api_key":     testAPIKey,
				"base_url":    "https://api.siliconflow.com/v1",
				"embed_model": "custom-embed",
				"chat_model":  "custom-chat",
			},
			wantError: false,
		},
		{
			name:      "missing api_key",
			config:    map[string]any{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("expected provider, got nil")
				}
				if provider != nil && provider.Name() != ProviderName {
					t.Errorf("expected provider name %s, got %s", ProviderName, provider.Name())
				}
			}
		})
	}
}

func TestProviderEmbed(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("expected path /embeddings, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// 检查请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("expected Authorization Bearer test-key")
		}

		// 返回模拟响应
		resp := embeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
				{
					Object:    "embedding",
					Embedding: []float32{0.4, 0.5, 0.6},
					Index:     1,
				},
			},
			Model: "BAAI/bge-m3",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建供应商
	cfg := DefaultConfig()
	cfg.BaseURL = server.URL
	cfg.APIKey = testAPIKey
	provider := NewProviderWithConfig(cfg)

	// 测试 Embed
	ctx := context.Background()
	texts := []string{"text1", "text2"}
	embeddings, err := provider.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(embeddings))
	}
	if len(embeddings[0]) != 3 {
		t.Errorf("expected embedding dimension 3, got %d", len(embeddings[0]))
	}
}

func TestProviderEmbedSingle(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := embeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "BAAI/bge-m3",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建供应商
	cfg := DefaultConfig()
	cfg.BaseURL = server.URL
	cfg.APIKey = testAPIKey
	provider := NewProviderWithConfig(cfg)

	// 测试 EmbedSingle
	ctx := context.Background()
	embedding, err := provider.EmbedSingle(ctx, "test text")
	if err != nil {
		t.Fatalf("EmbedSingle failed: %v", err)
	}

	if len(embedding) != 3 {
		t.Errorf("expected embedding dimension 3, got %d", len(embedding))
	}
}

func TestProviderChat(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// 检查请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("expected Authorization Bearer test-key")
		}

		// 返回模拟响应
		resp := chatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "Qwen/Qwen2.5-7B-Instruct",
			Choices: []struct {
				Index        int         `json:"index"`
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: chatMessage{
						Role:    "assistant",
						Content: "测试响应",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建供应商
	cfg := DefaultConfig()
	cfg.BaseURL = server.URL
	cfg.APIKey = testAPIKey
	provider := NewProviderWithConfig(cfg)

	// 测试 Chat
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "你好"},
	}
	response, err := provider.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != "测试响应" {
		t.Errorf("expected response '测试响应', got '%s'", response)
	}
}

func TestProviderGenerate(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "Qwen/Qwen2.5-7B-Instruct",
			Choices: []struct {
				Index        int         `json:"index"`
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: chatMessage{
						Role:    "assistant",
						Content: "生成的文本",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建供应商
	cfg := DefaultConfig()
	cfg.BaseURL = server.URL
	cfg.APIKey = testAPIKey
	provider := NewProviderWithConfig(cfg)

	// 测试 Generate
	ctx := context.Background()
	response, err := provider.Generate(ctx, "生成一段文本", "你是一个助手")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if response.Content != "生成的文本" {
		t.Errorf("expected response '生成的文本', got '%s'", response.Content)
	}
}

func TestProviderEmbedEmpty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIKey = testAPIKey
	provider := NewProviderWithConfig(cfg)

	ctx := context.Background()
	embeddings, err := provider.Embed(ctx, []string{})
	if err != nil {
		t.Fatalf("Embed with empty texts failed: %v", err)
	}

	if embeddings != nil {
		t.Error("expected nil embeddings for empty input")
	}
}

func TestNewProviderWithAdvancedParams(t *testing.T) {
	// 测试高级参数配置
	provider, err := NewProvider(map[string]any{
		"api_key":            testAPIKey,
		"temperature":        0.7,
		"top_p":              0.9,
		"top_k":              50,
		"min_p":              0.05,
		"max_tokens":         2000,
		"repetition_penalty": 1.1,
	})
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	p, ok := provider.(*Provider)
	if !ok {
		t.Fatal("provider is not *Provider type")
	}

	// 验证参数正确设置
	if p.config.Temperature != 0.7 {
		t.Errorf("expected Temperature 0.7, got %f", p.config.Temperature)
	}
	if p.config.TopP != 0.9 {
		t.Errorf("expected TopP 0.9, got %f", p.config.TopP)
	}
	if p.config.TopK != 50 {
		t.Errorf("expected TopK 50, got %d", p.config.TopK)
	}
	if p.config.MinP != 0.05 {
		t.Errorf("expected MinP 0.05, got %f", p.config.MinP)
	}
	if p.config.MaxTokens != 2000 {
		t.Errorf("expected MaxTokens 2000, got %d", p.config.MaxTokens)
	}
	if p.config.RepetitionPenalty != 1.1 {
		t.Errorf("expected RepetitionPenalty 1.1, got %f", p.config.RepetitionPenalty)
	}
}

func TestChatWithAdvancedParams(t *testing.T) {
	// 创建模拟服务器，验证请求中包含高级参数
	var receivedReq chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析请求体
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// 返回模拟响应
		resp := chatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "Qwen/Qwen2.5-7B-Instruct",
			Choices: []struct {
				Index        int         `json:"index"`
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: chatMessage{
						Role:    "assistant",
						Content: "测试响应",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建带高级参数的供应商
	cfg := DefaultConfig()
	cfg.BaseURL = server.URL
	cfg.APIKey = testAPIKey
	cfg.Temperature = 0.8
	cfg.TopP = 0.95
	cfg.TopK = 40
	cfg.MinP = 0.1
	cfg.MaxTokens = 1500
	cfg.RepetitionPenalty = 1.2
	provider := NewProviderWithConfig(cfg)

	// 测试 Chat
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "你好"},
	}
	_, err := provider.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// 验证请求中包含高级参数
	if receivedReq.Temperature != 0.8 {
		t.Errorf("expected Temperature 0.8, got %f", receivedReq.Temperature)
	}
	if receivedReq.TopP != 0.95 {
		t.Errorf("expected TopP 0.95, got %f", receivedReq.TopP)
	}
	if receivedReq.TopK != 40 {
		t.Errorf("expected TopK 40, got %d", receivedReq.TopK)
	}
	if receivedReq.MinP != 0.1 {
		t.Errorf("expected MinP 0.1, got %f", receivedReq.MinP)
	}
	if receivedReq.MaxTokens != 1500 {
		t.Errorf("expected MaxTokens 1500, got %d", receivedReq.MaxTokens)
	}
	if receivedReq.RepetitionPenalty != 1.2 {
		t.Errorf("expected RepetitionPenalty 1.2, got %f", receivedReq.RepetitionPenalty)
	}
}
