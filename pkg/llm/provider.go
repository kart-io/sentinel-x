// Package llm 提供统一的 LLM 供应商抽象层。
// 支持 Embedding 和 Chat 使用不同供应商的模型。
package llm

import (
	"context"
	"fmt"
	"sync"
)

// EmbeddingProvider 定义 Embedding 供应商接口。
type EmbeddingProvider interface {
	// Embed 为多个文本生成向量嵌入。
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle 为单个文本生成向量嵌入。
	EmbedSingle(ctx context.Context, text string) ([]float32, error)

	// Name 返回供应商名称。
	Name() string
}

// ChatProvider 定义 Chat 供应商接口。
type ChatProvider interface {
	// Chat 进行多轮对话。
	Chat(ctx context.Context, messages []Message) (string, error)

	// Generate 根据提示生成文本（单轮）。
	Generate(ctx context.Context, prompt string, systemPrompt string) (string, error)

	// Name 返回供应商名称。
	Name() string
}

// Message 表示对话中的一条消息。
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Role 定义消息角色。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Provider 同时支持 Embedding 和 Chat 的完整供应商。
type Provider interface {
	EmbeddingProvider
	ChatProvider
}

// ProviderFactory 供应商工厂函数类型。
type ProviderFactory func(config map[string]any) (Provider, error)

// EmbeddingProviderFactory Embedding 供应商工厂函数类型。
type EmbeddingProviderFactory func(config map[string]any) (EmbeddingProvider, error)

// ChatProviderFactory Chat 供应商工厂函数类型。
type ChatProviderFactory func(config map[string]any) (ChatProvider, error)

// registry 供应商注册表。
var registry = &providerRegistry{
	providers:          make(map[string]ProviderFactory),
	embeddingProviders: make(map[string]EmbeddingProviderFactory),
	chatProviders:      make(map[string]ChatProviderFactory),
}

type providerRegistry struct {
	mu                 sync.RWMutex
	providers          map[string]ProviderFactory
	embeddingProviders map[string]EmbeddingProviderFactory
	chatProviders      map[string]ChatProviderFactory
}

// RegisterProvider 注册完整供应商工厂。
func RegisterProvider(name string, factory ProviderFactory) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.providers[name] = factory
}

// RegisterEmbeddingProvider 注册 Embedding 供应商工厂。
func RegisterEmbeddingProvider(name string, factory EmbeddingProviderFactory) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.embeddingProviders[name] = factory
}

// RegisterChatProvider 注册 Chat 供应商工厂。
func RegisterChatProvider(name string, factory ChatProviderFactory) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.chatProviders[name] = factory
}

// NewProvider 根据名称创建完整供应商实例。
func NewProvider(name string, config map[string]any) (Provider, error) {
	registry.mu.RLock()
	factory, ok := registry.providers[name]
	registry.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	return factory(config)
}

// NewEmbeddingProvider 根据名称创建 Embedding 供应商实例。
// 优先查找专用 Embedding 工厂，其次查找完整供应商工厂。
func NewEmbeddingProvider(name string, config map[string]any) (EmbeddingProvider, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// 优先使用专用 Embedding 工厂
	if factory, ok := registry.embeddingProviders[name]; ok {
		return factory(config)
	}

	// 回退到完整供应商工厂
	if factory, ok := registry.providers[name]; ok {
		return factory(config)
	}

	return nil, fmt.Errorf("unknown embedding provider: %s", name)
}

// NewChatProvider 根据名称创建 Chat 供应商实例。
// 优先查找专用 Chat 工厂，其次查找完整供应商工厂。
func NewChatProvider(name string, config map[string]any) (ChatProvider, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// 优先使用专用 Chat 工厂
	if factory, ok := registry.chatProviders[name]; ok {
		return factory(config)
	}

	// 回退到完整供应商工厂
	if factory, ok := registry.providers[name]; ok {
		return factory(config)
	}

	return nil, fmt.Errorf("unknown chat provider: %s", name)
}

// ListProviders 列出所有已注册的供应商名称。
func ListProviders() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	seen := make(map[string]bool)
	var names []string

	for name := range registry.providers {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for name := range registry.embeddingProviders {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for name := range registry.chatProviders {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return names
}
