package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/auth"
	"github.com/labstack/echo/v4"
)

// Authorizer provides middleware for authorization
type Authorizer struct {
	service auth.PermissionService
}

// NewAuthorizer creates a new Authorizer
func NewAuthorizer(service auth.PermissionService) *Authorizer {
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
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authorization error"})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}

// EchoMiddleware returns an Echo middleware for authorization
func (a *Authorizer) EchoMiddleware(
	subFn func(echo.Context) string,
	objFn func(echo.Context) string,
	actFn func(echo.Context) string,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sub := subFn(c)
			obj := objFn(c)
			act := actFn(c)

			allowed, err := a.service.Enforce(sub, obj, act)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "authorization error")
			}

			if !allowed {
				return echo.NewHTTPError(http.StatusForbidden, "forbidden")
			}

			return next(c)
		}
	}
}
