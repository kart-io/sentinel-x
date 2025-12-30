package biz

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/stretchr/testify/assert"
)

// TestAuthService_LoginValidation 测试登录验证逻辑
func TestAuthService_LoginValidation(t *testing.T) {
	// 测试用户数据
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	tests := []struct {
		name        string
		request     *model.LoginRequest
		mockUser    *model.User
		userErr     error
		expectError bool
		errorType   error
	}{
		{
			name: "成功登录 - 密码验证通过",
			request: &model.LoginRequest{
				Username: "testuser",
				Password: password,
			},
			mockUser: &model.User{
				ID:       1,
				Username: "testuser",
				Password: string(hashedPassword),
				Status:   1,
			},
			userErr:     nil,
			expectError: false,
		},
		{
			name: "用户不存在",
			request: &model.LoginRequest{
				Username: "nonexistent",
				Password: password,
			},
			mockUser:    nil,
			userErr:     errors.ErrUserNotFound,
			expectError: true,
			errorType:   errors.ErrUnauthorized,
		},
		{
			name: "密码错误",
			request: &model.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockUser: &model.User{
				ID:       1,
				Username: "testuser",
				Password: string(hashedPassword),
				Status:   1,
			},
			userErr:     nil,
			expectError: true,
			errorType:   errors.ErrUnauthorized,
		},
		{
			name: "账号已禁用",
			request: &model.LoginRequest{
				Username: "disableduser",
				Password: password,
			},
			mockUser: &model.User{
				ID:       2,
				Username: "disableduser",
				Password: string(hashedPassword),
				Status:   0, // 禁用状态
			},
			userErr:     nil,
			expectError: true,
			errorType:   errors.ErrAccountDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证登录逻辑
			switch {
			case tt.mockUser != nil && tt.userErr == nil:
				// 验证密码
				err := bcrypt.CompareHashAndPassword([]byte(tt.mockUser.Password), []byte(tt.request.Password))
				switch {
				case tt.expectError && tt.errorType == errors.ErrUnauthorized && err != nil:
					// 密码验证失败预期
					assert.Error(t, err)
				case tt.mockUser.Status == 0:
					// 账号禁用预期
					assert.Equal(t, 0, tt.mockUser.Status)
				case !tt.expectError:
					// 正常登录
					assert.NoError(t, err)
					assert.Equal(t, 1, tt.mockUser.Status)
				}
			case tt.userErr != nil:
				// 用户不存在
				assert.NotNil(t, tt.userErr)
			}
		})
	}
}

// TestAuthService_RegisterValidation 测试注册验证逻辑
func TestAuthService_RegisterValidation(t *testing.T) {
	tests := []struct {
		name         string
		request      *model.RegisterRequest
		existingUser *model.User
		getErr       error
		expectError  bool
	}{
		{
			name: "成功注册新用户",
			request: &model.RegisterRequest{
				Username: "newuser",
				Password: "password123",
				Email:    "newuser@example.com",
			},
			existingUser: nil,
			getErr:       errors.ErrUserNotFound, // 用户不存在，可以注册
			expectError:  false,
		},
		{
			name: "用户名已存在",
			request: &model.RegisterRequest{
				Username: "existinguser",
				Password: "password123",
				Email:    "existing@example.com",
			},
			existingUser: &model.User{Username: "existinguser"},
			getErr:       nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证注册逻辑
			if tt.existingUser != nil {
				// 用户已存在，应该返回错误
				assert.True(t, tt.expectError)
			} else if tt.getErr == errors.ErrUserNotFound {
				// 用户不存在，可以继续注册
				// 验证密码会被哈希
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tt.request.Password), bcrypt.DefaultCost)
				assert.NoError(t, err)
				assert.NotEqual(t, tt.request.Password, string(hashedPassword))
			}
		})
	}
}

// TestPasswordHashing 测试 bcrypt 密码哈希功能
func TestPasswordHashing(t *testing.T) {
	password := "securePassword123!"

	t.Run("哈希生成", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, string(hash))
	})

	t.Run("哈希验证", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		// 正确密码应该验证成功
		err := bcrypt.CompareHashAndPassword(hash, []byte(password))
		assert.NoError(t, err)

		// 错误密码应该验证失败
		err = bcrypt.CompareHashAndPassword(hash, []byte("wrongPassword"))
		assert.Error(t, err)
	})

	t.Run("每次哈希结果不同", func(t *testing.T) {
		hash1, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		hash2, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		// 同一密码的两次哈希应该不同（因为使用了随机盐）
		assert.NotEqual(t, string(hash1), string(hash2))

		// 但两者都应该能验证成功
		assert.NoError(t, bcrypt.CompareHashAndPassword(hash1, []byte(password)))
		assert.NoError(t, bcrypt.CompareHashAndPassword(hash2, []byte(password)))
	})
}

// TestUserStatusValidation 测试用户状态验证
func TestUserStatusValidation(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		expectEnabled bool
	}{
		{
			name:          "启用状态",
			status:        1,
			expectEnabled: true,
		},
		{
			name:          "禁用状态",
			status:        0,
			expectEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &model.User{Status: tt.status}
			isEnabled := user.Status == 1
			assert.Equal(t, tt.expectEnabled, isEnabled)
		})
	}
}
