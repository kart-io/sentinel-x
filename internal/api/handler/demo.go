// Package handler 提供 API 服务的 HTTP 处理器。
package handler

import (
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// DemoHandler 处理演示 API 请求。
type DemoHandler struct{}

// NewDemoHandler 创建新的 DemoHandler。
func NewDemoHandler() *DemoHandler {
	return &DemoHandler{}
}

// HelloResponse 是 Hello API 的响应结构。
type HelloResponse struct {
	Message string `json:"message" example:"Hello from Sentinel-X API!"`
	Version string `json:"version" example:"v1"`
}

// ProtectedResponse 是受保护 API 的响应结构。
type ProtectedResponse struct {
	Message  string   `json:"message" example:"This is a protected resource"`
	UserID   string   `json:"user_id" example:"user-123"`
	Username string   `json:"username" example:"admin"`
	Roles    []string `json:"roles" example:"admin,user"`
}

// ProfileResponse 是用户 Profile API 的响应结构。
type ProfileResponse struct {
	ID        string   `json:"id" example:"user-123"`
	Username  string   `json:"username" example:"admin"`
	Roles     []string `json:"roles" example:"admin,user"`
	Issuer    string   `json:"issuer" example:"sentinel-x"`
	ExpiresAt int64    `json:"expires_at" example:"1735689600"`
	IssuedAt  int64    `json:"issued_at" example:"1735603200"`
}

// Hello godoc
//
//	@Summary		Say Hello
//	@Description	公开接口，返回欢迎消息（无需认证）
//	@Tags			Demo
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=HelloResponse}	"成功响应"
//	@Router			/hello [get]
func (h *DemoHandler) Hello(c transport.Context) {
	resp := response.Success(map[string]string{
		"message": "Hello from Sentinel-X API!",
		"version": "v1",
	})
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// Protected godoc
//
//	@Summary		Protected Resource
//	@Description	需要 JWT 认证的受保护资源
//	@Tags			Demo
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	response.Response{data=ProtectedResponse}	"成功响应"
//	@Failure		401	{object}	response.Response							"未授权"
//	@Router			/protected [get]
func (h *DemoHandler) Protected(c transport.Context) {
	claims := auth.ClaimsFromContext(c.Request())
	if claims == nil {
		resp := response.ErrorWithCode(401, "unauthorized")
		defer response.Release(resp)
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	resp := response.Success(map[string]interface{}{
		"message":  "This is a protected resource",
		"user_id":  claims.Subject,
		"username": claims.GetExtraString("username"),
		"roles":    claims.Extra["roles"],
	})
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// Profile godoc
//
//	@Summary		Get Current User Profile
//	@Description	获取当前登录用户的详细信息
//	@Tags			Demo
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	response.Response{data=ProfileResponse}	"成功响应"
//	@Failure		401	{object}	response.Response						"未授权"
//	@Router			/profile [get]
func (h *DemoHandler) Profile(c transport.Context) {
	claims := auth.ClaimsFromContext(c.Request())
	if claims == nil {
		resp := response.ErrorWithCode(401, "unauthorized")
		defer response.Release(resp)
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	resp := response.Success(map[string]interface{}{
		"id":         claims.Subject,
		"username":   claims.GetExtraString("username"),
		"roles":      claims.Extra["roles"],
		"issuer":     claims.Issuer,
		"expires_at": claims.ExpiresAt,
		"issued_at":  claims.IssuedAt,
	})
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}
