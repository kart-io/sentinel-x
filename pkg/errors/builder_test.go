package errors

import (
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestRegisterService(t *testing.T) {
	// Register a new service
	RegisterService(99, "test-service")

	// Get service name
	name, ok := GetServiceName(99)
	if !ok {
		t.Error("GetServiceName should find registered service")
	}
	if name != "test-service" {
		t.Errorf("GetServiceName() = %q, want %q", name, "test-service")
	}

	// Register same code with same name should not panic
	RegisterService(99, "test-service")

	// Register same code with different name should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("RegisterService should panic on conflict")
		}
	}()
	RegisterService(99, "different-service")
}

func TestGetAllServices(t *testing.T) {
	RegisterService(98, "another-test-service")

	all := GetAllServices()
	if _, ok := all[98]; !ok {
		t.Error("GetAllServices should include registered service")
	}

	// Verify it's a copy
	all[97] = "modified"
	if _, ok := GetServiceName(97); ok {
		t.Error("GetAllServices should return a copy")
	}
}

func TestNewBuilder(t *testing.T) {
	// Use a unique service code to avoid conflicts
	const testService = 80

	builder := NewBuilder(testService, CategoryRequest, 100)
	if builder.service != testService {
		t.Errorf("service = %d, want %d", builder.service, testService)
	}
	if builder.category != CategoryRequest {
		t.Errorf("category = %d, want %d", builder.category, CategoryRequest)
	}
	if builder.sequence != 100 {
		t.Errorf("sequence = %d, want %d", builder.sequence, 100)
	}
}

func TestErrnoBuilderHTTP(t *testing.T) {
	builder := NewBuilder(80, CategoryRequest, 101).
		HTTP(http.StatusTeapot)

	if builder.http != http.StatusTeapot {
		t.Errorf("http = %d, want %d", builder.http, http.StatusTeapot)
	}
}

func TestErrnoBuilderGRPC(t *testing.T) {
	builder := NewBuilder(80, CategoryRequest, 102).
		GRPC(codes.Aborted)

	if builder.grpc != codes.Aborted {
		t.Errorf("grpc = %v, want %v", builder.grpc, codes.Aborted)
	}
}

func TestErrnoBuilderMessage(t *testing.T) {
	builder := NewBuilder(80, CategoryRequest, 103).
		Message("English", "中文")

	if builder.messageEN != "English" {
		t.Errorf("messageEN = %q, want %q", builder.messageEN, "English")
	}
	if builder.messageZH != "中文" {
		t.Errorf("messageZH = %q, want %q", builder.messageZH, "中文")
	}
}

func TestErrnoBuilderMessageEN(t *testing.T) {
	builder := NewBuilder(80, CategoryRequest, 104).
		MessageEN("Only English")

	if builder.messageEN != "Only English" {
		t.Errorf("messageEN = %q, want %q", builder.messageEN, "Only English")
	}
}

func TestErrnoBuilderMessageZH(t *testing.T) {
	builder := NewBuilder(80, CategoryRequest, 105).
		MessageEN("English").
		MessageZH("只有中文")

	if builder.messageZH != "只有中文" {
		t.Errorf("messageZH = %q, want %q", builder.messageZH, "只有中文")
	}
}

func TestErrnoBuilderBuild(t *testing.T) {
	errno, err := NewBuilder(80, CategoryRequest, 106).
		HTTP(http.StatusBadRequest).
		GRPC(codes.InvalidArgument).
		Message("Test error", "测试错误").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedCode := MakeCode(80, CategoryRequest, 106)
	if errno.Code != expectedCode {
		t.Errorf("Code = %d, want %d", errno.Code, expectedCode)
	}
	if errno.HTTP != http.StatusBadRequest {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusBadRequest)
	}
	if errno.GRPCCode != codes.InvalidArgument {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.InvalidArgument)
	}
	if errno.MessageEN != "Test error" {
		t.Errorf("MessageEN = %q, want %q", errno.MessageEN, "Test error")
	}
	if errno.MessageZH != "测试错误" {
		t.Errorf("MessageZH = %q, want %q", errno.MessageZH, "测试错误")
	}

	// Verify it's registered
	if e, ok := Lookup(expectedCode); !ok || e != errno {
		t.Error("Build should register the errno")
	}
}

func TestErrnoBuilderBuildWithoutMessage(t *testing.T) {
	_, err := NewBuilder(80, CategoryRequest, 107).Build()

	if err == nil {
		t.Error("Build() should return error when messageEN is empty")
	}
}

func TestErrnoBuilderBuildDuplicate(t *testing.T) {
	// First build should succeed
	_, err := NewBuilder(80, CategoryRequest, 108).
		Message("First", "第一").
		Build()
	if err != nil {
		t.Fatalf("First Build() error = %v", err)
	}

	// Second build with same code should fail
	_, err = NewBuilder(80, CategoryRequest, 108).
		Message("Second", "第二").
		Build()
	if err == nil {
		t.Error("Build() should return error for duplicate code")
	}
}

func TestErrnoBuilderMustBuild(t *testing.T) {
	errno := NewBuilder(80, CategoryRequest, 109).
		Message("Must build test", "必须构建测试").
		MustBuild()

	if errno == nil {
		t.Error("MustBuild() should return errno")
	}
}

func TestErrnoBuilderMustBuildPanic(t *testing.T) {
	// First registration
	_ = NewBuilder(80, CategoryRequest, 110).
		Message("First", "第一").
		MustBuild()

	// Second should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild should panic on duplicate")
		}
	}()

	_ = NewBuilder(80, CategoryRequest, 110).
		Message("Second", "第二").
		MustBuild()
}

func TestNewRequestError(t *testing.T) {
	errno := NewRequestError(80, 111).
		Message("Request error", "请求错误").
		MustBuild()

	if errno.HTTP != http.StatusBadRequest {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusBadRequest)
	}
	if errno.GRPCCode != codes.InvalidArgument {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.InvalidArgument)
	}
}

func TestNewAuthError(t *testing.T) {
	errno := NewAuthError(80, 112).
		Message("Auth error", "认证错误").
		MustBuild()

	if errno.HTTP != http.StatusUnauthorized {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusUnauthorized)
	}
	if errno.GRPCCode != codes.Unauthenticated {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.Unauthenticated)
	}
}

func TestNewPermissionError(t *testing.T) {
	errno := NewPermissionError(80, 113).
		Message("Permission error", "权限错误").
		MustBuild()

	if errno.HTTP != http.StatusForbidden {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusForbidden)
	}
	if errno.GRPCCode != codes.PermissionDenied {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.PermissionDenied)
	}
}

func TestNewNotFoundError(t *testing.T) {
	errno := NewNotFoundError(80, 114).
		Message("Not found error", "未找到错误").
		MustBuild()

	if errno.HTTP != http.StatusNotFound {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusNotFound)
	}
	if errno.GRPCCode != codes.NotFound {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.NotFound)
	}
}

func TestNewConflictError(t *testing.T) {
	errno := NewConflictError(80, 115).
		Message("Conflict error", "冲突错误").
		MustBuild()

	if errno.HTTP != http.StatusConflict {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusConflict)
	}
	if errno.GRPCCode != codes.AlreadyExists {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.AlreadyExists)
	}
}

func TestNewRateLimitError(t *testing.T) {
	errno := NewRateLimitError(80, 116).
		Message("Rate limit error", "限流错误").
		MustBuild()

	if errno.HTTP != http.StatusTooManyRequests {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusTooManyRequests)
	}
	if errno.GRPCCode != codes.ResourceExhausted {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.ResourceExhausted)
	}
}

func TestNewInternalError(t *testing.T) {
	errno := NewInternalError(80, 117).
		Message("Internal error", "内部错误").
		MustBuild()

	if errno.HTTP != http.StatusInternalServerError {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusInternalServerError)
	}
	if errno.GRPCCode != codes.Internal {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.Internal)
	}
}

func TestNewTimeoutError(t *testing.T) {
	errno := NewTimeoutError(80, 118).
		Message("Timeout error", "超时错误").
		MustBuild()

	if errno.HTTP != http.StatusGatewayTimeout {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusGatewayTimeout)
	}
	if errno.GRPCCode != codes.DeadlineExceeded {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.DeadlineExceeded)
	}
}

func TestQuickCreationFunctions(t *testing.T) {
	// Test NewRequestErr
	err1 := NewRequestErr(81, 1, "Request", "请求")
	if err1.HTTP != http.StatusBadRequest {
		t.Errorf("NewRequestErr HTTP = %d, want %d", err1.HTTP, http.StatusBadRequest)
	}

	// Test NewNotFoundErr
	err2 := NewNotFoundErr(81, 2, "Not found", "未找到")
	if err2.HTTP != http.StatusNotFound {
		t.Errorf("NewNotFoundErr HTTP = %d, want %d", err2.HTTP, http.StatusNotFound)
	}

	// Test NewConflictErr
	err3 := NewConflictErr(81, 3, "Conflict", "冲突")
	if err3.HTTP != http.StatusConflict {
		t.Errorf("NewConflictErr HTTP = %d, want %d", err3.HTTP, http.StatusConflict)
	}

	// Test NewInternalErr
	err4 := NewInternalErr(81, 4, "Internal", "内部")
	if err4.HTTP != http.StatusInternalServerError {
		t.Errorf("NewInternalErr HTTP = %d, want %d", err4.HTTP, http.StatusInternalServerError)
	}
}

// TestNewBuilderBoundaryValidation tests the boundary validation for service, category, and sequence.
func TestNewBuilderBoundaryValidation(t *testing.T) {
	tests := []struct {
		name     string
		service  int
		category int
		sequence int
		wantPanic bool
		panicMsg string
	}{
		{
			name:     "valid_min_values",
			service:  0,
			category: 0,
			sequence: 0,
			wantPanic: false,
		},
		{
			name:     "valid_max_values",
			service:  99,
			category: 99,
			sequence: 999,
			wantPanic: false,
		},
		{
			name:     "service_too_small",
			service:  -1,
			category: 0,
			sequence: 0,
			wantPanic: true,
			panicMsg: "service code must be 0-99",
		},
		{
			name:     "service_too_large",
			service:  100,
			category: 0,
			sequence: 0,
			wantPanic: true,
			panicMsg: "service code must be 0-99",
		},
		{
			name:     "category_too_small",
			service:  0,
			category: -1,
			sequence: 0,
			wantPanic: true,
			panicMsg: "category code must be 0-99",
		},
		{
			name:     "category_too_large",
			service:  0,
			category: 100,
			sequence: 0,
			wantPanic: true,
			panicMsg: "category code must be 0-99",
		},
		{
			name:     "sequence_too_small",
			service:  0,
			category: 0,
			sequence: -1,
			wantPanic: true,
			panicMsg: "sequence must be 0-999",
		},
		{
			name:     "sequence_too_large",
			service:  0,
			category: 0,
			sequence: 1000,
			wantPanic: true,
			panicMsg: "sequence must be 0-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantPanic {
					if r == nil {
						t.Errorf("NewBuilder() should panic for %s", tt.name)
					}
					// Verify panic message contains expected text
					if msg, ok := r.(string); ok {
						if !contains(msg, tt.panicMsg) {
							t.Errorf("Panic message = %q, want to contain %q", msg, tt.panicMsg)
						}
					}
				} else {
					if r != nil {
						t.Errorf("NewBuilder() should not panic for %s, got: %v", tt.name, r)
					}
				}
			}()

			_ = NewBuilder(tt.service, tt.category, tt.sequence)
		})
	}
}

// contains checks if s contains substr (helper for panic message verification).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
