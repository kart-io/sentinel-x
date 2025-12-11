package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	// Import gin adapter to register it
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	grpcopts "github.com/kart-io/sentinel-x/pkg/infra/server/grpc"
	httpopts "github.com/kart-io/sentinel-x/pkg/infra/server/http"
	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// mockService implements service.Service for testing.
type mockService struct {
	name        string
	initCalled  bool
	closeCalled bool
	initErr     error
	closeErr    error
	mu          sync.Mutex
}

func (s *mockService) ServiceName() string {
	return s.name
}

func (s *mockService) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initCalled = true
	return s.initErr
}

func (s *mockService) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeCalled = true
	return s.closeErr
}

func (s *mockService) WasInitCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initCalled
}

func (s *mockService) WasCloseCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeCalled
}

// mockHTTPHandler implements transport.HTTPHandler for testing.
type mockHTTPHandler struct{}

func (h *mockHTTPHandler) RegisterRoutes(router transport.Router) {
	router.Handle("GET", "/test", func(ctx transport.Context) {
		ctx.String(200, "test")
	})
}

// mockRunnable implements Runnable for testing.
type mockRunnable struct {
	name        string
	startCalled bool
	stopCalled  bool
	startErr    error
	stopErr     error
	mu          sync.Mutex
}

func (r *mockRunnable) Name() string {
	return r.name
}

func (r *mockRunnable) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startCalled = true
	return r.startErr
}

func (r *mockRunnable) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopCalled = true
	return r.stopErr
}

func (r *mockRunnable) WasStartCalled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startCalled
}

func (r *mockRunnable) WasStopCalled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopCalled
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want func(*Manager) bool
	}{
		{
			name: "default options",
			opts: nil,
			want: func(m *Manager) bool {
				return m != nil && m.registry != nil
			},
		},
		{
			name: "http only mode",
			opts: []Option{
				WithMode(ModeHTTPOnly),
			},
			want: func(m *Manager) bool {
				return m != nil && m.httpServer != nil && m.grpcServer == nil
			},
		},
		{
			name: "grpc only mode",
			opts: []Option{
				WithMode(ModeGRPCOnly),
			},
			want: func(m *Manager) bool {
				return m != nil && m.httpServer == nil && m.grpcServer != nil
			},
		},
		{
			name: "both modes",
			opts: []Option{
				WithMode(ModeBoth),
			},
			want: func(m *Manager) bool {
				return m != nil && m.httpServer != nil && m.grpcServer != nil
			},
		},
		{
			name: "custom shutdown timeout",
			opts: []Option{
				WithShutdownTimeout(60 * time.Second),
			},
			want: func(m *Manager) bool {
				return m != nil && m.opts.ShutdownTimeout == 60*time.Second
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(tt.opts...)
			if !tt.want(mgr) {
				t.Errorf("NewManager() validation failed for test case: %s", tt.name)
			}
		})
	}
}

func TestManagerRegistry(t *testing.T) {
	mgr := NewManager()
	registry := mgr.Registry()

	if registry == nil {
		t.Error("Registry() returned nil")
	}

	if registry != mgr.registry {
		t.Error("Registry() did not return the same registry instance")
	}
}

func TestManagerHTTPServer(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want bool
	}{
		{
			name: "http enabled",
			opts: []Option{
				WithMode(ModeHTTPOnly),
			},
			want: true,
		},
		{
			name: "http disabled",
			opts: []Option{
				WithMode(ModeGRPCOnly),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(tt.opts...)
			httpServer := mgr.HTTPServer()

			if tt.want && httpServer == nil {
				t.Error("HTTPServer() returned nil when HTTP was enabled")
			}
			if !tt.want && httpServer != nil {
				t.Error("HTTPServer() returned non-nil when HTTP was disabled")
			}
		})
	}
}

func TestManagerGRPCServer(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want bool
	}{
		{
			name: "grpc enabled",
			opts: []Option{
				WithMode(ModeGRPCOnly),
			},
			want: true,
		},
		{
			name: "grpc disabled",
			opts: []Option{
				WithMode(ModeHTTPOnly),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(tt.opts...)
			grpcServer := mgr.GRPCServer()

			if tt.want && grpcServer == nil {
				t.Error("GRPCServer() returned nil when gRPC was enabled")
			}
			if !tt.want && grpcServer != nil {
				t.Error("GRPCServer() returned non-nil when gRPC was disabled")
			}
		})
	}
}

func TestManagerRegisterService(t *testing.T) {
	mgr := NewManager()
	svc := &mockService{name: "test-service"}
	handler := &mockHTTPHandler{}

	err := mgr.RegisterService(svc, handler, nil)
	if err != nil {
		t.Fatalf("RegisterService() error = %v", err)
	}

	// Verify service was registered
	registeredSvc, ok := mgr.registry.GetService("test-service")
	if !ok {
		t.Error("Service was not registered")
	}
	if registeredSvc != svc {
		t.Error("Registered service does not match")
	}
}

func TestManagerRegisterHTTP(t *testing.T) {
	mgr := NewManager()
	svc := &mockService{name: "http-service"}
	handler := &mockHTTPHandler{}

	err := mgr.RegisterHTTP(svc, handler)
	if err != nil {
		t.Fatalf("RegisterHTTP() error = %v", err)
	}

	// Verify service was registered
	registeredSvc, ok := mgr.registry.GetService("http-service")
	if !ok {
		t.Error("HTTP service was not registered")
	}
	if registeredSvc != svc {
		t.Error("Registered HTTP service does not match")
	}
}

func TestManagerRegisterGRPC(t *testing.T) {
	mgr := NewManager()
	svc := &mockService{name: "grpc-service"}
	desc := &transport.GRPCServiceDesc{
		ServiceDesc: "test-desc",
		ServiceImpl: "test-impl",
	}

	err := mgr.RegisterGRPC(svc, desc)
	if err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	// Verify service was registered
	registeredSvc, ok := mgr.registry.GetService("grpc-service")
	if !ok {
		t.Error("gRPC service was not registered")
	}
	if registeredSvc != svc {
		t.Error("Registered gRPC service does not match")
	}
}

func TestManagerAddServer(t *testing.T) {
	mgr := NewManager()
	runnable := &mockRunnable{name: "custom-server"}

	mgr.AddServer(runnable)

	if len(mgr.servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(mgr.servers))
	}
	if mgr.servers[0] != runnable {
		t.Error("Added server does not match")
	}
}

func TestManagerAddServerConcurrent(t *testing.T) {
	mgr := NewManager()
	var wg sync.WaitGroup

	// Add servers concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runnable := &mockRunnable{name: "server"}
			mgr.AddServer(runnable)
		}(i)
	}

	wg.Wait()

	if len(mgr.servers) != 100 {
		t.Errorf("Expected 100 servers, got %d", len(mgr.servers))
	}
}

func TestManagerStartStop(t *testing.T) {
	// Skip this test if HTTP server is not properly initialized
	// This is a basic lifecycle test
	mgr := NewManager(
		WithMode(ModeHTTPOnly),
		WithHTTPOptions(&httpopts.Options{
			Addr:         ":0", // Use random port
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
			Adapter:      httpopts.AdapterGin,
			Middleware:   mwopts.NewOptions(), // Use default middleware options
		}),
	)

	svc := &mockService{name: "test-service"}
	handler := &mockHTTPHandler{}
	err := mgr.RegisterHTTP(svc, handler)
	if err != nil {
		t.Fatalf("RegisterHTTP() error = %v", err)
	}

	ctx := context.Background()

	// Note: We cannot actually start the server without a registered adapter
	// This test validates the flow, but actual server start may fail
	// without proper adapter registration
	_ = ctx

	// Test double start
	mgr.started = true
	err = mgr.Start(context.Background())
	if err == nil {
		t.Error("Expected error when starting already started manager")
	}
	mgr.started = false

	// Test stop without start
	err = mgr.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() on non-started manager should not error, got: %v", err)
	}
}

func TestManagerServiceLifecycle(t *testing.T) {
	mgr := NewManager(WithMode(ModeHTTPOnly))

	svc := &mockService{name: "lifecycle-service"}
	handler := &mockHTTPHandler{}

	err := mgr.RegisterHTTP(svc, handler)
	if err != nil {
		t.Fatalf("RegisterHTTP() error = %v", err)
	}

	// Simulate initialization
	ctx := context.Background()
	services := mgr.registry.GetAllServices()
	for _, s := range services {
		if init, ok := s.(service.Initializable); ok {
			if err := init.Init(ctx); err != nil {
				t.Fatalf("Init() error = %v", err)
			}
		}
	}

	if !svc.WasInitCalled() {
		t.Error("Service Init() was not called")
	}

	// Simulate close
	for _, s := range services {
		if closer, ok := s.(service.Closeable); ok {
			if err := closer.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		}
	}

	if !svc.WasCloseCalled() {
		t.Error("Service Close() was not called")
	}
}

func TestManagerServiceInitError(t *testing.T) {
	mgr := NewManager(WithMode(ModeHTTPOnly))

	svc := &mockService{
		name:    "error-service",
		initErr: errors.New("init failed"),
	}
	handler := &mockHTTPHandler{}

	err := mgr.RegisterHTTP(svc, handler)
	if err != nil {
		t.Fatalf("RegisterHTTP() error = %v", err)
	}

	ctx := context.Background()
	services := mgr.registry.GetAllServices()
	for _, s := range services {
		if init, ok := s.(service.Initializable); ok {
			err := init.Init(ctx)
			if err == nil {
				t.Error("Expected init error, got nil")
			}
			if err.Error() != "init failed" {
				t.Errorf("Expected 'init failed', got '%v'", err)
			}
		}
	}
}

func TestManagerCustomServerLifecycle(t *testing.T) {
	mgr := NewManager()
	runnable := &mockRunnable{name: "custom-server"}

	mgr.AddServer(runnable)

	ctx := context.Background()

	// Simulate start
	for _, server := range mgr.servers {
		if err := server.Start(ctx); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	}

	if !runnable.WasStartCalled() {
		t.Error("Custom server Start() was not called")
	}

	// Simulate stop
	for _, server := range mgr.servers {
		if err := server.Stop(ctx); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
	}

	if !runnable.WasStopCalled() {
		t.Error("Custom server Stop() was not called")
	}
}

func TestManagerCustomServerErrors(t *testing.T) {
	mgr := NewManager()
	runnable := &mockRunnable{
		name:     "error-server",
		startErr: errors.New("start failed"),
		stopErr:  errors.New("stop failed"),
	}

	mgr.AddServer(runnable)

	ctx := context.Background()

	// Test start error
	for _, server := range mgr.servers {
		err := server.Start(ctx)
		if err == nil {
			t.Error("Expected start error, got nil")
		}
		if err.Error() != "start failed" {
			t.Errorf("Expected 'start failed', got '%v'", err)
		}
	}

	// Test stop error
	for _, server := range mgr.servers {
		err := server.Stop(ctx)
		if err == nil {
			t.Error("Expected stop error, got nil")
		}
		if err.Error() != "stop failed" {
			t.Errorf("Expected 'stop failed', got '%v'", err)
		}
	}
}

func TestManagerWait_NotStarted(t *testing.T) {
	mgr := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := mgr.Wait(ctx)
	if err == nil {
		t.Error("Wait() should return error when manager is not started")
	}
	if err.Error() != "server manager not started" {
		t.Errorf("Expected 'server manager not started' error, got: %v", err)
	}
}

func TestManagerWait_NoServers(t *testing.T) {
	// Create manager with servers explicitly disabled
	mgr := NewManager(WithMode(ModeHTTPOnly))

	// Manually set started flag and clear servers for this test
	mgr.mu.Lock()
	mgr.started = true
	mgr.httpServer = nil
	mgr.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := mgr.Wait(ctx)
	if err == nil {
		t.Error("Wait() should return error when no servers are configured")
	}
	if err.Error() != "no servers configured" {
		t.Errorf("Expected 'no servers configured' error, got: %v", err)
	}
}

func TestManagerWait_HTTPServerReady(t *testing.T) {
	// Create manager with HTTP server
	mgr := NewManager(
		WithMode(ModeHTTPOnly),
		WithHTTPOptions(&httpopts.Options{
			Addr:         ":0", // Random port
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
			Adapter:      httpopts.AdapterGin,
			Middleware:   mwopts.NewOptions(),
		}),
	)

	// Start the server
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mgr.Stop(shutdownCtx)
	}()

	// Wait for server to be ready
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mgr.Wait(waitCtx)
	if err != nil {
		t.Errorf("Wait() should succeed when server is ready, got error: %v", err)
	}
}

func TestManagerWait_GRPCServerReady(t *testing.T) {
	// Create manager with gRPC server
	mgr := NewManager(
		WithMode(ModeGRPCOnly),
		WithGRPCOptions(&grpcopts.Options{
			Addr:             ":0", // Use random port
			Timeout:          10 * time.Second,
			MaxRecvMsgSize:   16 * 1024 * 1024,
			MaxSendMsgSize:   16 * 1024 * 1024,
			EnableReflection: true,
		}),
	)

	// Start the server
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mgr.Stop(shutdownCtx)
	}()

	// Wait for server to be ready
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mgr.Wait(waitCtx)
	if err != nil {
		t.Errorf("Wait() should succeed when gRPC server is ready, got error: %v", err)
	}
}

func TestManagerWait_BothServersReady(t *testing.T) {
	// Create manager with both HTTP and gRPC servers
	mgr := NewManager(
		WithMode(ModeBoth),
		WithHTTPOptions(&httpopts.Options{
			Addr:         ":0", // Use random port
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
			Adapter:      httpopts.AdapterGin,
			Middleware:   mwopts.NewOptions(),
		}),
		WithGRPCOptions(&grpcopts.Options{
			Addr:             ":0", // Use random port
			Timeout:          10 * time.Second,
			MaxRecvMsgSize:   16 * 1024 * 1024,
			MaxSendMsgSize:   16 * 1024 * 1024,
			EnableReflection: true,
		}),
	)

	// Start the servers
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Failed to start servers: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mgr.Stop(shutdownCtx)
	}()

	// Wait for servers to be ready
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mgr.Wait(waitCtx)
	if err != nil {
		t.Errorf("Wait() should succeed when both servers are ready, got error: %v", err)
	}
}

func TestManagerWait_CalledMultipleTimes(t *testing.T) {
	// Create manager with HTTP server
	mgr := NewManager(
		WithMode(ModeHTTPOnly),
		WithHTTPOptions(&httpopts.Options{
			Addr:         ":0",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
			Adapter:      httpopts.AdapterGin,
			Middleware:   mwopts.NewOptions(),
		}),
	)

	// Start the server
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mgr.Stop(shutdownCtx)
	}()

	// Call Wait multiple times
	for i := 0; i < 3; i++ {
		waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := mgr.Wait(waitCtx)
		cancel()

		if err != nil {
			t.Errorf("Wait() call %d should succeed, got error: %v", i+1, err)
		}
	}
}

// Benchmark Wait method
func BenchmarkManagerWait(b *testing.B) {
	mgr := NewManager(
		WithMode(ModeHTTPOnly),
		WithHTTPOptions(&httpopts.Options{
			Addr:         ":0",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
			Adapter:      httpopts.AdapterGin,
			Middleware:   mwopts.NewOptions(),
		}),
	)

	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mgr.Stop(shutdownCtx)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = mgr.Wait(waitCtx)
		cancel()
	}
}
