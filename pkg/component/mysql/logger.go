package mysql

import (
	"context"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"github.com/kart-io/logger"
)

// GormLogger adapts the unified logger to GORM's logger interface.
type GormLogger struct {
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

// NewGormLogger creates a new GormLogger.
func NewGormLogger(logLevel gormlogger.LogLevel, slowThreshold time.Duration, ignoreRecordNotFoundError bool) *GormLogger {
	return &GormLogger{
		LogLevel:                  logLevel,
		SlowThreshold:             slowThreshold,
		IgnoreRecordNotFoundError: ignoreRecordNotFoundError,
	}
}

// LogMode sets the log level.
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info messages.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		logger.Global().WithCtx(ctx).Infof(msg, data...)
	}
}

// Warn logs warning messages.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		logger.Global().WithCtx(ctx).Warnf(msg, data...)
	}
}

// Error logs error messages.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		logger.Global().WithCtx(ctx).Errorf(msg, data...)
	}
}

// Trace logs SQL queries.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!l.IgnoreRecordNotFoundError || !isRecordNotFoundError(err)):
		sql, rows := fc()
		logger.Global().WithCtx(ctx).Errorw("Database query failed",
			"error", err,
			"sql", sql,
			"rows", rows,
			"duration_ms", float64(elapsed.Nanoseconds())/1e6,
		)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		logger.Global().WithCtx(ctx).Warnw("Slow database query detected",
			"sql", sql,
			"rows", rows,
			"duration_ms", float64(elapsed.Nanoseconds())/1e6,
			"threshold_ms", float64(l.SlowThreshold.Nanoseconds())/1e6,
		)
	case l.LogLevel >= gormlogger.Info:
		sql, rows := fc()
		logger.Global().WithCtx(ctx).Infow("Database query executed",
			"sql", sql,
			"rows", rows,
			"duration_ms", float64(elapsed.Nanoseconds())/1e6,
		)
	}
}

func isRecordNotFoundError(err error) bool {
	return err != nil && err == gormlogger.ErrRecordNotFound
}

// Ensure interface compliance
var _ gormlogger.Interface = (*GormLogger)(nil)
