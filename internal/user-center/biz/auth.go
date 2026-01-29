package biz

import (
	"context"
	stderrors "errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// AuthService 处理认证业务逻辑
type AuthService struct {
	jwtAuth   *jwt.JWT
	userStore *store.UserStore
	logStore  *store.LogStore
	redis     *redis.Client
}

// NewAuthService 创建新的 AuthService
func NewAuthService(jwtAuth *jwt.JWT, userStore *store.UserStore, logStore *store.LogStore, redis *redis.Client) *AuthService {
	return &AuthService{
		jwtAuth:   jwtAuth,
		userStore: userStore,
		logStore:  logStore,
		redis:     redis,
	}
}

// Login 用户登录并返回访问令牌
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest, userAgent, ip string) (*model.LoginResponse, error) {
	// 1. 记录登录日志
	log := &model.LoginLog{
		Username:  req.Username,
		UserAgent: userAgent,
		IP:        ip,
		Status:    0,
	}
	defer func() {
		if err := s.logStore.Create(ctx, log); err != nil {
			// 记录日志失败不应影响登录流程，仅打印日志
			// logger.Errorf("Failed to create login log: %v", err)
		}
	}()

	// 2. 检查验证码
	// TODO: 在 Handler 层调用 GetCaptcha 后，前端提交时带上 captcha_id 和 captcha_code
	// 这里假设 req 中包含验证码信息，需要修改 model.LoginRequest 或新增参数
	// 暂时跳过验证码校验，后续补充

	// 3. 检查账号锁定状态
	lockKey := "login_lock:" + req.Username
	if exists, _ := s.redis.Client().Exists(ctx, lockKey).Result(); exists > 0 {
		log.Message = "Account locked"
		return nil, errors.ErrTooManyRequests.WithMessage("账号已被锁定，请稍后再试")
	}

	// 4. 获取用户信息
	user, err := s.userStore.Get(ctx, req.Username)
	if err != nil {
		log.Message = "User not found"
		return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
	}
	log.UserID = user.ID

	// 5. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// 记录失败次数
		s.handleLoginValidFailed(ctx, req.Username)
		log.Message = "Invalid password"
		return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
	}

	// 6. 检查用户状态
	if user.Status == 0 {
		log.Message = "Account disabled"
		return nil, errors.ErrAccountDisabled.WithMessage("账号已被禁用")
	}

	// 7. 登录成功，清除失败次数
	s.redis.Client().Del(ctx, "login_fail:" + req.Username)

	// 8. 生成访问令牌 (Access Token)
	accessToken, err := s.jwtAuth.Sign(ctx, req.Username, auth.WithExtra(map[string]interface{}{
		"id": user.ID,
	}))
	if err != nil {
		log.Message = "Generate token failed"
		return nil, errors.ErrInternal.WithCause(err)
	}

	// 9. 生成刷新令牌 (Refresh Token)
	// 刷新令牌有效期通常比访问令牌长，例如 7 天
	refreshToken, err := s.jwtAuth.Sign(ctx, req.Username, auth.WithExtra(map[string]interface{}{
		"id": user.ID,
		"type": "refresh",
	}), auth.WithExpiresAt(time.Now().Add(7*24*time.Hour)))
	if err != nil {
		log.Message = "Generate refresh token failed"
		return nil, errors.ErrInternal.WithCause(err)
	}

	log.Status = 1
	log.Message = "Success"

	return &model.LoginResponse{
		Token:        accessToken.GetAccessToken(),
		ExpiresIn:    accessToken.GetExpiresAt(),
		RefreshToken: refreshToken.GetAccessToken(),
		UserID:       user.ID,
	}, nil
}

// handleLoginValidFailed 处理登录失败逻辑
func (s *AuthService) handleLoginValidFailed(ctx context.Context, username string) {
	failKey := "login_fail:" + username
	lockKey := "login_lock:" + username

	// Increment failure count
	count, _ := s.redis.Client().Incr(ctx, failKey).Result()
	s.redis.Client().Expire(ctx, failKey, 10*time.Minute)

	if count >= 5 {
		// Lock account for 30 minutes
		s.redis.Client().Set(ctx, lockKey, 1, 30*time.Minute)
		s.redis.Client().Del(ctx, failKey)
	}
}

// RefreshToken 刷新访问令牌
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*model.LoginResponse, error) {
	// 验证刷新令牌
	newAccessToken, err := s.jwtAuth.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, errors.ErrUnauthorized.WithCause(err).WithMessage("无效的刷新令牌")
	}

	// 解析声明以获取用户ID
	// 这里简化处理，直接返回新令牌
	return &model.LoginResponse{
		Token:     newAccessToken.GetAccessToken(),
		ExpiresIn: newAccessToken.GetExpiresAt(),
		RefreshToken: refreshToken, // 保持原刷新令牌，或者也可以轮换刷新令牌
	}, nil
}

// GetCaptcha 获取验证码 (Mock Implementation)
func (s *AuthService) GetCaptcha(ctx context.Context) (string, string, error) {
	// Mock implementation due to network restriction on base64Captcha
	id := "mock-captcha-id"
	b64s := "data:image/png;base64,mock-captcha-image"
	// Store fixed answer "1234" for testing
	s.redis.Client().Set(ctx, "captcha:"+id, "1234", 10*time.Minute)
	return id, b64s, nil
}

// VerifyCaptcha 验证验证码 (Mock Implementation)
func (s *AuthService) VerifyCaptcha(ctx context.Context, id, answer string) bool {
    // For mock id, check against fixed answer or redis
    if id == "mock-captcha-id" {
        return answer == "1234"
    }
	val, err := s.redis.Client().Get(ctx, "captcha:"+id).Result()
    if err != nil {
        return false
    }
    return val == answer
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
	existingUser, err := s.userStore.Get(ctx, req.Username)
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

	return s.userStore.Create(ctx, user)
}
