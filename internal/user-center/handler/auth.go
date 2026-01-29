package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/pkg/httputils"
	"github.com/kart-io/sentinel-x/internal/user-center/biz"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	svc *biz.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *biz.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt int64  `json:"expires_at" example:"1735689600"`
}

// Login godoc
//
//	@Summary		用户登录
//	@Description	通过用户名和密码登录，获取 JWT token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.LoginRequest							true	"登录请求"
//	@Success		200		{object}	LoginResponse	"成功响应"
//	@Failure		401		{object}	object			"认证失败"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

    // Verify Captcha
    if req.CaptchaID != "" && req.CaptchaCode != "" {
        if !h.svc.VerifyCaptcha(c.Request.Context(), req.CaptchaID, req.CaptchaCode) {
            httputils.WriteResponse(c, errors.ErrInvalidParam.WithMessage("验证码错误"), nil)
            return
        }
    }

	respData, err := h.svc.Login(c.Request.Context(), &model.LoginRequest{
		Username:    req.Username,
		Password:    req.Password,
        CaptchaID:   req.CaptchaID,
        CaptchaCode: req.CaptchaCode,
	}, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		logger.Warnf("Login failed: %v", err)
		httputils.WriteResponse(c, err, nil)
		return
	}

	httputils.WriteResponse(c, nil, respData)
}

// RefreshToken godoc
//
//	@Summary		刷新令牌
//	@Description	使用刷新令牌获取新的访问令牌
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			refresh_token	body		string	true	"刷新令牌"
//	@Success		200				{object}	LoginResponse	"成功响应"
//	@Failure		401				{object}	object			"认证失败"
//	@Router			/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
    type RefreshRequest struct {
        RefreshToken string `json:"refresh_token" binding:"required"`
    }
    var req RefreshRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
        return
    }

    respData, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
    if err != nil {
        logger.Warnf("RefreshToken failed: %v", err)
        httputils.WriteResponse(c, err, nil)
        return
    }

    httputils.WriteResponse(c, nil, respData)
}

// GetCaptcha godoc
//
//	@Summary		获取验证码
//	@Description	获取图形验证码
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	object	"成功响应"
//	@Router			/auth/captcha [get]
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
    id, b64s, err := h.svc.GetCaptcha(c.Request.Context())
    if err != nil {
        httputils.WriteResponse(c, errors.ErrInternal.WithMessage("获取验证码失败"), nil)
        return
    }
    httputils.WriteResponse(c, nil, gin.H{
        "captcha_id":   id,
        "captcha_code": b64s,
    })
}

// Logout godoc
//
//	@Summary		用户登出
//	@Description	使当前 JWT token 失效
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer token"
//	@Success		200				{object}	object	"成功响应"
//	@Failure		400				{object}	object	"请求错误"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && strings.ToUpper(token[:7]) == "BEARER " {
		token = token[7:]
	}

	if msg := c.Query("token"); msg != "" && token == "" {
		token = msg
	}

	if token == "" {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage("token required"), nil)
		return
	}

	if err := h.svc.Logout(c.Request.Context(), token); err != nil {
		logger.Errorf("Logout failed: %v", err)
		httputils.WriteResponse(c, errors.ErrInternal.WithMessage("failed to logout"), nil)
		return
	}

	httputils.WriteResponse(c, nil, "logged out")
}

// Register godoc
//
//	@Summary		用户注册
//	@Description	创建新用户账号
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		v1.RegisterRequest	true	"注册请求"
//	@Success		200		{object}	object	"成功响应"
//	@Failure		400		{object}	object	"请求错误"
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req v1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	if err := validator.Global().Validate(&req); err != nil {
		httputils.WriteResponse(c, errors.ErrValidationFailed.WithMessage(err.Error()), nil)
		return
	}

	if err := h.svc.Register(c.Request.Context(), &model.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Mobile:   req.Mobile,
	}); err != nil {
		logger.Errorf("Register failed: %v", err)
		httputils.WriteResponse(c, errors.ErrInternal.WithMessage(err.Error()), nil)
		return
	}

	httputils.WriteResponse(c, nil, "user registered")
}
