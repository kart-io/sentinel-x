package http

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// mockBridge implements FrameworkBridge for testing.
type mockBridge struct {
	name string
}

func (m *mockBridge) Name() string {
	return m.name
}

func (m *mockBridge) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (m *mockBridge) AddRoute(_, _ string, _ BridgeHandler) {}

func (m *mockBridge) AddRouteGroup(_ string) RouteGroup {
	return &mockRouteGroup{}
}

func (m *mockBridge) AddMiddleware(_ BridgeMiddleware) {}

func (m *mockBridge) SetNotFoundHandler(_ BridgeHandler) {}

func (m *mockBridge) SetErrorHandler(_ BridgeErrorHandler) {}

func (m *mockBridge) Static(_, _ string) {}

func (m *mockBridge) Mount(_ string, _ http.Handler) {}

// mockRouteGroup implements RouteGroup for testing.
type mockRouteGroup struct{}

func (m *mockRouteGroup) AddRoute(_, _ string, _ BridgeHandler) {}

func (m *mockRouteGroup) AddRouteGroup(_ string) RouteGroup {
	return &mockRouteGroup{}
}

func (m *mockRouteGroup) AddMiddleware(_ BridgeMiddleware) {}

func (m *mockRouteGroup) Static(_, _ string) {}

func (m *mockRouteGroup) Mount(_ string, _ http.Handler) {}

func TestRegisterBridge(t *testing.T) {
	// Clear existing bridges for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	adapterType := httpopts.AdapterType("test-adapter")
	factory := func() FrameworkBridge {
		return &mockBridge{name: "test"}
	}

	RegisterBridge(adapterType, factory)

	// Verify bridge was registered
	retrievedFactory, ok := getBridge(adapterType)
	if !ok {
		t.Error("Bridge was not registered")
	}

	// Verify factory works
	bridge := retrievedFactory()
	if bridge.Name() != "test" {
		t.Errorf("Expected bridge name 'test', got '%s'", bridge.Name())
	}
}

func TestRegisterBridgeConcurrent(t *testing.T) {
	// Clear existing bridges for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	var wg sync.WaitGroup

	// Register bridges concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			adapterType := httpopts.AdapterType(fmt.Sprintf("test-%d", id))
			factory := func() FrameworkBridge {
				return &mockBridge{name: fmt.Sprintf("bridge-%d", id)}
			}
			RegisterBridge(adapterType, factory)
		}(i)
	}

	wg.Wait()

	// Verify all bridges were registered
	bridgesMu.RLock()
	count := len(bridges)
	bridgesMu.RUnlock()

	if count != 100 {
		t.Errorf("Expected 100 bridges, got %d", count)
	}
}

func TestGetBridgeConcurrent(t *testing.T) {
	// Clear and setup test bridges
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	// Pre-register some bridges
	for i := 0; i < 10; i++ {
		adapterType := httpopts.AdapterType(fmt.Sprintf("test-%d", i))
		factory := func(id int) BridgeFactory {
			return func() FrameworkBridge {
				return &mockBridge{name: fmt.Sprintf("bridge-%d", id)}
			}
		}(i)
		RegisterBridge(adapterType, factory)
	}

	var wg sync.WaitGroup

	// Read bridges concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			adapterType := httpopts.AdapterType(fmt.Sprintf("test-%d", id%10))
			factory, ok := getBridge(adapterType)
			if !ok {
				t.Errorf("Bridge not found: %s", adapterType)
				return
			}
			bridge := factory()
			if bridge == nil {
				t.Error("Factory returned nil bridge")
			}
		}(i)
	}

	wg.Wait()
}

func TestGetBridgeNonExistent(t *testing.T) {
	// Clear existing bridges for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	adapterType := httpopts.AdapterType("non-existent")
	_, ok := getBridge(adapterType)
	if ok {
		t.Error("getBridge() returned true for non-existent adapter")
	}
}

func TestGetAdapter(t *testing.T) {
	// Clear existing adapters for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	legacyAdaptersMu.Lock()
	legacyAdapters = make(map[httpopts.AdapterType]AdapterFactory)
	legacyAdaptersMu.Unlock()

	adapterType := httpopts.AdapterType("test-get-adapter")
	factory := func() FrameworkBridge {
		return &mockBridge{name: "test-get-adapter"}
	}

	RegisterBridge(adapterType, factory)

	adapter := GetAdapter(adapterType)
	if adapter == nil {
		t.Fatal("GetAdapter() returned nil")
	}

	if adapter.Name() != "test-get-adapter" {
		t.Errorf("Expected adapter name 'test-get-adapter', got '%s'", adapter.Name())
	}
}

func TestGetAdapterNonExistent(t *testing.T) {
	// Clear existing adapters for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	legacyAdaptersMu.Lock()
	legacyAdapters = make(map[httpopts.AdapterType]AdapterFactory)
	legacyAdaptersMu.Unlock()

	adapterType := httpopts.AdapterType("non-existent-adapter")
	adapter := GetAdapter(adapterType)
	if adapter != nil {
		t.Error("GetAdapter() should return nil for non-existent adapter")
	}
}

func TestGetBridgeFunction(t *testing.T) {
	// Clear existing bridges for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	adapterType := httpopts.AdapterType("test-get-bridge")
	factory := func() FrameworkBridge {
		return &mockBridge{name: "test-get-bridge"}
	}

	RegisterBridge(adapterType, factory)

	bridge := GetBridge(adapterType)
	if bridge == nil {
		t.Fatal("GetBridge() returned nil")
	}

	if bridge.Name() != "test-get-bridge" {
		t.Errorf("Expected bridge name 'test-get-bridge', got '%s'", bridge.Name())
	}
}

func TestGetBridgeNotFound(t *testing.T) {
	// Clear existing bridges for test isolation
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	adapterType := httpopts.AdapterType("non-existent-bridge")
	bridge := GetBridge(adapterType)
	if bridge != nil {
		t.Error("GetBridge() should return nil for non-existent bridge")
	}
}

func TestRegisterAdapter(t *testing.T) {
	// Clear existing adapters for test isolation
	legacyAdaptersMu.Lock()
	legacyAdapters = make(map[httpopts.AdapterType]AdapterFactory)
	legacyAdaptersMu.Unlock()

	// Create a mock adapter
	mockAdapter := &bridgeAdapter{
		bridge: &mockBridge{name: "legacy-test"},
		router: &bridgeRouter{group: &mockRouteGroup{}},
	}

	adapterType := httpopts.AdapterType("legacy-adapter")
	factory := func() Adapter {
		return mockAdapter
	}

	RegisterAdapter(adapterType, factory)

	// Verify adapter was registered
	retrievedFactory, ok := getLegacyAdapter(adapterType)
	if !ok {
		t.Error("Legacy adapter was not registered")
	}

	// Verify factory works
	adapter := retrievedFactory()
	if adapter.Name() != "legacy-test" {
		t.Errorf("Expected adapter name 'legacy-test', got '%s'", adapter.Name())
	}
}

func TestBridgeAdapterName(t *testing.T) {
	bridge := &mockBridge{name: "test-bridge"}
	adapter := newBridgeAdapter(bridge)

	if adapter.Name() != "test-bridge" {
		t.Errorf("Expected name 'test-bridge', got '%s'", adapter.Name())
	}
}

func TestBridgeAdapterRouter(t *testing.T) {
	bridge := &mockBridge{name: "test-bridge"}
	adapter := newBridgeAdapter(bridge)

	router := adapter.Router()
	if router == nil {
		t.Error("Router() returned nil")
	}
}

func TestBridgeAdapterHandler(t *testing.T) {
	bridge := &mockBridge{name: "test-bridge"}
	adapter := newBridgeAdapter(bridge)

	handler := adapter.Handler()
	if handler == nil {
		t.Error("Handler() returned nil")
	}
}

func TestBridgeAdapterBridge(t *testing.T) {
	bridge := &mockBridge{name: "test-bridge"}
	adapter := newBridgeAdapter(bridge)

	retrievedBridge := adapter.Bridge()
	if retrievedBridge != bridge {
		t.Error("Bridge() did not return the original bridge")
	}
}

func TestBridgeAdapterSetNotFoundHandler(_ *testing.T) {
	bridge := &mockBridge{name: "test-bridge"}
	adapter := newBridgeAdapter(bridge)

	// Should not panic
	adapter.SetNotFoundHandler(func(_ transport.Context) {})
}

func TestBridgeAdapterSetErrorHandler(_ *testing.T) {
	bridge := &mockBridge{name: "test-bridge"}
	adapter := newBridgeAdapter(bridge)

	// Should not panic
	adapter.SetErrorHandler(func(_ error, _ transport.Context) {})
}

func TestBridgeRouterHandle(_ *testing.T) {
	group := &mockRouteGroup{}
	router := &bridgeRouter{group: group}

	// Should not panic
	router.Handle("GET", "/test", func(_ transport.Context) {})
}

func TestBridgeRouterGroup(t *testing.T) {
	group := &mockRouteGroup{}
	router := &bridgeRouter{group: group}

	subRouter := router.Group("/api")
	if subRouter == nil {
		t.Error("Group() returned nil")
	}
}

func TestBridgeRouterUse(_ *testing.T) {
	group := &mockRouteGroup{}
	router := &bridgeRouter{group: group}

	// Should not panic
	middleware := func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(ctx transport.Context) {
			next(ctx)
		}
	}

	router.Use(middleware)
}

func TestConcurrentReadWrite(_ *testing.T) {
	// Clear existing state
	bridgesMu.Lock()
	bridges = make(map[httpopts.AdapterType]BridgeFactory)
	bridgesMu.Unlock()

	var wg sync.WaitGroup

	// Mix reads and writes
	for i := 0; i < 50; i++ {
		wg.Add(2)

		// Writer
		go func(id int) {
			defer wg.Done()
			adapterType := httpopts.AdapterType(fmt.Sprintf("concurrent-%d", id))
			factory := func() FrameworkBridge {
				return &mockBridge{name: fmt.Sprintf("bridge-%d", id)}
			}
			RegisterBridge(adapterType, factory)
		}(i)

		// Reader
		go func(id int) {
			defer wg.Done()
			adapterType := httpopts.AdapterType(fmt.Sprintf("concurrent-%d", id))
			getBridge(adapterType)
			GetAdapter(adapterType)
			GetBridge(adapterType)
		}(i)
	}

	wg.Wait()
}

func TestRequestContextCreation(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := &mockResponseWriter{}
	ctx := NewRequestContext(req, w)

	if ctx == nil {
		t.Fatal("NewRequestContext() returned nil")
	}

	if ctx.HTTPRequest() != req {
		t.Error("HTTPRequest() did not return the original request")
	}

	if ctx.ResponseWriter() != w {
		t.Error("ResponseWriter() did not return the original writer")
	}
}

func TestRequestContextParams(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	w := &mockResponseWriter{}
	ctx := NewRequestContext(req, w)

	ctx.SetParam("id", "123")
	if ctx.Param("id") != "123" {
		t.Errorf("Expected param 'id' to be '123', got '%s'", ctx.Param("id"))
	}

	ctx.SetParams(map[string]string{"name": "test", "age": "30"})
	if ctx.Param("name") != "test" {
		t.Errorf("Expected param 'name' to be 'test', got '%s'", ctx.Param("name"))
	}
	if ctx.Param("age") != "30" {
		t.Errorf("Expected param 'age' to be '30', got '%s'", ctx.Param("age"))
	}
}

func TestRequestContextQuery(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/test?foo=bar&baz=qux", nil)
	w := &mockResponseWriter{}
	ctx := NewRequestContext(req, w)

	if ctx.Query("foo") != "bar" {
		t.Errorf("Expected query 'foo' to be 'bar', got '%s'", ctx.Query("foo"))
	}

	if ctx.Query("baz") != "qux" {
		t.Errorf("Expected query 'baz' to be 'qux', got '%s'", ctx.Query("baz"))
	}

	if ctx.QueryDefault("missing", "default") != "default" {
		t.Errorf("Expected QueryDefault to return 'default', got '%s'", ctx.QueryDefault("missing", "default"))
	}
}

func TestRequestContextHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("X-Test", "value")
	w := &mockResponseWriter{header: make(http.Header)}
	ctx := NewRequestContext(req, w)

	if ctx.Header("X-Test") != "value" {
		t.Errorf("Expected header 'X-Test' to be 'value', got '%s'", ctx.Header("X-Test"))
	}

	ctx.SetHeader("X-Response", "response-value")
	if w.header.Get("X-Response") != "response-value" {
		t.Error("SetHeader did not set the response header")
	}
}

func TestRequestContextRawContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	w := &mockResponseWriter{}
	ctx := NewRequestContext(req, w)

	rawCtx := "raw-context-data"
	ctx.SetRawContext(rawCtx)

	if ctx.GetRawContext() != rawCtx {
		t.Error("GetRawContext() did not return the set raw context")
	}
}

// mockResponseWriter implements http.ResponseWriter for testing.
type mockResponseWriter struct {
	header http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriter) WriteHeader(_ int) {}
