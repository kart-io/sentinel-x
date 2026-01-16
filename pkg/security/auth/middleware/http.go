package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin"
)

// Authorizer provides middleware for authorization
type Authorizer struct {
	service casbin.PermissionService
}

// NewAuthorizer creates a new Authorizer
func NewAuthorizer(service casbin.PermissionService) *Authorizer {
	return &Authorizer{service: service}
}

// GinMiddleware returns a Gin middleware for authorization
// subFn: function to extract subject (user) from context
// objFn: function to extract object (resource) from context
// actFn: function to extract action (method) from context
func (a *Authorizer) GinMiddleware(
	subFn func(*gin.Context) string,
	objFn func(*gin.Context) string,
	actFn func(*gin.Context) string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := subFn(c)
		obj := objFn(c)
		act := actFn(c)

		allowed, err := a.service.Enforce(sub, obj, act)
		if err != nil {
			// Log authorization error for security audit
			logGinAuthzError(c, sub, obj, act, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authorization error"})
			return
		}

		if !allowed {
			// Log authorization denial for security audit
			logGinAuthzDenied(c, sub, obj, act)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}



// logGinAuthzError logs Gin authorization errors for security audit.
func logGinAuthzError(c *gin.Context, subject, resource, action string, err error) {
	logger.Warnw("authorization error",
		"subject", subject,
		"resource", resource,
		"action", action,
		"error", err.Error(),
		"remote_addr", c.ClientIP(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"user_agent", c.Request.UserAgent(),
	)
}

// logGinAuthzDenied logs Gin authorization denials for security audit.
func logGinAuthzDenied(c *gin.Context, subject, resource, action string) {
	logger.Warnw("authorization denied",
		"subject", subject,
		"resource", resource,
		"action", action,
		"remote_addr", c.ClientIP(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"user_agent", c.Request.UserAgent(),
	)
}


