package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/logger"
	"github.com/panjf2000/ants/v2"
)

// Type defines the type of worker pool.
type Type string

const (
	// DefaultPool 默认通用池
	DefaultPool Type = "default"
	// HealthCheckPool 健康检查专用池
	HealthCheckPool Type = "health-check"
	// BackgroundPool 后台任务池（清理、监控等）
	BackgroundPool Type = "background"
	// CallbackPool 回调执行池
	CallbackPool Type = "callback"
	// TimeoutPool 超时中间件池
	TimeoutPool Type = "timeout"
)

// Config defines the configuration for the worker pool.
type Config struct {
	// Capacity 池容量（最大并发 goroutine 数）
	// 设置为 0 表示无限制（不推荐）
	Capacity int
	// ExpiryDuration goroutine 空闲过期时间
	ExpiryDuration time.Duration
	// PreAlloc 是否预分配内存（降低 GC，但增加初始内存占用）
	PreAlloc bool
	// Nonblocking 提交任务是否非阻塞（若池满则返回错误）
	Nonblocking bool
	// MaxBlockingTasks 当 Nonblocking=false 时，最大等待任务数（0 表示无限制）
	MaxBlockingTasks int
	// PanicHandler 恐慌处理函数
	PanicHandler func(interface{})
}

// DefaultPoolConfig 返回默认池配置
func DefaultPoolConfig() *Config {
	return &Config{
		Capacity:         1000,
		ExpiryDuration:   10 * time.Second,
		PreAlloc:         false,
		Nonblocking:      false,
		MaxBlockingTasks: 0,
	}
}

// HealthCheckPoolConfig 返回健康检查池配置
func HealthCheckPoolConfig() *Config {
	return &Config{
		Capacity:         100,
		ExpiryDuration:   30 * time.Second,
		PreAlloc:         true,
		Nonblocking:      true,
		MaxBlockingTasks: 10,
	}
}

// BackgroundPoolConfig 返回后台任务池配置
func BackgroundPoolConfig() *Config {
	return &Config{
		Capacity:         50,
		ExpiryDuration:   60 * time.Second,
		PreAlloc:         false,
		Nonblocking:      true,
		MaxBlockingTasks: 100,
	}
}

// CallbackPoolConfig 返回回调执行池配置
func CallbackPoolConfig() *Config {
	return &Config{
		Capacity:         200,
		ExpiryDuration:   5 * time.Second,
		PreAlloc:         false,
		Nonblocking:      false,
		MaxBlockingTasks: 1000,
	}
}

// TimeoutPoolConfig 返回超时中间件池配置
func TimeoutPoolConfig() *Config {
	return &Config{
		Capacity:         5000, // 高并发场景
		ExpiryDuration:   5 * time.Second,
		PreAlloc:         true, // 预分配提升性能
		Nonblocking:      true, // 超时不应阻塞
		MaxBlockingTasks: 1000,
	}
}

// Pool represents a worker pool.
type Pool struct {
	name     string
	typ      Type
	pool     *ants.Pool
	config   *Config
	stats    *poolStatsCounter
	closed   atomic.Bool
	closedMu sync.Mutex
}

// poolStatsCounter 内部统计计数器
type poolStatsCounter struct {
	SubmittedTasks  atomic.Int64
	CompletedTasks  atomic.Int64
	FailedTasks     atomic.Int64
	RejectedTasks   atomic.Int64
	PanicRecovered  atomic.Int64
	TotalWaitTimeNs atomic.Int64
}

// Stats contains statistics about the worker pool.
type Stats struct {
	SubmittedTasks  int64 // 已提交任务数
	CompletedTasks  int64 // 已完成任务数
	FailedTasks     int64 // 失败任务数
	RejectedTasks   int64 // 拒绝任务数
	PanicRecovered  int64 // 恢复的 panic 数
	TotalWaitTimeNs int64 // 总等待时间（纳秒）
}

// NewPool creates a new worker pool with the given configuration.
func NewPool(name string, typ Type, config *Config) (*Pool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	p := &Pool{
		name:   name,
		typ:    typ,
		config: config,
		stats:  &poolStatsCounter{},
	}

	opts := buildAntsOptions(name, config)
	pool, err := ants.NewPool(config.Capacity, opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 ants 池失败: %w", err)
	}
	p.pool = pool

	logger.Infow("Worker pool created",
		"name", name,
		"capacity", config.Capacity,
		"preAlloc", config.PreAlloc,
	)

	return p, nil
}

// buildAntsOptions 构建 ants 池选项
func buildAntsOptions(name string, config *Config) []ants.Option {
	opts := []ants.Option{
		ants.WithExpiryDuration(config.ExpiryDuration),
		ants.WithPreAlloc(config.PreAlloc),
		ants.WithNonblocking(config.Nonblocking),
		ants.WithMaxBlockingTasks(config.MaxBlockingTasks),
		// 		ants.WithLogger(logger.StandardLogger()), // 使用 zap logger - deprecated, ants v2 custom logger interface mismatch
	}

	if config.PanicHandler != nil {
		opts = append(opts, ants.WithPanicHandler(config.PanicHandler))
	} else {
		// 默认 panic 处理
		opts = append(opts, ants.WithPanicHandler(func(p interface{}) {
			logger.Errorw("Worker panic recovered",
				"pool", name,
				"panic", p,
			)
		}))
	}

	return opts
}

// Name 返回池名称
func (p *Pool) Name() string {
	return p.name
}

// Type 返回池类型
func (p *Pool) Type() Type {
	return p.typ
}

// Cap 返回池容量
func (p *Pool) Cap() int {
	return p.pool.Cap()
}

// Running 返回正在运行的 goroutine 数量
func (p *Pool) Running() int {
	return p.pool.Running()
}

// Free 返回可用 goroutine 数量
func (p *Pool) Free() int {
	return p.pool.Free()
}

// Waiting 返回等待执行的任务数量
func (p *Pool) Waiting() int {
	return p.pool.Waiting()
}

// Submit 提交任务到池中执行
func (p *Pool) Submit(task func()) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	startTime := time.Now()
	err := p.pool.Submit(func() {
		waitTime := time.Since(startTime)
		p.stats.TotalWaitTimeNs.Add(int64(waitTime))
		p.stats.SubmittedTasks.Add(1)

		defer func() {
			if r := recover(); r != nil {
				p.stats.PanicRecovered.Add(1)
				p.stats.FailedTasks.Add(1)
				// Re-panic to let ants PanicHandler handle it
				panic(r)
			}
			p.stats.CompletedTasks.Add(1)
		}()

		task()
	})
	if err != nil {
		if errors.Is(err, ants.ErrPoolOverload) {
			p.stats.RejectedTasks.Add(1)
			return ErrPoolOverload
		}
		p.stats.FailedTasks.Add(1)
		return err
	}

	return nil
}

// SubmitWithContext 提交带上下文的任务
// 如果上下文取消，任务可能不会执行（取决于排队状态）
func (p *Pool) SubmitWithContext(ctx context.Context, task func()) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 继续提交
	}

	return p.Submit(func() {
		select {
		case <-ctx.Done():
			// 任务开始前上下文已取消
			return
		default:
			task()
		}
	})
}

// Release 关闭池并释放资源
func (p *Pool) Release() {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()

	if p.closed.Load() {
		return
	}

	p.closed.Store(true)
	p.pool.Release()
	logger.Infow("Worker pool released", "name", p.name)
}

// ReleaseTimeout 带超时关闭池
// 等待任务完成，直到超时
func (p *Pool) ReleaseTimeout(timeout time.Duration) error {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()

	if p.closed.Load() {
		return nil
	}

	p.closed.Store(true)
	return p.pool.ReleaseTimeout(timeout)
}

// Tune 动态调整池容量
func (p *Pool) Tune(size int) {
	p.pool.Tune(size)
	p.config.Capacity = size
	logger.Infow("Worker pool tuned", "name", p.name, "new_capacity", size)
}

// GetStats 获取详细统计信息（原子快照）
func (p *Pool) GetStats() (submitted, completed, failed, rejected, panics int64) {
	return p.stats.SubmittedTasks.Load(),
		p.stats.CompletedTasks.Load(),
		p.stats.FailedTasks.Load(),
		p.stats.RejectedTasks.Load(),
		p.stats.PanicRecovered.Load()
}

// Stats 返回池统计信息快照
func (p *Pool) Stats() Stats {
	return Stats{
		SubmittedTasks:  p.stats.SubmittedTasks.Load(),
		CompletedTasks:  p.stats.CompletedTasks.Load(),
		FailedTasks:     p.stats.FailedTasks.Load(),
		RejectedTasks:   p.stats.RejectedTasks.Load(),
		PanicRecovered:  p.stats.PanicRecovered.Load(),
		TotalWaitTimeNs: p.stats.TotalWaitTimeNs.Load(),
	}
}
