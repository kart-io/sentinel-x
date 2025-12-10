package errors

import (
	"context"
	"fmt"
)

// NewError 创建一个新的 AgentError，带有指定的错误码和消息
func NewError(code ErrorCode, message string) *AgentError {
	return &AgentError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
		Stack:   captureStack(2),
	}
}

// NewErrorWithCause 创建一个新的 AgentError，带有指定的错误码、消息和原因
func NewErrorWithCause(code ErrorCode, message string, cause error) *AgentError {
	if cause == nil {
		return NewError(code, message)
	}
	return &AgentError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
		Cause:   cause,
		Stack:   captureStack(2),
	}
}

// NewErrorf 创建一个新的 AgentError，带有格式化消息
func NewErrorf(code ErrorCode, format string, args ...interface{}) *AgentError {
	return &AgentError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Context: make(map[string]interface{}),
		Stack:   captureStack(2),
	}
}

// ErrorWithRetry 为错误添加重试信息
func ErrorWithRetry(err error, attempt, maxAttempts int) *AgentError {
	if err == nil {
		return nil
	}

	if agentErr, ok := err.(*AgentError); ok {
		return agentErr.
			WithContext("retry_attempt", attempt).
			WithContext("max_attempts", maxAttempts)
	}

	return Wrap(err, CodeInternal, "操作失败").
		WithContext("retry_attempt", attempt).
		WithContext("max_attempts", maxAttempts)
}

// ErrorWithDuration 为错误添加持续时间信息
func ErrorWithDuration(err error, durationMs int64) *AgentError {
	if err == nil {
		return nil
	}

	if agentErr, ok := err.(*AgentError); ok {
		return agentErr.WithContext("duration_ms", durationMs)
	}

	return Wrap(err, CodeInternal, "操作失败").
		WithContext("duration_ms", durationMs)
}

// ErrorWithContext 为错误添加上下文信息
func ErrorWithContext(err error, key string, value interface{}) *AgentError {
	if err == nil {
		return nil
	}

	if agentErr, ok := err.(*AgentError); ok {
		return agentErr.WithContext(key, value)
	}

	return Wrap(err, CodeInternal, "操作失败").
		WithContext(key, value)
}

// IsContextCanceled 检查错误是否为上下文取消
func IsContextCanceled(err error) bool {
	return err == context.Canceled || IsCode(err, CodeAgentTimeout)
}

// IsContextTimeout 检查错误是否为上下文超时
func IsContextTimeout(err error) bool {
	return err == context.DeadlineExceeded || IsCode(err, CodeAgentTimeout)
}

// IsNotFound 检查错误是否为资源未找到
func IsNotFound(err error) bool {
	return IsCode(err, CodeNotFound) || IsCode(err, CodeToolNotFound)
}

// IsValidationError 检查错误是否为验证错误
func IsValidationError(err error) bool {
	return IsCode(err, CodeInvalidInput) || IsCode(err, CodeToolValidation)
}
