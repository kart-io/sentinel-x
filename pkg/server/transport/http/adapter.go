package http

import (
	"net/http"

	httpopts "github.com/kart-io/sentinel-x/pkg/options/http"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
)

// Adapter defines the interface for HTTP framework adapters.
// This interface provides backward compatibility while using the new bridge system internally.
type Adapter interface {
	// Name returns the adapter name.
	Name() string

	// Router returns the abstract router for registering routes.
	Router() transport.Router

	// Handler returns the http.Handler for use with net/http server.
	Handler() http.Handler

	// SetNotFoundHandler sets the handler for 404 responses.
	SetNotFoundHandler(handler transport.HandlerFunc)

	// SetErrorHandler sets the global error handler.
	SetErrorHandler(handler func(err error, c transport.Context))

	// Bridge returns the underlying FrameworkBridge (optional, for advanced use).
	Bridge() FrameworkBridge
}

// bridgeAdapter wraps a FrameworkBridge to implement the Adapter interface.
// This provides backward compatibility with the existing code.
type bridgeAdapter struct {
	bridge FrameworkBridge
	router *bridgeRouter
}

// newBridgeAdapter creates a new bridgeAdapter from a FrameworkBridge.
func newBridgeAdapter(bridge FrameworkBridge) Adapter {
	return &bridgeAdapter{
		bridge: bridge,
		router: &bridgeRouter{group: bridge.AddRouteGroup("")},
	}
}

func (a *bridgeAdapter) Name() string {
	return a.bridge.Name()
}

func (a *bridgeAdapter) Router() transport.Router {
	return a.router
}

func (a *bridgeAdapter) Handler() http.Handler {
	return a.bridge.Handler()
}

func (a *bridgeAdapter) SetNotFoundHandler(handler transport.HandlerFunc) {
	a.bridge.SetNotFoundHandler(func(ctx *RequestContext) {
		handler(ctx)
	})
}

func (a *bridgeAdapter) SetErrorHandler(handler func(err error, c transport.Context)) {
	a.bridge.SetErrorHandler(func(err error, ctx *RequestContext) {
		handler(err, ctx)
	})
}

func (a *bridgeAdapter) Bridge() FrameworkBridge {
	return a.bridge
}

// bridgeRouter wraps RouteGroup to implement transport.Router.
type bridgeRouter struct {
	group RouteGroup
}

func (r *bridgeRouter) Handle(method, path string, handler transport.HandlerFunc) {
	r.group.AddRoute(method, path, func(ctx *RequestContext) {
		handler(ctx)
	})
}

func (r *bridgeRouter) Group(prefix string) transport.Router {
	return &bridgeRouter{group: r.group.AddRouteGroup(prefix)}
}

func (r *bridgeRouter) Use(middleware ...transport.MiddlewareFunc) {
	for _, m := range middleware {
		mw := m // capture
		r.group.AddMiddleware(func(next BridgeHandler) BridgeHandler {
			return func(ctx *RequestContext) {
				mw(func(c transport.Context) {
					// Only call next if response not yet written
					if !ctx.Written() {
						next(ctx)
					}
				})(ctx)
			}
		})
	}
}

// bridges stores registered bridge factories.
var bridges = make(map[httpopts.AdapterType]BridgeFactory)

// RegisterBridge registers a bridge factory for the given adapter type.
func RegisterBridge(adapterType httpopts.AdapterType, factory BridgeFactory) {
	bridges[adapterType] = factory
}

// GetAdapter returns an adapter for the given type.
// This creates a bridge and wraps it in the backward-compatible Adapter interface.
func GetAdapter(adapterType httpopts.AdapterType) Adapter {
	if factory, ok := bridges[adapterType]; ok {
		bridge := factory()
		return newBridgeAdapter(bridge)
	}
	return nil
}

// GetBridge returns a FrameworkBridge directly for the given type.
func GetBridge(adapterType httpopts.AdapterType) FrameworkBridge {
	if factory, ok := bridges[adapterType]; ok {
		return factory()
	}
	return nil
}

// Legacy support: RegisterAdapter for backward compatibility
// Deprecated: Use RegisterBridge instead.
type AdapterFactory func() Adapter

var legacyAdapters = make(map[httpopts.AdapterType]AdapterFactory)

// RegisterAdapter registers an adapter factory (legacy, for backward compatibility).
// Deprecated: Use RegisterBridge instead.
func RegisterAdapter(adapterType httpopts.AdapterType, factory AdapterFactory) {
	legacyAdapters[adapterType] = factory
}

// Ensure types implement interfaces.
var (
	_ Adapter           = (*bridgeAdapter)(nil)
	_ transport.Router  = (*bridgeRouter)(nil)
	_ transport.Context = (*RequestContext)(nil)
)
