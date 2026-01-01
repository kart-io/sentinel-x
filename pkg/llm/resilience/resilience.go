// Package resilience 提供 LLM 调用的韧性模式：重试、熔断器、超时控制。
package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/logger"
)

// RetryConfig 重试配置。
type RetryConfig struct {
	// MaxAttempts 最大尝试次数（包括首次调用）。
	MaxAttempts int
	// InitialDelay 初始延迟时间。
	InitialDelay time.Duration
	// MaxDelay 最大延迟时间。
	MaxDelay time.Duration
	// Multiplier 延迟倍增因子（指数退避）。
	Multiplier float64
	// RetryableErrors 可重试的错误判断函数。
	RetryableErrors func(error) bool
}

// DefaultRetryConfig 返回默认重试配置。
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		RetryableErrors: func(err error) bool {
			// 默认所有错误都可重试
			return true
		},
	}
}

// CircuitBreakerConfig 熔断器配置。
type CircuitBreakerConfig struct {
	// MaxFailures 触发熔断的最大失败次数。
	MaxFailures int
	// Timeout 熔断器打开后的超时时间。
	Timeout time.Duration
	// HalfOpenMaxCalls 半开状态允许的最大调用次数。
	HalfOpenMaxCalls int
}

// DefaultCircuitBreakerConfig 返回默认熔断器配置。
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:      5,
		Timeout:          60 * time.Second,
		HalfOpenMaxCalls: 1,
	}
}

// CircuitBreakerState 熔断器状态。
type CircuitBreakerState int

const (
	// StateClosed 熔断器关闭，正常工作。
	StateClosed CircuitBreakerState = iota
	// StateOpen 熔断器打开，拒绝所有请求。
	StateOpen
	// StateHalfOpen 熔断器半开，允许部分请求探测。
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker 熔断器实现。
type CircuitBreaker struct {
	config *CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitBreakerState
	failures          int
	lastFailureTime   time.Time
	halfOpenCalls     int
	halfOpenSuccesses int
}

// NewCircuitBreaker 创建熔断器。
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// ErrCircuitBreakerOpen 熔断器打开错误。
var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

// Execute 通过熔断器执行函数。
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// 检查是否允许执行
	if err := cb.beforeCall(); err != nil {
		return err
	}

	// 执行函数
	err := fn()

	// 记录结果
	cb.afterCall(err)

	return err
}

// beforeCall 调用前检查熔断器状态。
func (cb *CircuitBreaker) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// 熔断器关闭，允许调用
		return nil

	case StateOpen:
		// 检查是否应该进入半开状态
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			logger.Infow("circuit breaker transitioning to half-open")
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 0
			cb.halfOpenSuccesses = 0
			return nil
		}
		// 熔断器仍然打开
		return ErrCircuitBreakerOpen

	case StateHalfOpen:
		// 半开状态，检查是否还能接受调用
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			return ErrCircuitBreakerOpen
		}
		cb.halfOpenCalls++
		return nil

	default:
		return ErrCircuitBreakerOpen
	}
}

// afterCall 调用后记录结果并更新状态。
func (cb *CircuitBreaker) afterCall(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onSuccess 成功调用的处理。
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		// 关闭状态下成功，重置失败计数
		cb.failures = 0

	case StateHalfOpen:
		// 半开状态下成功
		cb.halfOpenSuccesses++
		// 如果所有半开状态的调用都成功，转为关闭状态
		if cb.halfOpenSuccesses >= cb.halfOpenCalls {
			logger.Infow("circuit breaker transitioning to closed")
			cb.state = StateClosed
			cb.failures = 0
		}
	}
}

// onFailure 失败调用的处理。
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// 关闭状态下失败次数达到阈值，打开熔断器
		if cb.failures >= cb.config.MaxFailures {
			logger.Warnw("circuit breaker opening",
				"failures", cb.failures,
				"max_failures", cb.config.MaxFailures,
			)
			cb.state = StateOpen
		}

	case StateHalfOpen:
		// 半开状态下失败，立即打开熔断器
		logger.Warnw("circuit breaker re-opening after half-open failure")
		cb.state = StateOpen
	}
}

// State 获取当前状态。
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats 获取熔断器统计信息。
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":              cb.state.String(),
		"failures":           cb.failures,
		"last_failure_time":  cb.lastFailureTime,
		"half_open_calls":    cb.halfOpenCalls,
		"half_open_successes": cb.halfOpenSuccesses,
	}
}

// Reset 重置熔断器状态。
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenCalls = 0
	cb.halfOpenSuccesses = 0
}

// RetryWithBackoff 使用指数退避重试函数。
func RetryWithBackoff(ctx context.Context, config *RetryConfig, fn func() error) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// 执行函数
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if !config.RetryableErrors(err) {
			logger.Debugw("error is not retryable", "error", err.Error())
			return err
		}

		// 最后一次尝试失败，不再重试
		if attempt >= config.MaxAttempts {
			logger.Warnw("max retry attempts reached",
				"attempts", attempt,
				"error", err.Error(),
			)
			return fmt.Errorf("max retry attempts (%d) reached: %w", config.MaxAttempts, lastErr)
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 等待后重试
		logger.Debugw("retrying after delay",
			"attempt", attempt,
			"delay", delay,
			"error", err.Error(),
		)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}

		// 计算下次延迟（指数退避）
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return lastErr
}

// RetryWithCircuitBreaker 结合重试和熔断器执行函数。
func RetryWithCircuitBreaker(
	ctx context.Context,
	retryConfig *RetryConfig,
	cb *CircuitBreaker,
	fn func() error,
) error {
	return RetryWithBackoff(ctx, retryConfig, func() error {
		return cb.Execute(fn)
	})
}
