package biz

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
)

// TestUserService_PasswordEncryption 测试密码加密功能
func TestUserService_PasswordEncryption(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "普通密码",
			password: "password123",
		},
		{
			name:     "复杂密码",
			password: "P@ssw0rd!#$%^&*()",
		},
		{
			name:     "长密码",
			password: "aVeryLongPasswordThatShouldStillWorkCorrectly123456789",
		},
		{
			name:     "中文密码",
			password: "密码测试123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试密码加密
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)
			assert.NoError(t, err)
			assert.NotEmpty(t, hashedPassword)
			assert.NotEqual(t, tt.password, string(hashedPassword))

			// 测试正确密码验证
			err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(tt.password))
			assert.NoError(t, err)

			// 测试错误密码验证
			err = bcrypt.CompareHashAndPassword(hashedPassword, []byte("wrongPassword"))
			assert.Error(t, err)
		})
	}
}

// TestUserService_PasswordValidation 测试密码验证逻辑
func TestUserService_PasswordValidation(t *testing.T) {
	password := "testPassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		inputPass   string
		storedHash  string
		expectError bool
	}{
		{
			name:        "正确密码",
			inputPass:   password,
			storedHash:  string(hashedPassword),
			expectError: false,
		},
		{
			name:        "错误密码",
			inputPass:   "wrongPassword",
			storedHash:  string(hashedPassword),
			expectError: true,
		},
		{
			name:        "空密码",
			inputPass:   "",
			storedHash:  string(hashedPassword),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bcrypt.CompareHashAndPassword([]byte(tt.storedHash), []byte(tt.inputPass))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUserService_PasswordChange 测试密码修改逻辑
func TestUserService_PasswordChange(t *testing.T) {
	oldPassword := "oldPassword123"
	newPassword := "newPassword456"

	// 模拟旧密码哈希
	oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	// 验证旧密码正确
	err = bcrypt.CompareHashAndPassword(oldHash, []byte(oldPassword))
	assert.NoError(t, err)

	// 生成新密码哈希
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	// 验证新密码
	err = bcrypt.CompareHashAndPassword(newHash, []byte(newPassword))
	assert.NoError(t, err)

	// 验证新旧密码哈希不同
	assert.NotEqual(t, string(oldHash), string(newHash))

	// 验证旧密码不能通过新哈希验证
	err = bcrypt.CompareHashAndPassword(newHash, []byte(oldPassword))
	assert.Error(t, err)
}

// TestPasswordHashUniqueness 测试每次哈希结果不同
func TestPasswordHashUniqueness(t *testing.T) {
	password := "samePassword123"

	hash1, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	hash2, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	// 同一密码的两次哈希应该不同（因为使用了随机盐）
	assert.NotEqual(t, string(hash1), string(hash2))

	// 但两者都应该能验证成功
	assert.NoError(t, bcrypt.CompareHashAndPassword(hash1, []byte(password)))
	assert.NoError(t, bcrypt.CompareHashAndPassword(hash2, []byte(password)))
}
