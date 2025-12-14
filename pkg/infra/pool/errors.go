// Package pool provides resource pooling.
package pool

import "errors"

// 池相关错误定义
var (
	// ErrPoolClosed 池已关闭
	ErrPoolClosed = errors.New("池已关闭")

	// ErrPoolNotFound 池不存在
	ErrPoolNotFound = errors.New("池不存在")

	// ErrPoolAlreadyExists 池已存在
	ErrPoolAlreadyExists = errors.New("池已存在")

	// ErrInvalidPoolConfig 无效的池配置
	ErrInvalidPoolConfig = errors.New("无效的池配置")

	// ErrManagerNotInitialized 管理器未初始化
	ErrManagerNotInitialized = errors.New("池管理器未初始化")

	// ErrPoolOverload 池已满
	ErrPoolOverload = errors.New("池已满")
)
