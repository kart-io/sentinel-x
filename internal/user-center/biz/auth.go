package biz

import (
	"context"
	stderrors "errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// AuthService 处理认证业务逻辑
type AuthService struct {
	jwtAuth *jwt.JWT
	store   store.Factory
}

// NewAuthService 创建新的 AuthService
func NewAuthService(jwtAuth *jwt.JWT, store store.Factory) *AuthService {
	return &AuthService{
		jwtAuth: jwtAuth,
		store:   store,
	}
}

// Login 用户登录并返回访问令牌
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// 获取用户信息
	user, err := s.store.Users().Get(ctx, req.Username)
	if err != nil {
		return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
	}

	// 检查用户状态
	if user.Status == 0 {
		return nil, errors.ErrAccountDisabled.WithMessage("账号已被禁用")
	}

	// 生成访问令牌
	token, err := s.jwtAuth.Sign(ctx, req.Username, auth.WithExtra(map[string]interface{}{
		"id": user.ID,
	}))
	if err != nil {
		return nil, errors.ErrInternal.WithCause(err)
	}

	return &model.LoginResponse{
		Token:     token.GetAccessToken(),
		ExpiresIn: token.GetExpiresAt(),
		UserID:    user.ID,
	}, nil
}

// Logout 撤销用户令牌
func (s *AuthService) Logout(ctx context.Context, token string) error {
	if err := s.jwtAuth.Revoke(ctx, token); err != nil {
		return errors.ErrInternal.WithCause(err)
	}
	return nil
}

// Register 注册新用户
func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest) error {
	// 检查用户名是否已存在
	existingUser, err := s.store.Users().Get(ctx, req.Username)
	if err != nil {
		// 区分"用户不存在"和"数据库错误"两种情况
		if !stderrors.Is(err, errors.ErrUserNotFound) {
			return errors.ErrInternal.WithCause(err)
		}
		// 用户不存在,可以继续注册
	} else if existingUser != nil {
		return errors.ErrAlreadyExists.WithMessage("用户名已存在")
	}

	// 对密码进行哈希处理
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.ErrInternal.WithCause(err)
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    &req.Email,
		Mobile:   req.Mobile,
		Status:   1,
	}

	return s.store.Users().Create(ctx, user)
}
