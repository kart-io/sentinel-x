package handler_test

import (
	"context"
	"testing"

	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/stretchr/testify/assert"
)

// TestGRPCServiceInterface 验证 gRPC 服务接口定义
func TestGRPCServiceInterface(t *testing.T) {
	// 验证 UserService 接口方法存在
	t.Run("UserService 接口方法", func(t *testing.T) {
		// 通过类型断言验证接口兼容性
		_ = (v1.UserServiceServer)(nil)

		// 验证服务描述存在
		desc := v1.UserService_ServiceDesc
		assert.Equal(t, "api.user.v1.UserService", desc.ServiceName)

		// 验证 RPC 方法数量
		expectedMethods := []string{
			"GetUser",
			"CreateUser",
			"UpdateUser",
			"DeleteUser",
			"ListUsers",
			"BatchDeleteUsers",
			"ChangePassword",
			"Login",
			"Register",
			"Logout",
		}

		assert.Equal(t, len(expectedMethods), len(desc.Methods))

		// 验证方法名称
		methodNames := make(map[string]bool)
		for _, m := range desc.Methods {
			methodNames[m.MethodName] = true
		}
		for _, expected := range expectedMethods {
			assert.True(t, methodNames[expected], "缺少方法: %s", expected)
		}
	})

	t.Run("RoleService 接口方法", func(t *testing.T) {
		_ = (v1.RoleServiceServer)(nil)

		desc := v1.RoleService_ServiceDesc
		assert.Equal(t, "api.user.v1.RoleService", desc.ServiceName)

		expectedMethods := []string{
			"CreateRole",
			"UpdateRole",
			"DeleteRole",
			"GetRole",
			"ListRoles",
			"AssignRole",
			"GetUserRoles",
		}

		assert.Equal(t, len(expectedMethods), len(desc.Methods))
	})
}

// TestGRPCRequestValidation 测试 gRPC 请求验证
func TestGRPCRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     interface{ Validate() error }
		wantErr bool
	}{
		{
			name: "有效的 LoginRequest",
			req: &v1.LoginRequest{
				Username: "validuser",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "无效的 LoginRequest - 用户名太短",
			req: &v1.LoginRequest{
				Username: "ab",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "无效的 LoginRequest - 密码太短",
			req: &v1.LoginRequest{
				Username: "validuser",
				Password: "123",
			},
			wantErr: true,
		},
		{
			name: "有效的 CreateUserRequest",
			req: &v1.CreateUserRequest{
				Username: "newuser",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "无效的 CreateUserRequest - 邮箱格式错误",
			req: &v1.CreateUserRequest{
				Username: "newuser",
				Password: "password123",
				Email:    "invalid-email",
			},
			wantErr: true,
		},
		{
			name: "有效的 RegisterRequest",
			req: &v1.RegisterRequest{
				Username: "newuser",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGRPCResponseTypes 测试 gRPC 响应类型
func TestGRPCResponseTypes(t *testing.T) {
	t.Run("UserResponse 字段", func(t *testing.T) {
		resp := &v1.UserResponse{
			Id:        "1",
			Username:  "testuser",
			Email:     "test@example.com",
			Mobile:    "13800138000",
			Role:      "admin",
			CreatedAt: 1704067200,
			UpdatedAt: 1704067200,
		}

		assert.Equal(t, "1", resp.GetId())
		assert.Equal(t, "testuser", resp.GetUsername())
		assert.Equal(t, "test@example.com", resp.GetEmail())
		assert.Equal(t, "13800138000", resp.GetMobile())
		assert.Equal(t, "admin", resp.GetRole())
	})

	t.Run("LoginResponse 字段", func(t *testing.T) {
		resp := &v1.LoginResponse{
			Token:    "jwt.token.here",
			ExpireAt: 1704153600,
		}

		assert.Equal(t, "jwt.token.here", resp.GetToken())
		assert.Equal(t, int64(1704153600), resp.GetExpireAt())
	})

	t.Run("ListUsersResponse 字段", func(t *testing.T) {
		resp := &v1.ListUsersResponse{
			Total: 100,
			Items: []*v1.UserResponse{
				{Id: "1", Username: "user1"},
				{Id: "2", Username: "user2"},
			},
		}

		assert.Equal(t, int64(100), resp.GetTotal())
		assert.Len(t, resp.GetItems(), 2)
	})
}

// TestGRPCContextUsage 测试 gRPC 上下文使用
func TestGRPCContextUsage(t *testing.T) {
	ctx := context.Background()

	// 验证上下文可以正常传递
	assert.NotNil(t, ctx)

	// 创建带取消功能的上下文
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 验证取消功能
	cancel()
	assert.Error(t, ctx.Err())
}
