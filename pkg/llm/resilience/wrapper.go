// Package resilience 提供 LLM 调用的韧性包装器。
package resilience

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// ResilientEmbeddingProvider 带韧性功能的 Embedding Provider 包装器。
type ResilientEmbeddingProvider struct {
	provider llm.EmbeddingProvider
	retry    *RetryConfig
	cb       *CircuitBreaker
}

// NewResilientEmbeddingProvider 创建带韧性功能的 Embedding Provider。
func NewResilientEmbeddingProvider(
	provider llm.EmbeddingProvider,
	retryConfig *RetryConfig,
	cbConfig *CircuitBreakerConfig,
) *ResilientEmbeddingProvider {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}
	if cbConfig == nil {
		cbConfig = DefaultCircuitBreakerConfig()
	}

	// 设置默认的可重试错误判断
	if retryConfig.RetryableErrors == nil {
		retryConfig.RetryableErrors = IsRetryableError
	}

	return &ResilientEmbeddingProvider{
		provider: provider,
		retry:    retryConfig,
		cb:       NewCircuitBreaker(cbConfig),
	}
}

// Embed 为多个文本生成向量嵌入（带重试和熔断）。
func (r *ResilientEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	var result [][]float32
	var err error

	err = RetryWithCircuitBreaker(ctx, r.retry, r.cb, func() error {
		result, err = r.provider.Embed(ctx, texts)
		return err
	})

	return result, err
}

// EmbedSingle 为单个文本生成向量嵌入（带重试和熔断）。
func (r *ResilientEmbeddingProvider) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	var result []float32
	var err error

	err = RetryWithCircuitBreaker(ctx, r.retry, r.cb, func() error {
		result, err = r.provider.EmbedSingle(ctx, text)
		return err
	})

	return result, err
}

// Name 返回供应商名称。
func (r *ResilientEmbeddingProvider) Name() string {
	return r.provider.Name() + "-resilient"
}

// CircuitBreaker 获取熔断器实例（用于监控）。
func (r *ResilientEmbeddingProvider) CircuitBreaker() *CircuitBreaker {
	return r.cb
}

// ResilientChatProvider 带韧性功能的 Chat Provider 包装器。
type ResilientChatProvider struct {
	provider llm.ChatProvider
	retry    *RetryConfig
	cb       *CircuitBreaker
}

// NewResilientChatProvider 创建带韧性功能的 Chat Provider。
func NewResilientChatProvider(
	provider llm.ChatProvider,
	retryConfig *RetryConfig,
	cbConfig *CircuitBreakerConfig,
) *ResilientChatProvider {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}
	if cbConfig == nil {
		cbConfig = DefaultCircuitBreakerConfig()
	}

	// 设置默认的可重试错误判断
	if retryConfig.RetryableErrors == nil {
		retryConfig.RetryableErrors = IsRetryableError
	}

	return &ResilientChatProvider{
		provider: provider,
		retry:    retryConfig,
		cb:       NewCircuitBreaker(cbConfig),
	}
}

// Chat 进行多轮对话（带重试和熔断）。
func (r *ResilientChatProvider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	var result string
	var err error

	err = RetryWithCircuitBreaker(ctx, r.retry, r.cb, func() error {
		result, err = r.provider.Chat(ctx, messages)
		return err
	})

	return result, err
}

// Generate 根据提示生成文本（带重试和熔断）。
func (r *ResilientChatProvider) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	var result string
	var err error

	err = RetryWithCircuitBreaker(ctx, r.retry, r.cb, func() error {
		result, err = r.provider.Generate(ctx, prompt, systemPrompt)
		return err
	})

	return result, err
}

// Name 返回供应商名称。
func (r *ResilientChatProvider) Name() string {
	return r.provider.Name() + "-resilient"
}

// CircuitBreaker 获取熔断器实例（用于监控）。
func (r *ResilientChatProvider) CircuitBreaker() *CircuitBreaker {
	return r.cb
}

// IsRetryableError 判断错误是否可重试。
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 熔断器打开错误不可重试
	if errors.Is(err, ErrCircuitBreakerOpen) {
		return false
	}

	// 上下文相关错误不可重试
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// 网络相关错误可重试
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			logger.Debugw("network timeout, retryable", "error", err.Error())
			return true
		}
		// 注意: Temporary() 已废弃,但仍保留以兼容旧版本 Go
		// 大多数临时错误实际上是超时错误,已在上面处理
	}

	// DNS 错误可重试
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		logger.Debugw("DNS error, retryable", "error", err.Error())
		return true
	}

	// 连接错误可重试
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		logger.Debugw("network operation error, retryable", "error", err.Error())
		return true
	}

	// HTTP 5xx 错误可重试
	errMsg := err.Error()
	if strings.Contains(errMsg, "status code 5") ||
		strings.Contains(errMsg, "状态码 5") ||
		strings.Contains(errMsg, "服务器错误") {
		logger.Debugw("server error, retryable", "error", errMsg)
		return true
	}

	// HTTP 429 (Too Many Requests) 可重试
	if strings.Contains(errMsg, "status code 429") ||
		strings.Contains(errMsg, "状态码 429") ||
		strings.Contains(errMsg, "rate limit") {
		logger.Debugw("rate limit error, retryable", "error", errMsg)
		return true
	}

	// HTTP 408 (Request Timeout) 可重试
	if strings.Contains(errMsg, "status code 408") ||
		strings.Contains(errMsg, "状态码 408") {
		logger.Debugw("request timeout, retryable", "error", errMsg)
		return true
	}

	// HTTP 503 (Service Unavailable) 可重试
	if strings.Contains(errMsg, "status code 503") ||
		strings.Contains(errMsg, "状态码 503") ||
		strings.Contains(errMsg, "service unavailable") {
		logger.Debugw("service unavailable, retryable", "error", errMsg)
		return true
	}

	// EOF 错误可重试
	if errors.Is(err, http.ErrServerClosed) ||
		strings.Contains(errMsg, "EOF") ||
		strings.Contains(errMsg, "connection reset") {
		logger.Debugw("connection error, retryable", "error", errMsg)
		return true
	}

	// 默认不重试
	logger.Debugw("error not retryable", "error", errMsg)
	return false
}

// Stats 获取韧性统计信息。
type Stats struct {
	CircuitBreakerState    string
	CircuitBreakerFailures int
	CircuitBreakerStats    map[string]interface{}
}

// GetEmbeddingProviderStats 获取 Embedding Provider 韧性统计。
func GetEmbeddingProviderStats(provider llm.EmbeddingProvider) *Stats {
	if rp, ok := provider.(*ResilientEmbeddingProvider); ok {
		cbStats := rp.cb.Stats()
		return &Stats{
			CircuitBreakerState:    cbStats["state"].(string),
			CircuitBreakerFailures: cbStats["failures"].(int),
			CircuitBreakerStats:    cbStats,
		}
	}
	return nil
}

// GetChatProviderStats 获取 Chat Provider 韧性统计。
func GetChatProviderStats(provider llm.ChatProvider) *Stats {
	if rp, ok := provider.(*ResilientChatProvider); ok {
		cbStats := rp.cb.Stats()
		return &Stats{
			CircuitBreakerState:    cbStats["state"].(string),
			CircuitBreakerFailures: cbStats["failures"].(int),
			CircuitBreakerStats:    cbStats,
		}
	}
	return nil
}
