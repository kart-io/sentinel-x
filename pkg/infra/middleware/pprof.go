package middleware

import (
	"net/http"
	"net/http/pprof"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// RegisterPprofRoutesWithOptions 注册 Pprof 路由端点。
// 这是推荐的 API，使用纯配置选项。
//
// 参数：
//   - engine: Gin 引擎
//   - opts: Pprof 配置选项
//
// 示例：
//
//	opts := mwopts.NewPprofOptions()
//	RegisterPprofRoutesWithOptions(engine, *opts)
func RegisterPprofRoutesWithOptions(engine *gin.Engine, opts mwopts.PprofOptions) {
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
	engine.GET(prefix+"/", gin.WrapF(pprof.Index))
	engine.GET(prefix, gin.WrapF(pprof.Index))

	// Cmdline handler
	if opts.EnableCmdline {
		engine.GET(prefix+"/cmdline", gin.WrapF(pprof.Cmdline))
	}

	// Profile handler (CPU profiling)
	if opts.EnableProfile {
		engine.GET(prefix+"/profile", gin.WrapF(pprof.Profile))
	}

	// Symbol handler
	if opts.EnableSymbol {
		engine.GET(prefix+"/symbol", gin.WrapF(pprof.Symbol))
		engine.POST(prefix+"/symbol", gin.WrapF(pprof.Symbol))
	}

	// Trace handler
	if opts.EnableTrace {
		engine.GET(prefix+"/trace", gin.WrapF(pprof.Trace))
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
		engine.GET(prefix+"/"+profile, gin.WrapF(pprof.Index))
	}
}

// PprofHandler returns a handler that serves pprof endpoints.
// This is useful when you need more control over the pprof routes.
type PprofHandler struct {
	opts mwopts.PprofOptions
}

// NewPprofHandler creates a new pprof handler.
func NewPprofHandler(opts mwopts.PprofOptions) *PprofHandler {
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
