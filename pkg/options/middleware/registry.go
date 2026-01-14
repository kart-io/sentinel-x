package middleware

import (
	"fmt"
	"sort"
	"sync"
)

// Registry 中间件配置注册器。
// 管理所有中间件配置的工厂函数，支持动态注册和创建。
type Registry struct {
	mu               sync.RWMutex
	factories        map[string]func() MiddlewareConfig
	handlerFactories map[string]Factory
	routeRegistrars  map[string]RouteRegistrar
}

// globalRegistry 全局中间件注册器实例。
var globalRegistry = &Registry{
	factories:        make(map[string]func() MiddlewareConfig),
	handlerFactories: make(map[string]Factory),
	routeRegistrars:  make(map[string]RouteRegistrar),
}

// Register 注册中间件配置工厂函数。
// 通常在各中间件文件的 init() 函数中调用。
// 如果同名中间件已注册，会触发 panic。
func Register(name string, factory func() MiddlewareConfig) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.factories[name]; exists {
		panic(fmt.Sprintf("middleware %q already registered", name))
	}
	globalRegistry.factories[name] = factory
}

// RegisterFactory 注册中间件工厂。
// 工厂用于根据配置创建 Gin 中间件处理函数。
// 如果同名工厂已注册，会触发 panic。
func RegisterFactory(f Factory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	name := f.Name()
	if _, exists := globalRegistry.handlerFactories[name]; exists {
		panic(fmt.Sprintf("middleware factory %q already registered", name))
	}
	globalRegistry.handlerFactories[name] = f
}

// RegisterRouteRegistrar 注册路由注册器。
// 某些中间件需要注册独立路由（如 health、metrics、pprof、version）。
func RegisterRouteRegistrar(name string, r RouteRegistrar) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.routeRegistrars[name] = r
}

// MustRegister 注册中间件配置工厂函数（允许覆盖）。
// 用于测试场景或需要覆盖默认实现的情况。
func MustRegister(name string, factory func() MiddlewareConfig) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.factories[name] = factory
}

// MustRegisterFactory 注册中间件工厂（允许覆盖）。
// 用于测试场景或需要覆盖默认实现的情况。
func MustRegisterFactory(f Factory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.handlerFactories[f.Name()] = f
}

// Create 创建中间件配置实例。
// 根据名称查找工厂函数并创建新的配置实例。
func Create(name string) (MiddlewareConfig, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	factory, ok := globalRegistry.factories[name]
	if !ok {
		return nil, fmt.Errorf("middleware %q not registered", name)
	}
	return factory(), nil
}

// GetFactory 获取中间件工厂。
func GetFactory(name string) (Factory, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	f, ok := globalRegistry.handlerFactories[name]
	return f, ok
}

// GetRouteRegistrar 获取路由注册器。
func GetRouteRegistrar(name string) (RouteRegistrar, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	r, ok := globalRegistry.routeRegistrars[name]
	return r, ok
}

// IsRegistered 检查中间件是否已注册。
func IsRegistered(name string) bool {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	_, ok := globalRegistry.factories[name]
	return ok
}

// IsFactoryRegistered 检查中间件工厂是否已注册。
func IsFactoryRegistered(name string) bool {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	_, ok := globalRegistry.handlerFactories[name]
	return ok
}

// ListRegistered 返回所有已注册的中间件名称（按字母排序）。
func ListRegistered() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0, len(globalRegistry.factories))
	for name := range globalRegistry.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListFactories 返回所有已注册的中间件工厂名称（按字母排序）。
func ListFactories() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0, len(globalRegistry.handlerFactories))
	for name := range globalRegistry.handlerFactories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CreateAll 创建所有已注册中间件的配置实例。
// 返回名称到配置实例的映射。
func CreateAll() map[string]MiddlewareConfig {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	configs := make(map[string]MiddlewareConfig, len(globalRegistry.factories))
	for name, factory := range globalRegistry.factories {
		configs[name] = factory()
	}
	return configs
}

// ResetRegistry 重置注册器（仅用于测试）。
func ResetRegistry() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.factories = make(map[string]func() MiddlewareConfig)
	globalRegistry.handlerFactories = make(map[string]Factory)
	globalRegistry.routeRegistrars = make(map[string]RouteRegistrar)
}
