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

// Hello 处理公开的 hello 请求（无需认证）。
func (h *DemoHandler) Hello(c transport.Context) {
	resp := response.Success(map[string]string{
		"message": "Hello from Sentinel-X API!",
		"version": "v1",
	})
	defer response.Release(resp)
	c.JSON(http.StatusOK, resp)
}

// Protected 处理需要认证的请求。
func (h *DemoHandler) Protected(c transport.Context) {
	// 调试：检查 request context 中是否有 claims
	ctx := c.Request()
	claims := auth.ClaimsFromContext(ctx)

	// 调试日志
	if claims == nil {
		// 尝试从 HTTP request context 获取
		httpCtx := c.HTTPRequest().Context()
		claims = auth.ClaimsFromContext(httpCtx)
	}

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

// Profile 获取当前用户信息。
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
