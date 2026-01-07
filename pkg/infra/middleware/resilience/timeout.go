package resilience

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/pool"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// Timeout returns a middleware that limits request processing time.
func Timeout(timeout time.Duration) transport.MiddlewareFunc {
	return TimeoutWithOptions(mwopts.TimeoutOptions{
		Timeout: timeout,
	})
}

// TimeoutWithOptions returns a Timeout middleware with TimeoutOptions.
// 这是推荐的构造函数，直接使用 pkg/options/middleware.TimeoutOptions。
func TimeoutWithOptions(opts mwopts.TimeoutOptions) transport.MiddlewareFunc {
	// Set defaults
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	skipPaths := make(map[string]bool)
	for _, path := range opts.SkipPaths {
		skipPaths[path] = true
	}

	return func(next transport.HandlerFunc) transport.HandlerFunc {
		return func(c transport.Context) {
			req := c.HTTPRequest()

			// Skip timeout for certain paths
			if skipPaths[req.URL.Path] {
				next(c)
				return
			}

			// Create timeout context to propagate cancellation to downstream handlers
			ctx, cancel := context.WithTimeout(c.Request(), opts.Timeout)
			defer cancel()

			// Update request context so downstream handlers can detect timeout
			c.SetRequest(ctx)

			// Create buffered done channel to prevent goroutine leak
			done := make(chan struct{}, 1)

			// Variable to track if timeout occurred
			var timedOut bool

			// 使用 ants 池提交任务，而非直接创建 goroutine
			// 这样可以限制并发数量，避免高并发下 goroutine 爆炸
			task := func() {
				defer func() {
					// Recover from panic to prevent process crash
					if r := recover(); r != nil {
						// Log panic information for debugging
						logPanic(r, req.URL.Path)
					}
					// Signal completion - buffered channel guarantees non-blocking send
					// This ensures the goroutine can exit even if the main goroutine
					// has already handled the timeout
					done <- struct{}{}
				}()

				// Handler will receive the timeout context and should respect it
				// If context is cancelled, handlers should stop processing
				next(c)
			}

			// 提交到超时专用池，如果提交失败则降级为同步执行
			if err := pool.SubmitToType(pool.TimeoutPool, task); err != nil {
				// 池不可用时降级为直接执行（同步模式）
				logger.Warnw("timeout middleware pool unavailable, fallback to sync execution",
					"error", err.Error(),
					"path", req.URL.Path,
				)
				task()
				return
			}

			select {
			case <-done:
				// Request completed normally
				// Check if it completed due to context cancellation
				if ctx.Err() == context.DeadlineExceeded {
					timedOut = true
				}
			case <-ctx.Done():
				// Timeout occurred before handler completed
				timedOut = true
			}

			// Only write timeout response if we haven't written a response yet
			// The handler might have already written a response before context cancellation
			if timedOut && ctx.Err() == context.DeadlineExceeded {
				response.Fail(c, errors.ErrRequestTimeout)
			}
		}
	}
}

// logPanic logs panic information with stack trace for debugging.
func logPanic(r interface{}, path string) {
	stack := debug.Stack()
	logger.Errorw("panic recovered in timeout middleware",
		"panic", fmt.Sprintf("%v", r),
		"path", path,
		"stack", string(stack),
	)
}
