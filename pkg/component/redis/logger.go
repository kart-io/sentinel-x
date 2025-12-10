package redis

import (
	"context"

	"github.com/kart-io/logger"
	goredis "github.com/redis/go-redis/v9"
)

// loggingAdapter adapts the unified logger to go-redis logger interface.
type loggingAdapter struct{}

// Printf logs messages using the unified logger.
func (l *loggingAdapter) Printf(ctx context.Context, format string, v ...interface{}) {
	logger.Global().WithCtx(ctx).Infof(format, v...)
}

// init configures go-redis to use the unified logger.
func init() {
	goredis.SetLogger(&loggingAdapter{})
}
