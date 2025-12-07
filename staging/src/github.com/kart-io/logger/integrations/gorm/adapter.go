package gorm

import (
	"context"
	"time"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/integrations"
)

// LogLevel represents GORM log levels without importing GORM
type LogLevel int

const (
	Silent LogLevel = iota + 1
	Error
	Warn
	Info
)

// Interface mimics GORM's logger interface to avoid direct dependency
type Interface interface {
	LogMode(LogLevel) Interface
	Info(context.Context, string, ...interface{})
	Warn(context.Context, string, ...interface{})
	Error(context.Context, string, ...interface{})
	Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error)
}

// GormAdapter implements GORM's logger interface using our unified logger
type GormAdapter struct {
	*integrations.BaseAdapter
	logLevel                  LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

// NewGormAdapter creates a new GORM adapter
func NewGormAdapter(coreLogger core.Logger) *GormAdapter {
	baseAdapter := integrations.NewBaseAdapter(coreLogger, "GORM", "v1.x")
	return &GormAdapter{
		BaseAdapter:               baseAdapter,
		logLevel:                  Info,
		slowThreshold:             200 * time.Millisecond,
		ignoreRecordNotFoundError: true,
	}
}

// NewGormAdapterWithConfig creates a new GORM adapter with configuration
func NewGormAdapterWithConfig(coreLogger core.Logger, config Config) *GormAdapter {
	baseAdapter := integrations.NewBaseAdapter(coreLogger, "GORM", "v1.x")
	return &GormAdapter{
		BaseAdapter:               baseAdapter,
		logLevel:                  config.LogLevel,
		slowThreshold:             config.SlowThreshold,
		ignoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

// Config holds configuration for the GORM adapter
type Config struct {
	LogLevel                  LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

// DefaultConfig returns default configuration for GORM adapter
func DefaultConfig() Config {
	return Config{
		LogLevel:                  Info,
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode implements GORM's logger interface
func (g *GormAdapter) LogMode(level LogLevel) Interface {
	newAdapter := *g
	newAdapter.logLevel = level
	return &newAdapter
}

// Info implements GORM's logger interface
func (g *GormAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.logLevel >= Info {
		g.GetLogger().WithCtx(ctx).Infof(msg, data...)
	}
}

// Warn implements GORM's logger interface
func (g *GormAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.logLevel >= Warn {
		g.GetLogger().WithCtx(ctx).Warnf(msg, data...)
	}
}

// Error implements GORM's logger interface
func (g *GormAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.logLevel >= Error {
		g.GetLogger().WithCtx(ctx).Errorf(msg, data...)
	}
}

// Trace implements GORM's logger interface for SQL logging
func (g *GormAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.logLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Create base fields
	fields := []interface{}{
		"sql", sql,
		"duration_ms", float64(elapsed.Nanoseconds()) / 1e6,
		"rows", rows,
	}

	switch {
	case err != nil && g.logLevel >= Error && (!g.ignoreRecordNotFoundError || !isRecordNotFoundError(err)):
		g.LogError(err, sql, fields...)

	case elapsed > g.slowThreshold && g.slowThreshold != 0 && g.logLevel >= Warn:
		g.LogSlowQuery(sql, elapsed.Nanoseconds(), g.slowThreshold.Nanoseconds(), fields...)

	case g.logLevel >= Info:
		g.LogQuery(sql, elapsed.Nanoseconds(), fields...)
	}
}

// LogQuery logs a database query (implements DatabaseAdapter interface)
func (g *GormAdapter) LogQuery(query string, duration int64, params ...interface{}) {
	fields := []interface{}{
		"component", "gorm",
		"operation", "query",
		"query", query,
		"duration_ns", duration,
	}
	fields = append(fields, params...)

	g.GetLogger().Infow("Database query executed", fields...)
}

// LogError logs a database error (implements DatabaseAdapter interface)
func (g *GormAdapter) LogError(err error, query string, params ...interface{}) {
	fields := []interface{}{
		"component", "gorm",
		"operation", "query",
		"query", query,
		"error", err.Error(),
	}
	fields = append(fields, params...)

	g.GetLogger().Errorw("Database query failed", fields...)
}

// LogSlowQuery logs queries that exceed the slow query threshold (implements DatabaseAdapter interface)
func (g *GormAdapter) LogSlowQuery(query string, duration int64, threshold int64, params ...interface{}) {
	fields := []interface{}{
		"component", "gorm",
		"operation", "slow_query",
		"query", query,
		"duration_ns", duration,
		"threshold_ns", threshold,
		"slowdown_factor", float64(duration) / float64(threshold),
	}
	fields = append(fields, params...)

	g.GetLogger().Warnw("Slow database query detected", fields...)
}

// SetSlowThreshold sets the slow query threshold
func (g *GormAdapter) SetSlowThreshold(threshold time.Duration) {
	g.slowThreshold = threshold
}

// GetSlowThreshold returns the current slow query threshold
func (g *GormAdapter) GetSlowThreshold() time.Duration {
	return g.slowThreshold
}

// SetIgnoreRecordNotFoundError sets whether to ignore RecordNotFound errors
func (g *GormAdapter) SetIgnoreRecordNotFoundError(ignore bool) {
	g.ignoreRecordNotFoundError = ignore
}

// GetIgnoreRecordNotFoundError returns whether RecordNotFound errors are ignored
func (g *GormAdapter) GetIgnoreRecordNotFoundError() bool {
	return g.ignoreRecordNotFoundError
}

// Helper function to check if an error is a RecordNotFound error
func isRecordNotFoundError(err error) bool {
	// This would typically check for GORM's specific RecordNotFound error
	// For now, we'll do a simple string check
	return err != nil && err.Error() == "record not found"
}

// Verify that GormAdapter implements both GORM's logger interface and our DatabaseAdapter interface
var _ Interface = (*GormAdapter)(nil)
var _ integrations.DatabaseAdapter = (*GormAdapter)(nil)
