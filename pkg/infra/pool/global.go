package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/logger"
)

// 全局池管理器
var (
	globalManager            *Manager
	globalManagerMu          sync.Mutex
	globalManagerInitialized uint32
)

// InitGlobal 初始化全局池管理器
// 自动注册常用池类型
func InitGlobal() error {
	return InitGlobalWithConfig(nil)
}

// GlobalConfig 全局池配置
type GlobalConfig struct {
	// DefaultPool 默认池配置
	DefaultPool *Config
	// HealthCheckPool 健康检查池配置
	HealthCheckPool *Config
	// BackgroundPool 后台任务池配置
	BackgroundPool *Config
	// CallbackPool 回调执行池配置
	CallbackPool *Config
	// TimeoutPool 超时中间件池配置
	TimeoutPool *Config
	// CustomPools 自定义池配置
	CustomPools map[string]*Config
}

// DefaultGlobalConfig 返回默认全局配置
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		DefaultPool:     DefaultPoolConfig(),
		HealthCheckPool: HealthCheckPoolConfig(),
		BackgroundPool:  BackgroundPoolConfig(),
		CallbackPool:    CallbackPoolConfig(),
		TimeoutPool:     TimeoutPoolConfig(),
		CustomPools:     make(map[string]*Config),
	}
}

// InitGlobalWithConfig 使用自定义配置初始化全局池管理器
func InitGlobalWithConfig(config *GlobalConfig) error {
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	if atomic.LoadUint32(&globalManagerInitialized) == 1 {
		return nil // 已初始化
	}

	if config == nil {
		config = DefaultGlobalConfig()
	}

	manager := NewManager()

	// 注册标准池
	pools := map[Type]*Config{
		DefaultPool:     config.DefaultPool,
		HealthCheckPool: config.HealthCheckPool,
		BackgroundPool:  config.BackgroundPool,
		CallbackPool:    config.CallbackPool,
		TimeoutPool:     config.TimeoutPool,
	}

	for poolType, poolConfig := range pools {
		if poolConfig == nil {
			continue
		}
		if err := manager.RegisterWithType(poolType, poolConfig); err != nil {
			// 清理已注册的池
			manager.ReleaseAll()
			return err
		}
	}

	// 注册自定义池
	for name, poolConfig := range config.CustomPools {
		// Custom pools default to DefaultPool type for now if not specified,
		// but since Register required a type, we use DefaultPool.
		if err := manager.Register(name, DefaultPool, poolConfig); err != nil {
			manager.ReleaseAll()
			return err
		}
	}

	globalManager = manager
	atomic.StoreUint32(&globalManagerInitialized, 1)

	logger.Infow("全局池管理器初始化完成",
		"pools", manager.List(),
	)

	return nil
}

// GetGlobal 获取全局池管理器
func GetGlobal() *Manager {
	if atomic.LoadUint32(&globalManagerInitialized) == 0 {
		// 自动初始化
		if err := InitGlobal(); err != nil {
			logger.Errorw("自动初始化全局池管理器失败", "error", err)
			return nil
		}
	}
	return globalManager
}

// MustGetGlobal 获取全局池管理器，未初始化时 panic
func MustGetGlobal() *Manager {
	mgr := GetGlobal()
	if mgr == nil {
		panic(ErrManagerNotInitialized)
	}
	return mgr
}

// CloseGlobal 关闭全局池管理器
func CloseGlobal() error {
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	if atomic.LoadUint32(&globalManagerInitialized) == 0 {
		return nil
	}

	if globalManager != nil {
		globalManager.ReleaseAll()
		globalManager = nil
	}
	atomic.StoreUint32(&globalManagerInitialized, 0)

	logger.Infow("全局池管理器已关闭")
	return nil
}

// CloseGlobalTimeout 带超时关闭全局池管理器
func CloseGlobalTimeout(timeout time.Duration) error {
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	if atomic.LoadUint32(&globalManagerInitialized) == 0 {
		return nil
	}

	var err error
	if globalManager != nil {
		err = globalManager.ReleaseAllTimeout(timeout)
		globalManager = nil
	}
	atomic.StoreUint32(&globalManagerInitialized, 0)

	logger.Infow("全局池管理器已关闭", "timeout", timeout)
	return err
}

// ResetGlobal 重置全局池管理器（仅用于测试）
func ResetGlobal() {
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	if globalManager != nil {
		globalManager.ReleaseAll()
		globalManager = nil
	}
	atomic.StoreUint32(&globalManagerInitialized, 0)
}

// ============================================================================
// 便捷函数 - 使用全局池管理器
// ============================================================================

// Submit 提交任务到默认池
func Submit(task func()) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrManagerNotInitialized
	}
	return mgr.SubmitToDefault(task)
}

// SubmitTo 提交任务到指定池
func SubmitTo(poolName string, task func()) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrManagerNotInitialized
	}
	return mgr.Submit(poolName, task)
}

// SubmitToType 提交任务到指定类型的池
func SubmitToType(poolType Type, task func()) error {
	return SubmitTo(string(poolType), task)
}

// SubmitWithContext 提交带上下文的任务到默认池
func SubmitWithContext(ctx context.Context, task func()) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrManagerNotInitialized
	}
	return mgr.SubmitWithContext(ctx, string(DefaultPool), task)
}

// SubmitToWithContext 提交带上下文的任务到指定池
func SubmitToWithContext(ctx context.Context, poolName string, task func()) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrManagerNotInitialized
	}
	return mgr.SubmitWithContext(ctx, poolName, task)
}

// Register registers a new pool with the global manager.
func Register(name string, typ Type, config *Config) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrManagerNotInitialized
	}
	return mgr.Register(name, typ, config)
}

// Get 获取指定名称的池
func Get(name string) (*Pool, error) {
	mgr := GetGlobal()
	if mgr == nil {
		return nil, ErrManagerNotInitialized
	}
	return mgr.Get(name)
}

// GetByType 获取指定类型的池
func GetByType(poolType Type) (*Pool, error) {
	return Get(string(poolType))
}

// MustGet 获取池，失败时 panic
func MustGet(name string) *Pool {
	pool, err := Get(name)
	if err != nil {
		panic(err)
	}
	return pool
}

// StatsGlobal returns statistics for all pools.
func StatsGlobal() map[string]Info {
	mgr := GetGlobal()
	if mgr == nil {
		return nil
	}
	return mgr.Stats()
}

// Tune 动态调整指定池的容量
func Tune(name string, size int) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrManagerNotInitialized
	}
	return mgr.Tune(name, size)
}
