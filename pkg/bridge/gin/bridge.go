// Package gin provides Gin HTTP framework bridge implementation.
// This package isolates all Gin-specific code, making framework upgrades easier.
package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/http"
	httpserver "github.com/kart-io/sentinel-x/pkg/server/transport/http"
)

func init() {
	// Register the Gin bridge
	httpserver.RegisterBridge(httpopts.AdapterGin, NewBridge)
}

// Bridge is the Gin HTTP framework bridge implementation.
// This implements FrameworkBridge interface and isolates all Gin-specific code.
type Bridge struct {
	engine *gin.Engine
}

// NewBridge creates a new Gin bridge.
func NewBridge() httpserver.FrameworkBridge {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	return &Bridge{
		engine: engine,
	}
}

// Name returns the framework name.
func (b *Bridge) Name() string {
	return "gin"
}

// Handler returns the http.Handler.
func (b *Bridge) Handler() http.Handler {
	return b.engine
}

// AddRoute adds a route handler.
func (b *Bridge) AddRoute(method, path string, handler httpserver.BridgeHandler) {
	b.engine.Handle(method, path, b.wrapHandler(handler))
}

// AddRouteGroup creates a route group.
func (b *Bridge) AddRouteGroup(prefix string) httpserver.RouteGroup {
	return &routeGroup{
		group:  b.engine.Group(prefix),
		bridge: b,
	}
}

// AddMiddleware adds global middleware.
func (b *Bridge) AddMiddleware(middleware httpserver.BridgeMiddleware) {
	b.engine.Use(b.wrapMiddleware(middleware))
}

// SetNotFoundHandler sets the 404 handler.
func (b *Bridge) SetNotFoundHandler(handler httpserver.BridgeHandler) {
	b.engine.NoRoute(b.wrapHandler(handler))
}

// SetErrorHandler sets the error handler.
func (b *Bridge) SetErrorHandler(handler httpserver.BridgeErrorHandler) {
	b.engine.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			ctx := b.createContext(c)
			handler(c.Errors.Last(), ctx)
		}
	})
}

// wrapHandler converts a BridgeHandler to gin.HandlerFunc.
func (b *Bridge) wrapHandler(handler httpserver.BridgeHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := b.createContext(c)
		handler(ctx)
	}
}

// wrapMiddleware converts a BridgeMiddleware to gin.HandlerFunc.
func (b *Bridge) wrapMiddleware(middleware httpserver.BridgeMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := b.createContext(c)
		middleware(func(ctx *httpserver.RequestContext) {
			c.Next()
		})(ctx)
	}
}

// createContext creates a RequestContext from gin.Context.
func (b *Bridge) createContext(c *gin.Context) *httpserver.RequestContext {
	ctx := httpserver.NewRequestContext(c.Request, c.Writer)

	// Copy URL params
	for _, param := range c.Params {
		ctx.SetParam(param.Key, param.Value)
	}

	// Store raw gin context for advanced use
	ctx.SetRawContext(c)

	return ctx
}

// routeGroup implements httpserver.RouteGroup for Gin.
type routeGroup struct {
	group  *gin.RouterGroup
	bridge *Bridge
}

func (g *routeGroup) AddRoute(method, path string, handler httpserver.BridgeHandler) {
	g.group.Handle(method, path, g.bridge.wrapHandler(handler))
}

func (g *routeGroup) AddRouteGroup(prefix string) httpserver.RouteGroup {
	return &routeGroup{
		group:  g.group.Group(prefix),
		bridge: g.bridge,
	}
}

func (g *routeGroup) AddMiddleware(middleware httpserver.BridgeMiddleware) {
	g.group.Use(g.bridge.wrapMiddleware(middleware))
}

// Ensure Bridge implements FrameworkBridge.
var _ httpserver.FrameworkBridge = (*Bridge)(nil)

// Ensure routeGroup implements RouteGroup.
var _ httpserver.RouteGroup = (*routeGroup)(nil)
