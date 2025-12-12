package bootstrap

import (
	"context"
	"testing"
)

// mockInitializer 是测试用的模拟初始化器
type mockInitializer struct {
	name string
	deps []string
}

func (m *mockInitializer) Name() string           { return m.name }
func (m *mockInitializer) Dependencies() []string { return m.deps }
func (m *mockInitializer) Initialize(ctx context.Context) error {
	return nil
}

func newMock(name string, deps ...string) Initializer {
	return &mockInitializer{name: name, deps: deps}
}

func TestResolveDependencies_NoDependencies(t *testing.T) {
	inits := []Initializer{
		newMock("a"),
		newMock("b"),
		newMock("c"),
	}

	result, err := ResolveDependencies(inits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 initializers, got %d", len(result))
	}
}

func TestResolveDependencies_LinearDependencies(t *testing.T) {
	// a -> b -> c （c 最先，a 最后）
	inits := []Initializer{
		newMock("a", "b"),
		newMock("b", "c"),
		newMock("c"),
	}

	result, err := ResolveDependencies(inits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证顺序：c 应该在 b 之前，b 应该在 a 之前
	order := make(map[string]int)
	for i, init := range result {
		order[init.Name()] = i
	}

	if order["c"] >= order["b"] {
		t.Errorf("c should be before b, got c=%d, b=%d", order["c"], order["b"])
	}
	if order["b"] >= order["a"] {
		t.Errorf("b should be before a, got b=%d, a=%d", order["b"], order["a"])
	}
}

func TestResolveDependencies_DiamondDependency(t *testing.T) {
	// a -> b, a -> c, b -> d, c -> d
	inits := []Initializer{
		newMock("a", "b", "c"),
		newMock("b", "d"),
		newMock("c", "d"),
		newMock("d"),
	}

	result, err := ResolveDependencies(inits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := make(map[string]int)
	for i, init := range result {
		order[init.Name()] = i
	}

	// d 应该最先
	if order["d"] != 0 {
		t.Errorf("d should be first, got position %d", order["d"])
	}

	// a 应该最后
	if order["a"] != 3 {
		t.Errorf("a should be last, got position %d", order["a"])
	}
}

func TestResolveDependencies_CircularDependency(t *testing.T) {
	// a -> b -> c -> a （循环）
	inits := []Initializer{
		newMock("a", "b"),
		newMock("b", "c"),
		newMock("c", "a"),
	}

	_, err := ResolveDependencies(inits)
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}

	t.Logf("循环依赖错误: %v", err)
}

func TestResolveDependencies_SelfDependency(t *testing.T) {
	// a -> a （自循环）
	inits := []Initializer{
		newMock("a", "a"),
	}

	_, err := ResolveDependencies(inits)
	if err == nil {
		t.Fatal("expected error for self dependency, got nil")
	}

	t.Logf("自循环错误: %v", err)
}

func TestResolveDependencies_MissingDependency(t *testing.T) {
	// a -> missing
	inits := []Initializer{
		newMock("a", "missing"),
	}

	_, err := ResolveDependencies(inits)
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}

	t.Logf("缺失依赖错误: %v", err)
}

func TestResolveDependencies_DuplicateName(t *testing.T) {
	inits := []Initializer{
		newMock("a"),
		newMock("a"),
	}

	_, err := ResolveDependencies(inits)
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}

	t.Logf("重复名称错误: %v", err)
}

func TestResolveDependencies_Empty(t *testing.T) {
	result, err := ResolveDependencies(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result for empty input, got %v", result)
	}
}

func TestResolveDependencies_RealInitializers(t *testing.T) {
	// 测试实际的初始化器依赖链
	// logging -> datasources -> auth -> middleware -> server
	inits := []Initializer{
		newMock("server", "middleware"),
		newMock("middleware", "logging", "datasources", "auth"),
		newMock("auth", "logging", "datasources"),
		newMock("datasources", "logging"),
		newMock("logging"),
	}

	result, err := ResolveDependencies(inits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := make(map[string]int)
	for i, init := range result {
		order[init.Name()] = i
		t.Logf("%d: %s", i, init.Name())
	}

	// 验证顺序
	if order["logging"] >= order["datasources"] {
		t.Error("logging should be before datasources")
	}
	if order["datasources"] >= order["auth"] {
		t.Error("datasources should be before auth")
	}
	if order["auth"] >= order["middleware"] {
		t.Error("auth should be before middleware")
	}
	if order["middleware"] >= order["server"] {
		t.Error("middleware should be before server")
	}
}
