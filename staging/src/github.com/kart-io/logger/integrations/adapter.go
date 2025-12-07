package integrations

import (
	"github.com/kart-io/logger/core"
)

// Adapter defines the interface that framework adapters should implement
type Adapter interface {
	// GetLogger returns the logger instance used by this adapter
	GetLogger() core.Logger

	// SetLogger allows setting a new logger instance
	SetLogger(logger core.Logger)

	// Name returns the name of the framework this adapter serves
	Name() string

	// Version returns the version of the framework this adapter supports
	Version() string
}

// BaseAdapter provides common functionality for all framework adapters
type BaseAdapter struct {
	logger  core.Logger
	name    string
	version string
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(logger core.Logger, name, version string) *BaseAdapter {
	return &BaseAdapter{
		logger:  logger,
		name:    name,
		version: version,
	}
}

// GetLogger returns the logger instance
func (b *BaseAdapter) GetLogger() core.Logger {
	return b.logger
}

// SetLogger sets a new logger instance
func (b *BaseAdapter) SetLogger(logger core.Logger) {
	b.logger = logger
}

// Name returns the framework name
func (b *BaseAdapter) Name() string {
	return b.name
}

// Version returns the framework version
func (b *BaseAdapter) Version() string {
	return b.version
}

// DatabaseAdapter defines additional methods for database framework adapters
type DatabaseAdapter interface {
	Adapter

	// LogQuery logs a database query with execution time and parameters
	LogQuery(query string, duration int64, params ...interface{})

	// LogError logs a database error
	LogError(err error, query string, params ...interface{})

	// LogSlowQuery logs queries that exceed the slow query threshold
	LogSlowQuery(query string, duration int64, threshold int64, params ...interface{})
}

// HTTPAdapter defines additional methods for HTTP framework adapters
type HTTPAdapter interface {
	Adapter

	// LogRequest logs an HTTP request
	LogRequest(method, path string, statusCode int, duration int64, userID string)

	// LogMiddleware logs middleware execution
	LogMiddleware(middlewareName string, duration int64)

	// LogError logs HTTP-related errors
	LogError(err error, method, path string, statusCode int)
}
