package errors

import (
	"net/http"
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

func TestQuickCreationFunctions(t *testing.T) {
	// Test NewRequestErr
	err1 := NewRequestErr(81, 1, "Request", "请求")
	if err1.HTTP != http.StatusBadRequest {
		t.Errorf("NewRequestErr HTTP = %d, want %d", err1.HTTP, http.StatusBadRequest)
	}
	if err1.GRPCCode != codes.InvalidArgument {
		t.Errorf("NewRequestErr GRPCCode = %v, want %v", err1.GRPCCode, codes.InvalidArgument)
	}

	// Test NewAuthErr
	err2 := NewAuthErr(81, 2, "Auth error", "认证错误")
	if err2.HTTP != http.StatusUnauthorized {
		t.Errorf("NewAuthErr HTTP = %d, want %d", err2.HTTP, http.StatusUnauthorized)
	}
	if err2.GRPCCode != codes.Unauthenticated {
		t.Errorf("NewAuthErr GRPCCode = %v, want %v", err2.GRPCCode, codes.Unauthenticated)
	}

	// Test NewPermissionErr
	err3 := NewPermissionErr(81, 3, "Permission error", "权限错误")
	if err3.HTTP != http.StatusForbidden {
		t.Errorf("NewPermissionErr HTTP = %d, want %d", err3.HTTP, http.StatusForbidden)
	}
	if err3.GRPCCode != codes.PermissionDenied {
		t.Errorf("NewPermissionErr GRPCCode = %v, want %v", err3.GRPCCode, codes.PermissionDenied)
	}

	// Test NewNotFoundErr
	err4 := NewNotFoundErr(81, 4, "Not found", "未找到")
	if err4.HTTP != http.StatusNotFound {
		t.Errorf("NewNotFoundErr HTTP = %d, want %d", err4.HTTP, http.StatusNotFound)
	}
	if err4.GRPCCode != codes.NotFound {
		t.Errorf("NewNotFoundErr GRPCCode = %v, want %v", err4.GRPCCode, codes.NotFound)
	}

	// Test NewConflictErr
	err5 := NewConflictErr(81, 5, "Conflict", "冲突")
	if err5.HTTP != http.StatusConflict {
		t.Errorf("NewConflictErr HTTP = %d, want %d", err5.HTTP, http.StatusConflict)
	}
	if err5.GRPCCode != codes.AlreadyExists {
		t.Errorf("NewConflictErr GRPCCode = %v, want %v", err5.GRPCCode, codes.AlreadyExists)
	}

	// Test NewRateLimitErr
	err6 := NewRateLimitErr(81, 6, "Rate limit", "限流")
	if err6.HTTP != http.StatusTooManyRequests {
		t.Errorf("NewRateLimitErr HTTP = %d, want %d", err6.HTTP, http.StatusTooManyRequests)
	}
	if err6.GRPCCode != codes.ResourceExhausted {
		t.Errorf("NewRateLimitErr GRPCCode = %v, want %v", err6.GRPCCode, codes.ResourceExhausted)
	}

	// Test NewInternalErr
	err7 := NewInternalErr(81, 7, "Internal", "内部")
	if err7.HTTP != http.StatusInternalServerError {
		t.Errorf("NewInternalErr HTTP = %d, want %d", err7.HTTP, http.StatusInternalServerError)
	}
	if err7.GRPCCode != codes.Internal {
		t.Errorf("NewInternalErr GRPCCode = %v, want %v", err7.GRPCCode, codes.Internal)
	}

	// Test NewDatabaseErr
	err8 := NewDatabaseErr(81, 8, "Database", "数据库")
	if err8.HTTP != http.StatusInternalServerError {
		t.Errorf("NewDatabaseErr HTTP = %d, want %d", err8.HTTP, http.StatusInternalServerError)
	}

	// Test NewCacheErr
	err9 := NewCacheErr(81, 9, "Cache", "缓存")
	if err9.HTTP != http.StatusInternalServerError {
		t.Errorf("NewCacheErr HTTP = %d, want %d", err9.HTTP, http.StatusInternalServerError)
	}

	// Test NewNetworkErr
	err10 := NewNetworkErr(81, 10, "Network", "网络")
	if err10.HTTP != http.StatusServiceUnavailable {
		t.Errorf("NewNetworkErr HTTP = %d, want %d", err10.HTTP, http.StatusServiceUnavailable)
	}
	if err10.GRPCCode != codes.Unavailable {
		t.Errorf("NewNetworkErr GRPCCode = %v, want %v", err10.GRPCCode, codes.Unavailable)
	}

	// Test NewTimeoutErr
	err11 := NewTimeoutErr(81, 11, "Timeout", "超时")
	if err11.HTTP != http.StatusGatewayTimeout {
		t.Errorf("NewTimeoutErr HTTP = %d, want %d", err11.HTTP, http.StatusGatewayTimeout)
	}
	if err11.GRPCCode != codes.DeadlineExceeded {
		t.Errorf("NewTimeoutErr GRPCCode = %v, want %v", err11.GRPCCode, codes.DeadlineExceeded)
	}

	// Test NewConfigErr
	err12 := NewConfigErr(81, 12, "Config", "配置")
	if err12.HTTP != http.StatusInternalServerError {
		t.Errorf("NewConfigErr HTTP = %d, want %d", err12.HTTP, http.StatusInternalServerError)
	}
}

func TestNewError(t *testing.T) {
	// Test custom error creation with NewError
	errno := NewError(82, CategoryRequest, 1, http.StatusTeapot, codes.Aborted, "Custom error", "自定义错误")

	expectedCode := MakeCode(82, CategoryRequest, 1)
	if errno.Code != expectedCode {
		t.Errorf("Code = %d, want %d", errno.Code, expectedCode)
	}
	if errno.HTTP != http.StatusTeapot {
		t.Errorf("HTTP = %d, want %d", errno.HTTP, http.StatusTeapot)
	}
	if errno.GRPCCode != codes.Aborted {
		t.Errorf("GRPCCode = %v, want %v", errno.GRPCCode, codes.Aborted)
	}
	if errno.MessageEN != "Custom error" {
		t.Errorf("MessageEN = %q, want %q", errno.MessageEN, "Custom error")
	}
	if errno.MessageZH != "自定义错误" {
		t.Errorf("MessageZH = %q, want %q", errno.MessageZH, "自定义错误")
	}

	// Verify it's registered
	if e, ok := Lookup(expectedCode); !ok || e != errno {
		t.Error("NewError should register the errno")
	}
}

func TestNewErrorDuplicate(t *testing.T) {
	// First registration should succeed
	_ = NewRequestErr(83, 1, "First", "第一")

	// Second registration with same code should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewError should panic on duplicate code")
		}
	}()

	_ = NewRequestErr(83, 1, "Second", "第二")
}

func TestNewErrorEmptyMessage(t *testing.T) {
	// Should panic when messageEN is empty
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewError should panic when messageEN is empty")
		}
	}()

	_ = NewError(84, CategoryRequest, 1, http.StatusBadRequest, codes.InvalidArgument, "", "")
}

func TestNewErrorBoundaryValidation(t *testing.T) {
	tests := []struct {
		name      string
		service   int
		category  int
		sequence  int
		wantPanic bool
	}{
		{
			name:      "valid_min_values",
			service:   85,
			category:  1,
			sequence:  100,
			wantPanic: false,
		},
		{
			name:      "valid_max_values",
			service:   96,
			category:  98,
			sequence:  998,
			wantPanic: false,
		},
		{
			name:      "service_too_small",
			service:   -1,
			category:  0,
			sequence:  0,
			wantPanic: true,
		},
		{
			name:      "service_too_large",
			service:   100,
			category:  0,
			sequence:  0,
			wantPanic: true,
		},
		{
			name:      "category_too_small",
			service:   86,
			category:  -1,
			sequence:  100,
			wantPanic: true,
		},
		{
			name:      "category_too_large",
			service:   87,
			category:  100,
			sequence:  100,
			wantPanic: true,
		},
		{
			name:      "sequence_too_small",
			service:   88,
			category:  1,
			sequence:  -1,
			wantPanic: true,
		},
		{
			name:      "sequence_too_large",
			service:   89,
			category:  1,
			sequence:  1000,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantPanic {
					if r == nil {
						t.Errorf("NewError() should panic for %s", tt.name)
					}
				} else {
					if r != nil {
						t.Errorf("NewError() should not panic for %s, got: %v", tt.name, r)
					}
				}
			}()

			_ = NewError(tt.service, tt.category, tt.sequence, http.StatusBadRequest, codes.InvalidArgument, "Test", "测试")
		})
	}
}
