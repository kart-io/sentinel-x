// Package user provides error codes for User Service.
//
// This is an example of how business services should define their error codes
// in their own packages, separate from the core errors package.
package user

import (
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// ============================================================================
// User Service Configuration
// ============================================================================

// ServiceUser is the service code for User Service.
// Service codes 01-09 are reserved for core services.
const ServiceUser = 2

func init() {
	// Register the service to prevent code conflicts
	errors.RegisterService(ServiceUser, "user-service")
}

// ============================================================================
// User Service Request Errors (Category: 01)
// ============================================================================

var (
	// ErrUserInvalidUsername indicates invalid username.
	ErrUserInvalidUsername = errors.NewRequestErr(ServiceUser, 1,
		"Invalid username", "用户名无效")

	// ErrUserInvalidPassword indicates invalid password.
	ErrUserInvalidPassword = errors.NewRequestErr(ServiceUser, 2,
		"Invalid password", "密码无效")

	// ErrUserInvalidEmail indicates invalid email.
	ErrUserInvalidEmail = errors.NewRequestErr(ServiceUser, 3,
		"Invalid email", "邮箱无效")

	// ErrUserInvalidPhone indicates invalid phone number.
	ErrUserInvalidPhone = errors.NewRequestErr(ServiceUser, 4,
		"Invalid phone number", "手机号无效")

	// ErrUserPasswordTooWeak indicates password is too weak.
	ErrUserPasswordTooWeak = errors.NewRequestErr(ServiceUser, 5,
		"Password too weak", "密码强度不足")
)

// ============================================================================
// User Service Authentication Errors (Category: 02)
// ============================================================================

var (
	// ErrUserLoginFailed indicates login failed.
	ErrUserLoginFailed = errors.NewAuthErr(ServiceUser, 1,
		"Login failed", "登录失败")

	// ErrUserPasswordWrong indicates wrong password.
	ErrUserPasswordWrong = errors.NewAuthErr(ServiceUser, 2,
		"Wrong password", "密码错误")

	// ErrUserAccountLocked indicates account is locked.
	ErrUserAccountLocked = errors.NewAuthErr(ServiceUser, 3,
		"Account locked", "账号已锁定")

	// ErrUserTooManyLoginAttempts indicates too many login attempts.
	ErrUserTooManyLoginAttempts = errors.NewRateLimitErr(ServiceUser, 4,
		"Too many login attempts", "登录尝试次数过多")

	// ErrUserVerificationCodeExpired indicates verification code expired.
	ErrUserVerificationCodeExpired = errors.NewAuthErr(ServiceUser, 5,
		"Verification code expired", "验证码已过期")

	// ErrUserVerificationCodeWrong indicates wrong verification code.
	ErrUserVerificationCodeWrong = errors.NewAuthErr(ServiceUser, 6,
		"Wrong verification code", "验证码错误")
)

// ============================================================================
// User Service Permission Errors (Category: 03)
// ============================================================================

var (
	// ErrUserDisabled indicates user is disabled.
	ErrUserDisabled = errors.NewPermissionErr(ServiceUser, 1,
		"User disabled", "用户已禁用")

	// ErrUserNotActivated indicates user is not activated.
	ErrUserNotActivated = errors.NewPermissionErr(ServiceUser, 2,
		"User not activated", "用户未激活")

	// ErrUserNoRole indicates user has no role assigned.
	ErrUserNoRole = errors.NewPermissionErr(ServiceUser, 3,
		"User has no role", "用户未分配角色")
)

// ============================================================================
// User Service Resource Errors (Category: 04)
// ============================================================================

var (
	// ErrUserNotFoundByID indicates user not found by ID.
	ErrUserNotFoundByID = errors.NewNotFoundErr(ServiceUser, 1,
		"User not found", "用户不存在")

	// ErrUserNotFoundByUsername indicates user not found by username.
	ErrUserNotFoundByUsername = errors.NewNotFoundErr(ServiceUser, 2,
		"User not found", "用户不存在")

	// ErrUserNotFoundByEmail indicates user not found by email.
	ErrUserNotFoundByEmail = errors.NewNotFoundErr(ServiceUser, 3,
		"User not found", "用户不存在")

	// ErrRoleNotFound indicates role not found.
	ErrRoleNotFound = errors.NewNotFoundErr(ServiceUser, 4,
		"Role not found", "角色不存在")

	// ErrPermissionNotFound indicates permission not found.
	ErrPermissionNotFound = errors.NewNotFoundErr(ServiceUser, 5,
		"Permission not found", "权限不存在")
)

// ============================================================================
// User Service Conflict Errors (Category: 05)
// ============================================================================

var (
	// ErrUserAlreadyExists indicates user already exists.
	ErrUserAlreadyExists = errors.NewConflictErr(ServiceUser, 1,
		"User already exists", "用户已存在")

	// ErrEmailAlreadyExists indicates email already exists.
	ErrEmailAlreadyExists = errors.NewConflictErr(ServiceUser, 2,
		"Email already exists", "邮箱已被使用")

	// ErrPhoneAlreadyExists indicates phone already exists.
	ErrPhoneAlreadyExists = errors.NewConflictErr(ServiceUser, 3,
		"Phone already exists", "手机号已被使用")

	// ErrUsernameAlreadyExists indicates username already exists.
	ErrUsernameAlreadyExists = errors.NewConflictErr(ServiceUser, 4,
		"Username already exists", "用户名已被使用")

	// ErrRoleAlreadyExists indicates role already exists.
	ErrRoleAlreadyExists = errors.NewConflictErr(ServiceUser, 5,
		"Role already exists", "角色已存在")
)
