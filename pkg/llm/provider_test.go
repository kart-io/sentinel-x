package llm

import (
	"context"
	"testing"
)

// mockProvider 模拟供应商实现，用于测试。
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

func (m *mockProvider) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockProvider) Chat(_ context.Context, _ []Message) (string, error) {
	return "mock response", nil
}

func (m *mockProvider) Generate(_ context.Context, _ string, _ string) (string, error) {
	return "mock generated text", nil
}

func TestRegisterAndNewProvider(t *testing.T) {
	// 注册测试供应商
	RegisterProvider("test-provider", func(config map[string]any) (Provider, error) {
		name := "test-provider"
		if n, ok := config["name"].(string); ok {
			name = n
		}
		return &mockProvider{name: name}, nil
	})

	// 测试创建供应商
	provider, err := NewProvider("test-provider", map[string]any{"name": "custom-name"})
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if provider.Name() != "custom-name" {
		t.Errorf("expected name 'custom-name', got '%s'", provider.Name())
	}
}

func TestNewProviderUnknown(t *testing.T) {
	_, err := NewProvider("unknown-provider", nil)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewEmbeddingProvider(t *testing.T) {
	// 注册专用 Embedding 供应商
	RegisterEmbeddingProvider("embed-only", func(config map[string]any) (EmbeddingProvider, error) {
		return &mockProvider{name: "embed-only"}, nil
	})

	provider, err := NewEmbeddingProvider("embed-only", nil)
	if err != nil {
		t.Fatalf("NewEmbeddingProvider failed: %v", err)
	}

	if provider.Name() != "embed-only" {
		t.Errorf("expected name 'embed-only', got '%s'", provider.Name())
	}

	// 测试回退到完整供应商
	provider2, err := NewEmbeddingProvider("test-provider", nil)
	if err != nil {
		t.Fatalf("NewEmbeddingProvider fallback failed: %v", err)
	}
	if provider2 == nil {
		t.Error("expected non-nil provider")
	}
}

func TestNewChatProvider(t *testing.T) {
	// 注册专用 Chat 供应商
	RegisterChatProvider("chat-only", func(config map[string]any) (ChatProvider, error) {
		return &mockProvider{name: "chat-only"}, nil
	})

	provider, err := NewChatProvider("chat-only", nil)
	if err != nil {
		t.Fatalf("NewChatProvider failed: %v", err)
	}

	if provider.Name() != "chat-only" {
		t.Errorf("expected name 'chat-only', got '%s'", provider.Name())
	}

	// 测试回退到完整供应商
	provider2, err := NewChatProvider("test-provider", nil)
	if err != nil {
		t.Fatalf("NewChatProvider fallback failed: %v", err)
	}
	if provider2 == nil {
		t.Error("expected non-nil provider")
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	if len(providers) == 0 {
		t.Error("expected at least one registered provider")
	}

	// 检查测试供应商是否在列表中
	found := false
	for _, p := range providers {
		if p == "test-provider" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'test-provider' in provider list")
	}
}

func TestMessageRole(t *testing.T) {
	tests := []struct {
		role     Role
		expected string
	}{
		{RoleSystem, "system"},
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("expected role '%s', got '%s'", tt.expected, string(tt.role))
		}
	}
}

func TestMockProviderEmbed(t *testing.T) {
	provider := &mockProvider{name: "test"}

	embeddings, err := provider.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 3 {
			t.Errorf("embedding %d: expected 3 dimensions, got %d", i, len(emb))
		}
	}
}

func TestMockProviderChat(t *testing.T) {
	provider := &mockProvider{name: "test"}

	messages := []Message{
		{Role: RoleUser, Content: "Hello"},
	}

	response, err := provider.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != "mock response" {
		t.Errorf("expected 'mock response', got '%s'", response)
	}
}

func TestMockProviderGenerate(t *testing.T) {
	provider := &mockProvider{name: "test"}

	response, err := provider.Generate(context.Background(), "prompt", "system")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if response != "mock generated text" {
		t.Errorf("expected 'mock generated text', got '%s'", response)
	}
}
