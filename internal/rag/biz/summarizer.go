package biz

import (
	"context"
	"fmt"
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// SummarizerConfig 摘要生成器配置。
type SummarizerConfig struct {
	// MaxTokens 摘要最大 token 长度。
	MaxTokens int
	// Model 使用的模型名称。
	Model string
}

// Summarizer 负责为文档节点生成摘要。
type Summarizer struct {
	chatProvider llm.ChatProvider
	config       *SummarizerConfig
}

// NewSummarizer 创建摘要生成器实例。
func NewSummarizer(chatProvider llm.ChatProvider, config *SummarizerConfig) *Summarizer {
	return &Summarizer{
		chatProvider: chatProvider,
		config:       config,
	}
}

// Summarize 为一组内容生成简洁摘要。
// 如果 LLM 调用失败，会降级到内容截断策略。
func (s *Summarizer) Summarize(ctx context.Context, contents []string) (string, error) {
	if len(contents) == 0 {
		return "", fmt.Errorf("内容列表为空")
	}

	// 1. 合并内容
	combined := s.combineContents(contents)
	if len(combined) == 0 {
		return "", fmt.Errorf("合并后的内容为空")
	}

	// 2. 构建 Prompt
	prompt := s.buildPrompt(combined)

	// 3. 调用 LLM 生成摘要
	summary, err := s.generateWithLLM(ctx, prompt)
	if err != nil {
		logger.Warnw("LLM 摘要生成失败，使用降级策略", "error", err.Error())
		// 降级策略：使用内容截断
		return s.fallbackSummary(combined), nil
	}

	// 4. 质量验证
	if !s.validateSummary(summary) {
		logger.Warnw("摘要质量验证失败，使用降级策略",
			"summary_length", len(summary),
		)
		return s.fallbackSummary(combined), nil
	}

	return summary, nil
}

// combineContents 合并多个内容，用换行符分隔。
func (s *Summarizer) combineContents(contents []string) string {
	// 过滤空内容
	nonEmpty := make([]string, 0, len(contents))
	for _, content := range contents {
		if trimmed := strings.TrimSpace(content); len(trimmed) > 0 {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}

	return strings.Join(nonEmpty, "\n\n")
}

// buildPrompt 构建摘要生成的 Prompt。
func (s *Summarizer) buildPrompt(content string) string {
	// 如果内容过长，先截断（避免超过 LLM 上下文限制）
	maxContentLen := 4000 // 约 1000 tokens
	if len(content) > maxContentLen {
		content = textutil.TruncateString(content, maxContentLen)
	}

	prompt := fmt.Sprintf(`你是一个专业的文档摘要助手。请对以下文本生成简洁摘要。

要求：
1. 长度不超过 200 个字符
2. 提取核心主题和关键信息
3. 保持语义完整性
4. 使用简体中文

原始文本：
%s

摘要：`, content)

	return prompt
}

// generateWithLLM 使用 LLM 生成摘要。
func (s *Summarizer) generateWithLLM(ctx context.Context, prompt string) (string, error) {
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	resp, err := s.chatProvider.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	if len(resp) == 0 {
		return "", fmt.Errorf("LLM 返回空内容")
	}

	// 清理摘要（去除前后空白）
	summary := strings.TrimSpace(resp)

	return summary, nil
}

// validateSummary 验证摘要质量。
// 返回 true 表示质量合格。
func (s *Summarizer) validateSummary(summary string) bool {
	length := len(summary)

	// 检查长度范围：20-250 字符
	// 太短说明生成失败，太长说明超出限制
	if length < 20 {
		return false
	}

	if length > 250 {
		return false
	}

	// 检查是否包含实际内容（非纯空白）
	if len(strings.TrimSpace(summary)) == 0 {
		return false
	}

	return true
}

// fallbackSummary 降级策略：使用内容截断生成摘要。
func (s *Summarizer) fallbackSummary(content string) string {
	// 截断到 200 字符
	summary := textutil.TruncateString(content, 200)

	// 清理空白字符
	summary = strings.TrimSpace(summary)

	// 如果截断后以句号结尾，保持完整；否则添加省略号
	if !strings.HasSuffix(summary, "。") && !strings.HasSuffix(summary, ".") {
		summary += "..."
	}

	return summary
}
