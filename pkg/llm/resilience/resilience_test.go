package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	assert.Equal(t, StateClosed, cb.State())

	// 成功调用应保持关闭状态
	err := cb.Execute(func() error { return nil })
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_OpenOnMaxFailures(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      3,
		Timeout:          1 * time.Second,
		HalfOpenMaxCalls: 1,
	}
	cb := NewCircuitBreaker(config)

	// 连续失败应打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error { return testErr })
		assert.Error(t, err)
	}

	assert.Equal(t, StateOpen, cb.State())

	// 熔断器打开后，应拒绝新请求
	err := cb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	}
	cb := NewCircuitBreaker(config)

	// 打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return testErr })
	}
	assert.Equal(t, StateOpen, cb.State())

	// 等待超时，应进入半开状态
	time.Sleep(150 * time.Millisecond)

	// 半开状态下的成功调用应关闭熔断器
	err := cb.Execute(func() error { return nil })
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	}
	cb := NewCircuitBreaker(config)

	// 打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return testErr })
	}

	// 等待超时进入半开状态
	time.Sleep(150 * time.Millisecond)

	// 半开状态下的失败应重新打开熔断器
	err := cb.Execute(func() error { return testErr })
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	// 打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 5; i++ {
		_ = cb.Execute(func() error { return testErr })
	}
	assert.Equal(t, StateOpen, cb.State())

	// 重置应关闭熔断器
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())

	// 重置后应能正常执行
	err := cb.Execute(func() error { return nil })
	require.NoError(t, err)
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	stats := cb.Stats()
	assert.Equal(t, "closed", stats["state"])
	assert.Equal(t, 0, stats["failures"])

	// 执行一些失败
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	stats = cb.Stats()
	assert.Equal(t, 1, stats["failures"])
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	callCount := 0
	err := RetryWithBackoff(ctx, config, func() error {
		callCount++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // 第一次就成功
}

func TestRetryWithBackoff_EventualSuccess(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		RetryableErrors: func(_ error) bool {
			return true
		},
	}

	callCount := 0
	testErr := errors.New("temporary error")

	err := RetryWithBackoff(ctx, config, func() error {
		callCount++
		if callCount < 3 {
			return testErr
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, callCount) // 第三次成功
}

func TestRetryWithBackoff_MaxAttemptsReached(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		RetryableErrors: func(_ error) bool {
			return true
		},
	}

	callCount := 0
	testErr := errors.New("persistent error")

	err := RetryWithBackoff(ctx, config, func() error {
		callCount++
		return testErr
	})

	require.Error(t, err)
	assert.Equal(t, 3, callCount) // 尝试了 3 次
	assert.Contains(t, err.Error(), "max retry attempts")
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		RetryableErrors: func(err error) bool {
			return err.Error() != "non-retryable"
		},
	}

	callCount := 0
	nonRetryableErr := errors.New("non-retryable")

	err := RetryWithBackoff(ctx, config, func() error {
		callCount++
		return nonRetryableErr
	})

	require.Error(t, err)
	assert.Equal(t, 1, callCount) // 只尝试了 1 次
	assert.Equal(t, nonRetryableErr, err)
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		RetryableErrors: func(_ error) bool {
			return true
		},
	}

	callCount := 0
	testErr := errors.New("test error")

	// 在第一次重试延迟期间取消上下文
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := RetryWithBackoff(ctx, config, func() error {
		callCount++
		return testErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.LessOrEqual(t, callCount, 2) // 应该在取消前调用 1-2 次
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		RetryableErrors: func(_ error) bool {
			return true
		},
	}

	testErr := errors.New("test error")
	start := time.Now()

	_ = RetryWithBackoff(ctx, config, func() error {
		return testErr
	})

	elapsed := time.Since(start)

	// 总延迟应该约为: 100ms + 200ms = 300ms
	// 允许一定的时间误差
	assert.Greater(t, elapsed, 250*time.Millisecond)
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestRetryWithCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	retryConfig := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		RetryableErrors: func(err error) bool {
			return !errors.Is(err, ErrCircuitBreakerOpen)
		},
	}

	cbConfig := &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	}
	cb := NewCircuitBreaker(cbConfig)

	testErr := errors.New("test error")

	// 第一次调用应该触发重试和熔断器打开
	err := RetryWithCircuitBreaker(ctx, retryConfig, cb, func() error {
		return testErr
	})
	require.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())

	// 熔断器打开后，应立即返回错误
	err = RetryWithCircuitBreaker(ctx, retryConfig, cb, func() error {
		return testErr
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
}

func TestDefaultConfigs(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	assert.Equal(t, 3, retryConfig.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, retryConfig.InitialDelay)
	assert.Equal(t, 10*time.Second, retryConfig.MaxDelay)
	assert.Equal(t, 2.0, retryConfig.Multiplier)

	cbConfig := DefaultCircuitBreakerConfig()
	assert.Equal(t, 5, cbConfig.MaxFailures)
	assert.Equal(t, 60*time.Second, cbConfig.Timeout)
	assert.Equal(t, 1, cbConfig.HalfOpenMaxCalls)
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}
}

func BenchmarkRetryWithBackoff_NoRetry(b *testing.B) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RetryWithBackoff(ctx, config, func() error {
			return nil
		})
	}
}
