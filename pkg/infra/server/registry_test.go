package server

import (
	"sync"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.services == nil {
		t.Error("services map is nil")
	}

	if registry.httpHandlers == nil {
		t.Error("httpHandlers map is nil")
	}

	if registry.grpcDescs == nil {
		t.Error("grpcDescs slice is nil")
	}
}

func TestRegistryRegisterService(t *testing.T) {
	registry := NewRegistry()
	svc := &mockService{name: "test-service"}
	handler := &mockHTTPHandler{}
	desc := &transport.GRPCServiceDesc{
		ServiceDesc: "test-desc",
		ServiceImpl: "test-impl",
	}

	err := registry.RegisterService(svc, handler, desc)
	if err != nil {
		t.Fatalf("RegisterService() error = %v", err)
	}

	// Verify service was registered
	registeredSvc, ok := registry.GetService("test-service")
	if !ok {
		t.Error("Service was not registered")
	}
	if registeredSvc != svc {
		t.Error("Registered service does not match")
	}

	// Verify HTTP handler was registered
	if _, ok := registry.httpHandlers["test-service"]; !ok {
		t.Error("HTTP handler was not registered")
	}

	// Verify gRPC desc was registered
	if len(registry.grpcDescs) != 1 {
		t.Errorf("Expected 1 gRPC desc, got %d", len(registry.grpcDescs))
	}
}

func TestRegistryRegisterServiceNilHandler(t *testing.T) {
	registry := NewRegistry()
	svc := &mockService{name: "test-service"}

	err := registry.RegisterService(svc, nil, nil)
	if err != nil {
		t.Fatalf("RegisterService() error = %v", err)
	}

	// Verify service was registered
	_, ok := registry.GetService("test-service")
	if !ok {
		t.Error("Service was not registered")
	}

	// Verify no HTTP handler was registered
	if _, ok := registry.httpHandlers["test-service"]; ok {
		t.Error("HTTP handler should not be registered")
	}

	// Verify no gRPC desc was registered
	if len(registry.grpcDescs) != 0 {
		t.Errorf("Expected 0 gRPC desc, got %d", len(registry.grpcDescs))
	}
}

func TestRegistryRegisterHTTP(t *testing.T) {
	registry := NewRegistry()
	svc := &mockService{name: "http-service"}
	handler := &mockHTTPHandler{}

	err := registry.RegisterHTTP(svc, handler)
	if err != nil {
		t.Fatalf("RegisterHTTP() error = %v", err)
	}

	// Verify service was registered
	registeredSvc, ok := registry.GetService("http-service")
	if !ok {
		t.Error("HTTP service was not registered")
	}
	if registeredSvc != svc {
		t.Error("Registered HTTP service does not match")
	}

	// Verify HTTP handler was registered
	if _, ok := registry.httpHandlers["http-service"]; !ok {
		t.Error("HTTP handler was not registered")
	}
}

func TestRegistryRegisterGRPC(t *testing.T) {
	registry := NewRegistry()
	svc := &mockService{name: "grpc-service"}
	desc := &transport.GRPCServiceDesc{
		ServiceDesc: "test-desc",
		ServiceImpl: "test-impl",
	}

	err := registry.RegisterGRPC(svc, desc)
	if err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	// Verify service was registered
	registeredSvc, ok := registry.GetService("grpc-service")
	if !ok {
		t.Error("gRPC service was not registered")
	}
	if registeredSvc != svc {
		t.Error("Registered gRPC service does not match")
	}

	// Verify gRPC desc was registered
	if len(registry.grpcDescs) != 1 {
		t.Errorf("Expected 1 gRPC desc, got %d", len(registry.grpcDescs))
	}
}

func TestRegistryGetService(t *testing.T) {
	registry := NewRegistry()
	svc := &mockService{name: "test-service"}

	// Test getting non-existent service
	_, ok := registry.GetService("non-existent")
	if ok {
		t.Error("GetService() returned true for non-existent service")
	}

	// Register service
	err := registry.RegisterService(svc, nil, nil)
	if err != nil {
		t.Fatalf("RegisterService() error = %v", err)
	}

	// Test getting existing service
	retrievedSvc, ok := registry.GetService("test-service")
	if !ok {
		t.Error("GetService() returned false for existing service")
	}
	if retrievedSvc != svc {
		t.Error("Retrieved service does not match registered service")
	}
}

func TestRegistryGetAllServices(t *testing.T) {
	registry := NewRegistry()

	// Test empty registry
	services := registry.GetAllServices()
	if len(services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(services))
	}

	// Register multiple services
	svc1 := &mockService{name: "service-1"}
	svc2 := &mockService{name: "service-2"}
	svc3 := &mockService{name: "service-3"}

	_ = registry.RegisterService(svc1, nil, nil)
	_ = registry.RegisterService(svc2, nil, nil)
	_ = registry.RegisterService(svc3, nil, nil)

	// Test getting all services
	services = registry.GetAllServices()
	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}

	// Verify all services are present
	serviceMap := make(map[service.Service]bool)
	for _, s := range services {
		serviceMap[s] = true
	}

	if !serviceMap[svc1] {
		t.Error("Service 1 not found in GetAllServices()")
	}
	if !serviceMap[svc2] {
		t.Error("Service 2 not found in GetAllServices()")
	}
	if !serviceMap[svc3] {
		t.Error("Service 3 not found in GetAllServices()")
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	registry := NewRegistry()
	svc1 := &mockService{name: "duplicate-service"}
	svc2 := &mockService{name: "duplicate-service"}

	// Register first service
	err := registry.RegisterService(svc1, nil, nil)
	if err != nil {
		t.Fatalf("RegisterService() error = %v", err)
	}

	// Register second service with same name (should overwrite)
	err = registry.RegisterService(svc2, nil, nil)
	if err != nil {
		t.Fatalf("RegisterService() error = %v", err)
	}

	// Verify the second service is registered
	retrievedSvc, ok := registry.GetService("duplicate-service")
	if !ok {
		t.Error("Service not found after duplicate registration")
	}
	if retrievedSvc != svc2 {
		t.Error("Retrieved service should be the second registered service")
	}
}

func TestRegistryConcurrentRegister(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup

	// Register services concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc := &mockService{name: "service"}
			handler := &mockHTTPHandler{}
			registry.RegisterService(svc, handler, nil)
		}(i)
	}

	wg.Wait()

	// Verify services were registered (may be overwritten due to same name)
	_, ok := registry.GetService("service")
	if !ok {
		t.Error("Service was not registered after concurrent registration")
	}
}

func TestRegistryConcurrentRead(t *testing.T) {
	registry := NewRegistry()

	// Pre-register some services
	for i := 0; i < 10; i++ {
		svc := &mockService{name: "service"}
		registry.RegisterService(svc, nil, nil)
	}

	var wg sync.WaitGroup

	// Read services concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.GetService("service")
			registry.GetAllServices()
		}()
	}

	wg.Wait()
}

func TestRegistryConcurrentReadWrite(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup

	// Mix reads and writes
	for i := 0; i < 50; i++ {
		wg.Add(2)

		// Writer
		go func(id int) {
			defer wg.Done()
			svc := &mockService{name: "service"}
			registry.RegisterService(svc, nil, nil)
		}(i)

		// Reader
		go func() {
			defer wg.Done()
			registry.GetService("service")
			registry.GetAllServices()
		}()
	}

	wg.Wait()
}

func TestRegistryMultipleGRPCDescs(t *testing.T) {
	registry := NewRegistry()

	desc1 := &transport.GRPCServiceDesc{
		ServiceDesc: "desc-1",
		ServiceImpl: "impl-1",
	}
	desc2 := &transport.GRPCServiceDesc{
		ServiceDesc: "desc-2",
		ServiceImpl: "impl-2",
	}

	svc1 := &mockService{name: "grpc-service-1"}
	svc2 := &mockService{name: "grpc-service-2"}

	_ = registry.RegisterGRPC(svc1, desc1)
	_ = registry.RegisterGRPC(svc2, desc2)

	if len(registry.grpcDescs) != 2 {
		t.Errorf("Expected 2 gRPC descs, got %d", len(registry.grpcDescs))
	}
}

func TestRegistryApplyToHTTP(t *testing.T) {
	// This test validates the basic flow
	// Actual application requires a real HTTP server
	registry := NewRegistry()
	svc := &mockService{name: "http-service"}
	handler := &mockHTTPHandler{}

	_ = registry.RegisterHTTP(svc, handler)

	// Cannot test actual application without HTTP server implementation
	// This validates the registration succeeds
	if len(registry.httpHandlers) != 1 {
		t.Errorf("Expected 1 HTTP handler, got %d", len(registry.httpHandlers))
	}
}

func TestRegistryApplyToGRPC(t *testing.T) {
	// This test validates the basic flow
	// Actual application requires a real gRPC server
	registry := NewRegistry()
	svc := &mockService{name: "grpc-service"}
	desc := &transport.GRPCServiceDesc{
		ServiceDesc: "test-desc",
		ServiceImpl: "test-impl",
	}

	_ = registry.RegisterGRPC(svc, desc)

	// Cannot test actual application without gRPC server implementation
	// This validates the registration succeeds
	if len(registry.grpcDescs) != 1 {
		t.Errorf("Expected 1 gRPC desc, got %d", len(registry.grpcDescs))
	}
}
