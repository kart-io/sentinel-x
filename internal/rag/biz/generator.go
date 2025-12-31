package biz

import (
	"context"
	"fmt"
	"strings"

	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// GeneratorConfig 生成器配置。
type GeneratorConfig struct {
	// SystemPrompt 系统提示词模板。
	SystemPrompt string
}

// Generator 负责答案生成。
type Generator struct {
	chatProvider llm.ChatProvider
	config       *GeneratorConfig
}

// NewGenerator 创建生成器实例。
func NewGenerator(chatProvider llm.ChatProvider, config *GeneratorConfig) *Generator {
	return &Generator{
		chatProvider: chatProvider,
		config:       config,
	}
}

// GenerateAnswer 根据检索结果生成答案。
func (g *Generator) GenerateAnswer(ctx context.Context, question string, results []*store.SearchResult) (string, error) {
	if len(results) == 0 {
		return "I couldn't find any relevant information in the knowledge base.", nil
	}

	// 构建上下文
	var contextBuilder strings.Builder
	for i, result := range results {
		contextBuilder.WriteString(fmt.Sprintf("[%d] From %s - %s:\n%s\n\n",
			i+1, result.DocumentName, result.Section, result.Content))
	}

	// 生成提示词
	prompt := strings.ReplaceAll(g.config.SystemPrompt, "{{context}}", contextBuilder.String())
	prompt = strings.ReplaceAll(prompt, "{{question}}", question)

	// 调用 LLM 生成答案
	answer, err := g.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, nil
}
