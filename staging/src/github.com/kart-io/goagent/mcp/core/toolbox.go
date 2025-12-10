package core

import (
	"context"
	"fmt"
)

// ToolRegistry 工具注册表接口
//
// ToolRegistry 管理工具的注册、注销和查询。
type ToolRegistry interface {
	// Register 注册工具
	Register(tool Tool) error

	// Unregister 注销工具
	Unregister(name string) error

	// Get 获取工具
	Get(name string) (Tool, bool)

	// List 列出所有工具
	List() []Tool

	// Has 检查工具是否存在
	Has(name string) bool
}

// ToolBox MCP 工具箱接口
//
// ToolBox 管理所有可用的工具，提供工具的注册、发现、执行等功能。
// ToolBox 组合了 ToolRegistry 的功能，并提供执行和权限管理能力。
type ToolBox interface {
	ToolRegistry

	// Execute 执行工具
	Execute(ctx context.Context, call *ToolCall) (*ToolCallResult, error)

	// ExecuteWithPermission 执行工具（带权限检查）
	ExecuteWithPermission(ctx context.Context, call *ToolCall, permission *ToolPermission) (*ToolCallResult, error)

	// ExecuteBatch 批量执行工具
	ExecuteBatch(ctx context.Context, calls []*ToolCall) ([]*ToolCallResult, error)

	// ListByCategory 按分类列出工具
	ListByCategory(category string) []Tool

	// Search 搜索工具（按名称或描述）
	Search(query string) []Tool

	// GetMetadata 获取工具元数据
	GetMetadata(name string) (*ToolMetadata, error)

	// ListMetadata 列出所有工具元数据
	ListMetadata() []*ToolMetadata

	// Validate 验证工具调用
	Validate(call *ToolCall) error

	// HasPermission 检查权限
	HasPermission(userID string, toolName string) (bool, error)

	// Statistics 获取工具使用统计
	Statistics() *ToolBoxStatistics
}


// ToolBoxStatistics 工具箱统计信息
type ToolBoxStatistics struct {
	// TotalTools 工具总数
	TotalTools int `json:"total_tools"`

	// TotalCalls 总调用次数
	TotalCalls int64 `json:"total_calls"`

	// SuccessfulCalls 成功调用次数
	SuccessfulCalls int64 `json:"successful_calls"`

	// FailedCalls 失败调用次数
	FailedCalls int64 `json:"failed_calls"`

	// AverageLatency 平均延迟（毫秒）
	AverageLatency float64 `json:"average_latency"`

	// ToolUsage 各工具使用次数
	ToolUsage map[string]int64 `json:"tool_usage"`

	// CategoryUsage 各分类使用次数
	CategoryUsage map[string]int64 `json:"category_usage"`
}

// ToolPermission 工具权限
type ToolPermission struct {
	// UserID 用户 ID
	UserID string `json:"user_id"`

	// ToolName 工具名称
	ToolName string `json:"tool_name"`

	// Allowed 是否允许
	Allowed bool `json:"allowed"`

	// MaxCallsPerMinute 每分钟最大调用次数（0 表示无限制）
	MaxCallsPerMinute int `json:"max_calls_per_minute"`

	// AllowDangerousOps 是否允许危险操作
	AllowDangerousOps bool `json:"allow_dangerous_ops"`

	// Reason 权限原因
	Reason string `json:"reason,omitempty"`
}

// ErrToolNotFound 工具未找到错误
//
//nolint:errname // Using traditional Err prefix for sentinel errors
type ErrToolNotFound struct {
	ToolName string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.ToolName)
}

// ErrToolAlreadyExists 工具已存在错误
//
//nolint:errname // Using traditional Err prefix for sentinel errors
type ErrToolAlreadyExists struct {
	ToolName string
}

func (e *ErrToolAlreadyExists) Error() string {
	return fmt.Sprintf("tool already exists: %s", e.ToolName)
}

// ErrInvalidInput 无效输入错误
//
//nolint:errname // Using traditional Err prefix for sentinel errors
type ErrInvalidInput struct {
	Field   string
	Message string
}

func (e *ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input for field '%s': %s", e.Field, e.Message)
}

// ErrPermissionDenied 权限拒绝错误
//
//nolint:errname // Using traditional Err prefix for sentinel errors
type ErrPermissionDenied struct {
	UserID   string
	ToolName string
	Reason   string
}

func (e *ErrPermissionDenied) Error() string {
	return fmt.Sprintf("permission denied for user '%s' to use tool '%s': %s", e.UserID, e.ToolName, e.Reason)
}

// ErrExecutionFailed 执行失败错误
//
//nolint:errname // Using traditional Err prefix for sentinel errors
type ErrExecutionFailed struct {
	ToolName string
	Reason   string
}

func (e *ErrExecutionFailed) Error() string {
	return fmt.Sprintf("tool execution failed for '%s': %s", e.ToolName, e.Reason)
}
