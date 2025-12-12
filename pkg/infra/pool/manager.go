package pool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Manager 池管理器，管理多个命名池
type Manager struct {
	mu     sync.RWMutex
	pools  map[string]*Pool
	closed atomic.Bool
}

// NewManager 创建新的池管理器
func NewManager() *Manager {
	return &Manager{
		pools: make(map[string]*Pool),
	}
}

// Register 注册新池
func (m *Manager) Register(name string, config *PoolConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed.Load() {
		return ErrPoolClosed
	}

	if _, exists := m.pools[name]; exists {
		return fmt.Errorf("%w: %s", ErrPoolAlreadyExists, name)
	}

	pool, err := NewPool(name, config)
	if err != nil {
		return err
	}

	m.pools[name] = pool
	return nil
}

// RegisterWithType 使用预定义类型注册池
func (m *Manager) RegisterWithType(poolType PoolType, config *PoolConfig) error {
	return m.Register(string(poolType), config)
}

// Get 获取指定名称的池
func (m *Manager) Get(name string) (*Pool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed.Load() {
		return nil, ErrPoolClosed
	}

	pool, exists := m.pools[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrPoolNotFound, name)
	}

	return pool, nil
}

// GetByType 获取预定义类型的池
func (m *Manager) GetByType(poolType PoolType) (*Pool, error) {
	return m.Get(string(poolType))
}

// MustGet 获取池，不存在时 panic
func (m *Manager) MustGet(name string) *Pool {
	pool, err := m.Get(name)
	if err != nil {
		panic(fmt.Sprintf("获取池 '%s' 失败: %v", name, err))
	}
	return pool
}

// Submit 提交任务到指定池
func (m *Manager) Submit(poolName string, task func()) error {
	pool, err := m.Get(poolName)
	if err != nil {
		return err
	}
	return pool.Submit(task)
}

// SubmitToDefault 提交任务到默认池
func (m *Manager) SubmitToDefault(task func()) error {
	return m.Submit(string(DefaultPool), task)
}

// SubmitWithContext 提交带上下文的任务到指定池
func (m *Manager) SubmitWithContext(ctx context.Context, poolName string, task func()) error {
	pool, err := m.Get(poolName)
	if err != nil {
		return err
	}
	return pool.SubmitWithContext(ctx, task)
}

// List 返回所有已注册的池名称
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.pools))
	for name := range m.pools {
		names = append(names, name)
	}
	return names
}

// Stats 返回所有池的统计信息
func (m *Manager) Stats() map[string]PoolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]PoolInfo, len(m.pools))
	for name, pool := range m.pools {
		submitted, completed, failed, rejected, panics := pool.GetStats()
		stats[name] = PoolInfo{
			Name:           name,
			Running:        pool.Running(),
			Free:           pool.Free(),
			Capacity:       pool.Cap(),
			Waiting:        pool.Waiting(),
			SubmittedTasks: submitted,
			CompletedTasks: completed,
			FailedTasks:    failed,
			RejectedTasks:  rejected,
			PanicRecovered: panics,
		}
	}
	return stats
}

// PoolInfo 池信息
type PoolInfo struct {
	Name           string
	Running        int
	Free           int
	Capacity       int
	Waiting        int
	SubmittedTasks int64
	CompletedTasks int64
	FailedTasks    int64
	RejectedTasks  int64
	PanicRecovered int64
}

// Tune 动态调整指定池的容量
func (m *Manager) Tune(name string, size int) error {
	pool, err := m.Get(name)
	if err != nil {
		return err
	}
	pool.Tune(size)
	return nil
}

// Release 释放指定池
func (m *Manager) Release(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, exists := m.pools[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrPoolNotFound, name)
	}

	pool.Release()
	delete(m.pools, name)
	return nil
}

// ReleaseAll 释放所有池
func (m *Manager) ReleaseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed.Store(true)
	for _, pool := range m.pools {
		pool.Release()
	}
	m.pools = make(map[string]*Pool)
}

// ReleaseAllTimeout 带超时释放所有池
func (m *Manager) ReleaseAllTimeout(timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed.Store(true)
	var firstErr error

	for name, pool := range m.pools {
		if err := pool.ReleaseTimeout(timeout); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("释放池 '%s' 超时: %w", name, err)
		}
	}

	m.pools = make(map[string]*Pool)
	return firstErr
}

// Close 关闭管理器（等同于 ReleaseAll）
func (m *Manager) Close() error {
	m.ReleaseAll()
	return nil
}

// IsClosed 检查管理器是否已关闭
func (m *Manager) IsClosed() bool {
	return m.closed.Load()
}
