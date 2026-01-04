package openai

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
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected BaseURL https://api.openai.com/v1, got %s", cfg.BaseURL)
	}
	if cfg.EmbedModel != "text-embedding-3-small" {
		t.Errorf("expected EmbedModel text-embedding-3-small, got %s", cfg.EmbedModel)
	}
	if cfg.ChatModel != "gpt-4o-mini" {
		t.Errorf("expected ChatModel gpt-4o-mini, got %s", cfg.ChatModel)
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
				"api_key":      testAPIKey,
				"base_url":     "https://api.openai.com/v1",
				"embed_model":  "text-embedding-3-large",
				"chat_model":   "gpt-4o",
				"organization": "org-123",
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
			Model: "text-embedding-3-small",
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
			Model: "text-embedding-3-small",
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
			Model:   "gpt-4o-mini",
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
			Model:   "gpt-4o-mini",
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

	if response != "生成的文本" {
		t.Errorf("expected response '生成的文本', got '%s'", response)
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
		"api_key":           testAPIKey,
		"temperature":       0.7,
		"top_p":             0.9,
		"max_tokens":        2000,
		"frequency_penalty": 0.5,
		"presence_penalty":  0.5,
		"stop":              []string{"\n\n", "END"},
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
	if p.config.MaxTokens != 2000 {
		t.Errorf("expected MaxTokens 2000, got %d", p.config.MaxTokens)
	}
	if p.config.FrequencyPenalty != 0.5 {
		t.Errorf("expected FrequencyPenalty 0.5, got %f", p.config.FrequencyPenalty)
	}
	if p.config.PresencePenalty != 0.5 {
		t.Errorf("expected PresencePenalty 0.5, got %f", p.config.PresencePenalty)
	}
	if len(p.config.Stop) != 2 {
		t.Errorf("expected 2 stop sequences, got %d", len(p.config.Stop))
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
			Model:   "gpt-4o-mini",
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
	cfg.MaxTokens = 1500
	cfg.FrequencyPenalty = 0.6
	cfg.PresencePenalty = 0.4
	cfg.Stop = []string{"\n", "END"}
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
	if receivedReq.MaxTokens != 1500 {
		t.Errorf("expected MaxTokens 1500, got %d", receivedReq.MaxTokens)
	}
	if receivedReq.FrequencyPenalty != 0.6 {
		t.Errorf("expected FrequencyPenalty 0.6, got %f", receivedReq.FrequencyPenalty)
	}
	if receivedReq.PresencePenalty != 0.4 {
		t.Errorf("expected PresencePenalty 0.4, got %f", receivedReq.PresencePenalty)
	}
	if len(receivedReq.Stop) != 2 {
		t.Errorf("expected 2 stop sequences, got %d", len(receivedReq.Stop))
	}
}

func TestStopSequencesInterfaceSlice(t *testing.T) {
	// 测试 []interface{} 类型的 stop 参数
	provider, err := NewProvider(map[string]any{
		"api_key": testAPIKey,
		"stop":    []interface{}{"\n", "END", "STOP"},
	})
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	p, ok := provider.(*Provider)
	if !ok {
		t.Fatal("provider is not *Provider type")
	}

	if len(p.config.Stop) != 3 {
		t.Errorf("expected 3 stop sequences, got %d", len(p.config.Stop))
	}
	if p.config.Stop[0] != "\n" {
		t.Errorf("expected first stop sequence '\\n', got '%s'", p.config.Stop[0])
	}
}

func TestOrganizationHeader(t *testing.T) {
	// 创建模拟服务器，验证 Organization 头
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查 Organization 头
		if r.Header.Get("OpenAI-Organization") != "org-123" {
			t.Error("expected OpenAI-Organization header org-123")
		}

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
			Model: "text-embedding-3-small",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建带 Organization 的供应商
	cfg := DefaultConfig()
	cfg.BaseURL = server.URL
	cfg.APIKey = testAPIKey
	cfg.Organization = "org-123"
	provider := NewProviderWithConfig(cfg)

	// 测试 Embed
	ctx := context.Background()
	_, err := provider.EmbedSingle(ctx, "test")
	if err != nil {
		t.Fatalf("EmbedSingle failed: %v", err)
	}
}
