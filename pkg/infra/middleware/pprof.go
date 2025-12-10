package middleware

import (
	"net/http"
	"net/http/pprof"
	"runtime"
	"strings"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// RegisterPprofRoutes registers pprof routes.
func RegisterPprofRoutes(router transport.Router, opts PprofOptions) {
	// Set profiling rates if specified
	if opts.BlockProfileRate > 0 {
		runtime.SetBlockProfileRate(opts.BlockProfileRate)
	}
	if opts.MutexProfileFraction > 0 {
		runtime.SetMutexProfileFraction(opts.MutexProfileFraction)
	}

	prefix := opts.Prefix
	if prefix == "" {
		prefix = "/debug/pprof"
	}

	// Ensure prefix doesn't end with /
	prefix = strings.TrimSuffix(prefix, "/")

	// Index handler - shows all available profiles
	router.Handle(http.MethodGet, prefix+"/", wrapPprofHandler(pprof.Index))
	router.Handle(http.MethodGet, prefix, wrapPprofHandler(pprof.Index))

	// Cmdline handler
	if opts.EnableCmdline {
		router.Handle(http.MethodGet, prefix+"/cmdline", wrapPprofHandler(pprof.Cmdline))
	}

	// Profile handler (CPU profiling)
	if opts.EnableProfile {
		router.Handle(http.MethodGet, prefix+"/profile", wrapPprofHandler(pprof.Profile))
	}

	// Symbol handler
	if opts.EnableSymbol {
		router.Handle(http.MethodGet, prefix+"/symbol", wrapPprofHandler(pprof.Symbol))
		router.Handle(http.MethodPost, prefix+"/symbol", wrapPprofHandler(pprof.Symbol))
	}

	// Trace handler
	if opts.EnableTrace {
		router.Handle(http.MethodGet, prefix+"/trace", wrapPprofHandler(pprof.Trace))
	}

	// Standard pprof handlers for specific profiles
	profiles := []string{
		"allocs",
		"block",
		"goroutine",
		"heap",
		"mutex",
		"threadcreate",
	}

	for _, profile := range profiles {
		router.Handle(http.MethodGet, prefix+"/"+profile, wrapPprofHandler(pprof.Index))
	}
}

// wrapPprofHandler wraps a http.HandlerFunc to transport.HandlerFunc.
func wrapPprofHandler(h http.HandlerFunc) transport.HandlerFunc {
	return func(c transport.Context) {
		h(c.ResponseWriter(), c.HTTPRequest())
	}
}

// PprofHandler returns a handler that serves pprof endpoints.
// This is useful when you need more control over the pprof routes.
type PprofHandler struct {
	opts PprofOptions
}

// NewPprofHandler creates a new pprof handler.
func NewPprofHandler(opts PprofOptions) *PprofHandler {
	return &PprofHandler{opts: opts}
}

// ServeHTTP implements http.Handler.
func (h *PprofHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prefix := h.opts.Prefix
	if prefix == "" {
		prefix = "/debug/pprof"
	}
	prefix = strings.TrimSuffix(prefix, "/")

	path := strings.TrimPrefix(r.URL.Path, prefix)
	path = strings.TrimPrefix(path, "/")

	switch path {
	case "", "index":
		pprof.Index(w, r)
	case "cmdline":
		if h.opts.EnableCmdline {
			pprof.Cmdline(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "profile":
		if h.opts.EnableProfile {
			pprof.Profile(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "symbol":
		if h.opts.EnableSymbol {
			pprof.Symbol(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "trace":
		if h.opts.EnableTrace {
			pprof.Trace(w, r)
		} else {
			http.NotFound(w, r)
		}
	default:
		// Handle other profiles like heap, goroutine, etc.
		pprof.Index(w, r)
	}
}

// EnableBlockProfiling enables block profiling with the given rate.
// A rate of 1 records every blocking event, while 0 disables profiling.
func EnableBlockProfiling(rate int) {
	runtime.SetBlockProfileRate(rate)
}

// EnableMutexProfiling enables mutex profiling with the given fraction.
// A fraction of 1 records every contention event, while 0 disables profiling.
func EnableMutexProfiling(fraction int) {
	runtime.SetMutexProfileFraction(fraction)
}

// DisableBlockProfiling disables block profiling.
func DisableBlockProfiling() {
	runtime.SetBlockProfileRate(0)
}

// DisableMutexProfiling disables mutex profiling.
func DisableMutexProfiling() {
	runtime.SetMutexProfileFraction(0)
}
