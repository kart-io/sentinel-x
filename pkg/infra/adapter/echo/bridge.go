// Package echo provides Echo HTTP framework bridge implementation.
// This package isolates all Echo-specific code, making framework upgrades easier.
package echo

import (
	"net/http"

	httpserver "github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
	httpopts "github.com/kart-io/sentinel-x/pkg/options/server/http"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func init() {
	// Register the Echo bridge
	httpserver.RegisterBridge(httpopts.AdapterEcho, NewBridge)
}

// Bridge is the Echo HTTP framework bridge implementation.
// This implements FrameworkBridge interface and isolates all Echo-specific code.
type Bridge struct {
	engine *echo.Echo
}

// NewBridge creates a new Echo bridge.
func NewBridge() httpserver.FrameworkBridge {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())

	return &Bridge{
		engine: e,
	}
}

// Name returns the framework name.
func (b *Bridge) Name() string {
	return "echo"
}

// Handler returns the http.Handler.
func (b *Bridge) Handler() http.Handler {
	return b.engine
}

// AddRoute adds a route handler.
func (b *Bridge) AddRoute(method, path string, handler httpserver.BridgeHandler) {
	b.engine.Add(method, path, b.wrapHandler(handler))
}

// AddRouteGroup creates a route group.
func (b *Bridge) AddRouteGroup(prefix string) httpserver.RouteGroup {
	return &routeGroup{
		group:  b.engine.Group(prefix),
		bridge: b,
	}
}

// AddMiddleware adds global middleware.
func (b *Bridge) AddMiddleware(mw httpserver.BridgeMiddleware) {
	b.engine.Use(b.wrapMiddleware(mw))
}

// SetNotFoundHandler sets the 404 handler.
func (b *Bridge) SetNotFoundHandler(handler httpserver.BridgeHandler) {
	b.engine.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusNotFound {
			ctx := b.createContext(c)
			handler(ctx)
			return
		}
		b.engine.DefaultHTTPErrorHandler(err, c)
	}
}

// SetErrorHandler sets the error handler.
func (b *Bridge) SetErrorHandler(handler httpserver.BridgeErrorHandler) {
	b.engine.HTTPErrorHandler = func(err error, c echo.Context) {
		ctx := b.createContext(c)
		handler(err, ctx)
	}
}

// Static serves static files from the given root directory.
func (b *Bridge) Static(prefix, root string) {
	b.engine.Static(prefix, root)
}

// Mount mounts an http.Handler to the given prefix.
func (b *Bridge) Mount(prefix string, handler http.Handler) {
	b.engine.Any(prefix+"*", echo.WrapHandler(http.StripPrefix(prefix, handler)))
}

// wrapHandler converts a BridgeHandler to echo.HandlerFunc.
func (b *Bridge) wrapHandler(handler httpserver.BridgeHandler) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := b.createContext(c)
		handler(ctx)
		return nil
	}
}

// wrapMiddleware converts a BridgeMiddleware to echo.MiddlewareFunc.
func (b *Bridge) wrapMiddleware(mw httpserver.BridgeMiddleware) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := b.createContext(c)
			var nextCalled bool
			mw(func(ctx *httpserver.RequestContext) {
				nextCalled = true
				_ = next(c)
			})(ctx)
			if !nextCalled {
				return nil
			}
			return nil
		}
	}
}

// createContext creates a RequestContext from echo.Context.
func (b *Bridge) createContext(c echo.Context) *httpserver.RequestContext {
	ctx := httpserver.NewRequestContext(c.Request(), c.Response().Writer)

	// Copy URL params
	for _, name := range c.ParamNames() {
		ctx.SetParam(name, c.Param(name))
	}

	// Store raw echo context for advanced use
	ctx.SetRawContext(c)

	return ctx
}

// routeGroup implements httpserver.RouteGroup for Echo.
type routeGroup struct {
	group  *echo.Group
	bridge *Bridge
}

func (g *routeGroup) AddRoute(method, path string, handler httpserver.BridgeHandler) {
	g.group.Add(method, path, g.bridge.wrapHandler(handler))
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

func (g *routeGroup) Static(prefix, root string) {
	g.group.Static(prefix, root)
}

func (g *routeGroup) Mount(prefix string, handler http.Handler) {
	g.group.Any(prefix+"*", echo.WrapHandler(http.StripPrefix(prefix, handler)))
}

// Ensure Bridge implements FrameworkBridge.
var _ httpserver.FrameworkBridge = (*Bridge)(nil)

// Ensure routeGroup implements RouteGroup.
var _ httpserver.RouteGroup = (*routeGroup)(nil)
