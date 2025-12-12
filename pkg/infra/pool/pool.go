// Package pool 提供基于 ants 的统一 goroutine 池管理能力。
//
// 本包封装了 github.com/panjf2000/ants/v2，提供：
//   - 统一的池工厂，支持多池配置
//   - 安全的任务提交（自动 panic 恢复）
//   - 完善的生命周期管理
//   - 可观测性支持（指标收集）
//
// 设计原则：
//   - 禁止直接使用 go func()，所有并发任务必须通过池提交
//   - 每个业务场景使用独立的池，避免互相影响
//   - 池容量根据业务特性合理配置
//
// 基本用法：
//
//	// 初始化全局池管理器
//	pool.InitGlobal()
//	defer pool.CloseGlobal()
//
//	// 使用默认池提交任务
//	pool.Submit(func() {
//	    // 业务逻辑
//	})
//
//	// 使用指定池提交任务
//	pool.SubmitTo("health-check", func() {
//	    // 健康检查逻辑
//	})
package pool

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/logger"
	"github.com/panjf2000/ants/v2"
)

// PoolType 定义池类型
type PoolType string

const (
	// DefaultPool 默认通用池
	DefaultPool PoolType = "default"
	// HealthCheckPool 健康检查专用池
	HealthCheckPool PoolType = "health-check"
	// BackgroundPool 后台任务池（清理、监控等）
	BackgroundPool PoolType = "background"
	// CallbackPool 回调执行池
	CallbackPool PoolType = "callback"
	// TimeoutPool 超时中间件池
	TimeoutPool PoolType = "timeout"
)

// PoolConfig 池配置
type PoolConfig struct {
	// Capacity 池容量（最大并发 goroutine 数）
	// 设置为 0 表示无限制（不推荐）
	Capacity int

	// ExpiryDuration 空闲 worker 过期时间
	// 超过此时间未使用的 worker 会被回收
	// 默认: 10s
	ExpiryDuration time.Duration

	// PreAlloc 是否预分配 worker 队列
	// 适用于大容量池，可减少运行时内存分配
	// 默认: false
	PreAlloc bool

	// Nonblocking 是否非阻塞模式
	// true: 池满时立即返回 ErrPoolOverload
	// false: 池满时阻塞等待
	// 默认: false（阻塞模式）
	Nonblocking bool

	// MaxBlockingTasks 最大阻塞等待任务数（仅阻塞模式有效）
	// 超过此数量时返回 ErrPoolOverload
	// 设置为 0 表示无限制
	// 默认: 0
	MaxBlockingTasks int

	// DisablePurge 是否禁用自动清理过期 worker
	// 默认: false
	DisablePurge bool

	// PanicHandler 自定义 panic 处理器
	// 如果为 nil，使用默认的日志记录处理器
	PanicHandler func(interface{})
}

// DefaultPoolConfig 返回默认池配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		Capacity:         1000,
		ExpiryDuration:   10 * time.Second,
		PreAlloc:         false,
		Nonblocking:      false,
		MaxBlockingTasks: 0,
		DisablePurge:     false,
		PanicHandler:     nil,
	}
}

// HealthCheckPoolConfig 返回健康检查池配置
func HealthCheckPoolConfig() *PoolConfig {
	return &PoolConfig{
		Capacity:         100,
		ExpiryDuration:   30 * time.Second,
		PreAlloc:         true,
		Nonblocking:      false,
		MaxBlockingTasks: 50,
		DisablePurge:     false,
		PanicHandler:     nil,
	}
}

// BackgroundPoolConfig 返回后台任务池配置
func BackgroundPoolConfig() *PoolConfig {
	return &PoolConfig{
		Capacity:         50,
		ExpiryDuration:   60 * time.Second,
		PreAlloc:         false,
		Nonblocking:      false,
		MaxBlockingTasks: 100,
		DisablePurge:     false,
		PanicHandler:     nil,
	}
}

// CallbackPoolConfig 返回回调执行池配置
func CallbackPoolConfig() *PoolConfig {
	return &PoolConfig{
		Capacity:         200,
		ExpiryDuration:   5 * time.Second,
		PreAlloc:         false,
		Nonblocking:      true, // 回调使用非阻塞模式，避免阻塞主流程
		MaxBlockingTasks: 0,
		DisablePurge:     false,
		PanicHandler:     nil,
	}
}

// TimeoutPoolConfig 返回超时中间件池配置
func TimeoutPoolConfig() *PoolConfig {
	return &PoolConfig{
		Capacity:         5000, // 高并发场景
		ExpiryDuration:   5 * time.Second,
		PreAlloc:         true, // 预分配提升性能
		Nonblocking:      false,
		MaxBlockingTasks: 1000,
		DisablePurge:     false,
		PanicHandler:     nil,
	}
}

// Pool 封装 ants.Pool，提供安全的任务提交能力
type Pool struct {
	name     string
	pool     *ants.Pool
	config   *PoolConfig
	stats    *poolStatsCounter
	closed   atomic.Bool
	closedMu sync.Mutex
}

// poolStatsCounter 内部使用的原子计数器
type poolStatsCounter struct {
	SubmittedTasks  atomic.Int64
	CompletedTasks  atomic.Int64
	FailedTasks     atomic.Int64
	RejectedTasks   atomic.Int64
	PanicRecovered  atomic.Int64
	TotalWaitTimeNs atomic.Int64
}

// PoolStats 池统计信息（快照）
type PoolStats struct {
	SubmittedTasks  int64 // 已提交任务数
	CompletedTasks  int64 // 已完成任务数
	FailedTasks     int64 // 失败任务数
	RejectedTasks   int64 // 被拒绝任务数
	PanicRecovered  int64 // panic 恢复次数
	TotalWaitTimeNs int64 // 总等待时间（纳秒）
}

// NewPool 创建新的池实例
func NewPool(name string, config *PoolConfig) (*Pool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	// 构建 ants 选项
	opts := buildAntsOptions(name, config)

	antsPool, err := ants.NewPool(config.Capacity, opts...)
	if err != nil {
		return nil, fmt.Errorf("创建池 '%s' 失败: %w", name, err)
	}

	return &Pool{
		name:   name,
		pool:   antsPool,
		config: config,
		stats:  &poolStatsCounter{},
	}, nil
}

// buildAntsOptions 构建 ants 池选项
func buildAntsOptions(name string, config *PoolConfig) []ants.Option {
	opts := []ants.Option{
		ants.WithExpiryDuration(config.ExpiryDuration),
		ants.WithPreAlloc(config.PreAlloc),
		ants.WithNonblocking(config.Nonblocking),
		ants.WithMaxBlockingTasks(config.MaxBlockingTasks),
		ants.WithDisablePurge(config.DisablePurge),
	}

	// 设置 panic 处理器
	panicHandler := config.PanicHandler
	if panicHandler == nil {
		panicHandler = defaultPanicHandler(name)
	}
	opts = append(opts, ants.WithPanicHandler(panicHandler))

	return opts
}

// defaultPanicHandler 默认 panic 处理器
func defaultPanicHandler(poolName string) func(interface{}) {
	return func(r interface{}) {
		stack := debug.Stack()
		logger.Errorw("goroutine panic recovered in pool",
			"pool", poolName,
			"panic", fmt.Sprintf("%v", r),
			"stack", string(stack),
		)
	}
}

// Name 返回池名称
func (p *Pool) Name() string {
	return p.name
}

// Submit 提交任务到池
func (p *Pool) Submit(task func()) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	p.stats.SubmittedTasks.Add(1)
	start := time.Now()

	// 包装任务以收集统计信息
	wrappedTask := func() {
		defer func() {
			p.stats.CompletedTasks.Add(1)
			if r := recover(); r != nil {
				p.stats.PanicRecovered.Add(1)
				// panic 会被 ants 的 PanicHandler 处理
				panic(r)
			}
		}()
		task()
	}

	err := p.pool.Submit(wrappedTask)
	p.stats.TotalWaitTimeNs.Add(time.Since(start).Nanoseconds())

	if err != nil {
		p.stats.RejectedTasks.Add(1)
		return fmt.Errorf("提交任务到池 '%s' 失败: %w", p.name, err)
	}

	return nil
}

// SubmitWithContext 提交带上下文的任务
func (p *Pool) SubmitWithContext(ctx context.Context, task func()) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return p.Submit(func() {
		// 再次检查上下文
		select {
		case <-ctx.Done():
			return
		default:
			task()
		}
	})
}

// SubmitWithTimeout 提交带超时的任务
func (p *Pool) SubmitWithTimeout(timeout time.Duration, task func()) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.SubmitWithContext(ctx, task)
}

// Running 返回当前运行中的 worker 数
func (p *Pool) Running() int {
	return p.pool.Running()
}

// Free 返回空闲 worker 数
func (p *Pool) Free() int {
	return p.pool.Free()
}

// Cap 返回池容量
func (p *Pool) Cap() int {
	return p.pool.Cap()
}

// Waiting 返回等待中的任务数
func (p *Pool) Waiting() int {
	return p.pool.Waiting()
}

// IsClosed 检查池是否已关闭
func (p *Pool) IsClosed() bool {
	return p.closed.Load()
}

// Tune 动态调整池容量
func (p *Pool) Tune(size int) {
	p.pool.Tune(size)
}

// Stats 返回池统计信息快照
func (p *Pool) Stats() PoolStats {
	return PoolStats{
		SubmittedTasks:  p.stats.SubmittedTasks.Load(),
		CompletedTasks:  p.stats.CompletedTasks.Load(),
		FailedTasks:     p.stats.FailedTasks.Load(),
		RejectedTasks:   p.stats.RejectedTasks.Load(),
		PanicRecovered:  p.stats.PanicRecovered.Load(),
		TotalWaitTimeNs: p.stats.TotalWaitTimeNs.Load(),
	}
}

// GetStats 返回池统计信息的值
func (p *Pool) GetStats() (submitted, completed, failed, rejected, panics int64) {
	return p.stats.SubmittedTasks.Load(),
		p.stats.CompletedTasks.Load(),
		p.stats.FailedTasks.Load(),
		p.stats.RejectedTasks.Load(),
		p.stats.PanicRecovered.Load()
}

// Release 释放池资源
func (p *Pool) Release() {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()

	if p.closed.Load() {
		return
	}

	p.closed.Store(true)
	p.pool.Release()
}

// ReleaseTimeout 带超时的释放
func (p *Pool) ReleaseTimeout(timeout time.Duration) error {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()

	if p.closed.Load() {
		return nil
	}

	p.closed.Store(true)
	return p.pool.ReleaseTimeout(timeout)
}

// Reboot 重启池（必须在 Release 之后调用）
func (p *Pool) Reboot() {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()

	if !p.closed.Load() {
		return
	}

	p.pool.Reboot()
	p.closed.Store(false)
}
