package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/llm"
)

// mockChatProvider 模拟 ChatProvider 用于测试。
type mockChatProvider struct {
	response string
	err      error
}

func (m *mockChatProvider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockChatProvider) Generate(ctx context.Context, prompt string, systemPrompt string) (*llm.GenerateResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockChatProvider) Name() string {
	return "mock"
}

// TestSummarizer_Summarize_Success 测试成功生成摘要。
func TestSummarizer_Summarize_Success(t *testing.T) {
	mockProvider := &mockChatProvider{
		response: "这是一段测试摘要，长度合适。",
	}

	config := &SummarizerConfig{
		MaxTokens: 200,
		Model:     "test-model",
	}

	summarizer := NewSummarizer(mockProvider, config)

	contents := []string{
		"这是第一段内容，包含一些重要信息。",
		"这是第二段内容，也包含一些关键数据。",
	}

	summary, err := summarizer.Summarize(context.Background(), contents)
	if err != nil {
		t.Fatalf("Summarize() 返回错误: %v", err)
	}

	if summary != mockProvider.response {
		t.Errorf("期望摘要 = %q, 实际 = %q", mockProvider.response, summary)
	}
}

// TestSummarizer_Summarize_LLMFailure 测试 LLM 失败时的降级处理。
func TestSummarizer_Summarize_LLMFailure(t *testing.T) {
	mockProvider := &mockChatProvider{
		err: errors.New("LLM 服务不可用"),
	}

	config := &SummarizerConfig{
		MaxTokens: 200,
		Model:     "test-model",
	}

	summarizer := NewSummarizer(mockProvider, config)

	contents := []string{
		"这是一段测试内容，用于验证降级策略。当 LLM 失败时，应该使用内容截断作为降级方案。",
	}

	summary, err := summarizer.Summarize(context.Background(), contents)
	if err != nil {
		t.Fatalf("Summarize() 返回错误: %v", err)
	}

	// 降级策略应该返回截断的内容
	if len(summary) == 0 {
		t.Error("降级策略应该返回非空摘要")
	}

	if len(summary) > 250 {
		t.Errorf("降级摘要长度超过限制: %d", len(summary))
	}
}

// TestSummarizer_Summarize_QualityValidation 测试质量验证。
func TestSummarizer_Summarize_QualityValidation(t *testing.T) {
	tests := []struct {
		name           string
		llmResponse    string
		expectFallback bool
	}{
		{
			name:           "摘要过短",
			llmResponse:    "短",
			expectFallback: true,
		},
		{
			name:           "摘要过长",
			llmResponse:    string(make([]byte, 300)), // 300 字符
			expectFallback: true,
		},
		{
			name:           "摘要长度合适",
			llmResponse:    "这是一段长度合适的摘要，包含足够的信息。",
			expectFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &mockChatProvider{
				response: tt.llmResponse,
			}

			summarizer := NewSummarizer(mockProvider, nil)

			contents := []string{"测试内容"}
			summary, err := summarizer.Summarize(context.Background(), contents)
			if err != nil {
				t.Fatalf("Summarize() 返回错误: %v", err)
			}

			if tt.expectFallback {
				// 如果期望降级，摘要不应该等于 LLM 响应
				if summary == tt.llmResponse {
					t.Error("期望使用降级策略，但返回了 LLM 响应")
				}
			} else {
				// 如果不期望降级，摘要应该等于 LLM 响应
				if summary != tt.llmResponse {
					t.Errorf("期望摘要 = %q, 实际 = %q", tt.llmResponse, summary)
				}
			}
		})
	}
}

// TestSummarizer_Summarize_EmptyContents 测试空内容处理。
func TestSummarizer_Summarize_EmptyContents(t *testing.T) {
	mockProvider := &mockChatProvider{
		response: "测试摘要",
	}

	summarizer := NewSummarizer(mockProvider, nil)

	_, err := summarizer.Summarize(context.Background(), []string{})
	if err == nil {
		t.Error("期望返回错误，但没有返回")
	}
}

// TestSummarizer_combineContents 测试内容合并。
func TestSummarizer_combineContents(t *testing.T) {
	summarizer := NewSummarizer(nil, nil)

	tests := []struct {
		name     string
		contents []string
		want     string
	}{
		{
			name:     "正常合并",
			contents: []string{"内容1", "内容2", "内容3"},
			want:     "内容1\n\n内容2\n\n内容3",
		},
		{
			name:     "过滤空内容",
			contents: []string{"内容1", "", "内容2", "   ", "内容3"},
			want:     "内容1\n\n内容2\n\n内容3",
		},
		{
			name:     "全部为空",
			contents: []string{"", "   ", "\n"},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizer.combineContents(tt.contents)
			if got != tt.want {
				t.Errorf("combineContents() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSummarizer_validateSummary 测试摘要质量验证。
func TestSummarizer_validateSummary(t *testing.T) {
	summarizer := NewSummarizer(nil, nil)

	tests := []struct {
		name    string
		summary string
		want    bool
	}{
		{
			name:    "长度合适",
			summary: "这是一段长度合适的摘要内容，用于测试质量验证逻辑。",
			want:    true,
		},
		{
			name:    "太短",
			summary: "短",
			want:    false,
		},
		{
			name:    "太长",
			summary: string(make([]byte, 300)),
			want:    false,
		},
		{
			name:    "纯空白",
			summary: "   \n\t   ",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizer.validateSummary(tt.summary)
			if got != tt.want {
				t.Errorf("validateSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSummarizer_fallbackSummary 测试降级策略。
func TestSummarizer_fallbackSummary(t *testing.T) {
	summarizer := NewSummarizer(nil, nil)

	tests := []struct {
		name    string
		content string
		wantLen int // 期望长度不超过此值
	}{
		{
			name:    "短内容",
			content: "这是一段短内容。",
			wantLen: 250,
		},
		{
			name:    "长内容",
			content: string(make([]byte, 500)),
			wantLen: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizer.fallbackSummary(tt.content)
			if len(got) > tt.wantLen {
				t.Errorf("fallbackSummary() 长度 = %d, 期望不超过 %d", len(got), tt.wantLen)
			}
			if len(got) == 0 {
				t.Error("fallbackSummary() 返回空字符串")
			}
		})
	}
}
