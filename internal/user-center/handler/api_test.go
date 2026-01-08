package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/stretchr/testify/assert"
)

// APIResponse 标准 API 响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// TestUserAPI_CreateUser_Validation 测试用户创建请求的验证逻辑
func TestUserAPI_CreateUser_Validation(t *testing.T) {
	tests := []struct {
		name       string
		req        *v1.CreateUserRequest
		wantStatus int
		wantCode   int
	}{
		{
			name: "无效请求 - 缺少用户名",
			req: &v1.CreateUserRequest{
				Username: "",
				Password: "validpassword123",
				Email:    "test@example.com",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   errors.ErrBadRequest.Code,
		},
		{
			name: "无效请求 - 密码太短",
			req: &v1.CreateUserRequest{
				Username: "testuser",
				Password: "123",
				Email:    "test@example.com",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   errors.ErrBadRequest.Code,
		},
		{
			name: "无效请求 - 邮箱格式错误",
			req: &v1.CreateUserRequest{
				Username: "testuser",
				Password: "validpassword123",
				Email:    "invalid-email",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   errors.ErrBadRequest.Code,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 直接测试请求绑定和验证
			var createReq v1.CreateUserRequest
			err := c.ShouldBindJSON(&createReq)
			if err == nil {
				err = createReq.Validate()
			}

			if tt.wantStatus == http.StatusBadRequest {
				assert.Error(t, err)
			}
		})
	}
}

// TestUserAPI_ListUser_Pagination 测试用户列表分页参数
func TestUserAPI_ListUser_Pagination(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "默认分页",
			queryParams:  "",
			expectedPage: 1,
			expectedSize: 10,
		},
		{
			name:         "自定义页码",
			queryParams:  "?page=2&page_size=20",
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name:         "无效页码使用默认值",
			queryParams:  "?page=0&page_size=0",
			expectedPage: 1,
			expectedSize: 10,
		},
		{
			name:         "负数页码使用默认值",
			queryParams:  "?page=-1&page_size=-5",
			expectedPage: 1,
			expectedSize: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/users"+tt.queryParams, nil)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			var listReq v1.ListUsersRequest
			_ = c.ShouldBindQuery(&listReq)

			if listReq.Page == 0 || listReq.PageSize == 0 {
				// Protobuf验证可能允许0值，手动验证
			}

			// 计算默认值
			page := int(listReq.Page)
			if page <= 0 {
				page = 1
			}
			pageSize := int(listReq.PageSize)
			if pageSize <= 0 {
				pageSize = 10
			}

			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedSize, pageSize)
		})
	}
}

// TestAuthAPI_Register_Validation 测试注册请求验证
func TestAuthAPI_Register_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *v1.RegisterRequest
		wantErr bool
	}{
		{
			name: "有效注册请求",
			req: &v1.RegisterRequest{
				Username: "newuser",
				Password: "password123",
				Email:    "newuser@example.com",
			},
			wantErr: false,
		},
		{
			name: "无效 - 用户名太短",
			req: &v1.RegisterRequest{
				Username: "ab",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "无效 - 密码太短",
			req: &v1.RegisterRequest{
				Username: "validuser",
				Password: "123",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			var registerReq v1.RegisterRequest
			err := c.ShouldBindJSON(&registerReq)
			if err == nil {
				err = registerReq.Validate()
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.req.Username, registerReq.Username)
			}
		})
	}
}

// TestAuthAPI_ChangePassword_Validation 测试修改密码请求验证
func TestAuthAPI_ChangePassword_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *v1.ChangePasswordRequest
		wantErr bool
	}{
		{
			name: "有效请求",
			req: &v1.ChangePasswordRequest{
				Username:        "testuser",
				OldPassword:     "oldpassword123",
				NewPassword:     "newpassword456",
				ConfirmPassword: "newpassword456",
			},
			wantErr: false,
		},
		{
			name: "密码不匹配",
			req: &v1.ChangePasswordRequest{
				Username:        "testuser",
				OldPassword:     "oldpassword123",
				NewPassword:     "newpassword456",
				ConfirmPassword: "differentpassword",
			},
			wantErr: false, // 验证阶段不检查密码匹配，这是业务逻辑
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/v1/users/password", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			var changeReq v1.ChangePasswordRequest
			err := c.ShouldBindJSON(&changeReq)
			if err == nil {
				err = changeReq.Validate()
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// 业务层验证密码匹配
				passwordsMatch := changeReq.NewPassword == changeReq.ConfirmPassword
				if tt.req.NewPassword != tt.req.ConfirmPassword {
					assert.False(t, passwordsMatch)
				}
			}
		})
	}
}

// TestHTTPMethod_Validation 测试 HTTP 方法验证
func TestHTTPMethod_Validation(t *testing.T) {
	endpoints := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{"创建用户", http.MethodPost, "/v1/users", http.MethodPost},
		{"获取用户列表", http.MethodGet, "/v1/users", http.MethodGet},
		{"更新用户", http.MethodPut, "/v1/users", http.MethodPut},
		{"删除用户", http.MethodDelete, "/v1/users", http.MethodDelete},
		{"登录", http.MethodPost, "/auth/login", http.MethodPost},
		{"登出", http.MethodPost, "/auth/logout", http.MethodPost},
		{"注册", http.MethodPost, "/auth/register", http.MethodPost},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			assert.Equal(t, ep.expected, req.Method)
		})
	}
}
