package gin

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/integrations"
)

// Writer represents gin.ResponseWriter interface to avoid direct dependency
type ResponseWriter interface {
	http.ResponseWriter
	Status() int
	Size() int
	WriteString(string) (int, error)
	Written() bool
	WriteHeaderNow()
}

// Context represents gin.Context interface to avoid direct dependency
type Context interface {
	Request() *http.Request
	Writer() ResponseWriter
	Param(key string) string
	Query(key string) string
	PostForm(key string) string
	Get(key string) (interface{}, bool)
	Set(key, value interface{})
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetInt64(key string) int64
	GetFloat64(key string) float64
	GetTime(key string) time.Time
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	GetStringMapStringSlice(key string) map[string][]string
	ClientIP() string
	ContentType() string
	IsWebsocket() bool
	Header(key, value string)
	GetHeader(key string) string
	GetRawData() ([]byte, error)
	SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	Cookie(name string) (string, error)
	Render(code int, r interface{})
	HTML(code int, name string, obj interface{})
	IndentedJSON(code int, obj interface{})
	SecureJSON(code int, obj interface{})
	JSONP(code int, callback string, obj interface{})
	JSON(code int, obj interface{})
	AsciiJSON(code int, obj interface{})
	PureJSON(code int, obj interface{})
	XML(code int, obj interface{})
	YAML(code int, obj interface{})
	TOML(code int, obj interface{})
	ProtoBuf(code int, obj interface{})
	String(code int, format string, values ...interface{})
	Redirect(code int, location string)
	Data(code int, contentType string, data []byte)
	DataFromReader(code int, contentLength int64, contentType string, reader io.Reader, extraHeaders map[string]string)
	File(filepath string)
	FileFromFS(filepath string, fs http.FileSystem)
	FileAttachment(filepath, filename string)
	SSEvent(name string, message interface{})
	Stream(step func(w io.Writer) bool)
	Abort()
	AbortWithError(code int, err error) *HTTPError
	AbortWithStatus(code int)
	AbortWithStatusJSON(code int, jsonObj interface{})
	Next()
}

// HandlerFunc represents gin.HandlerFunc type
type HandlerFunc func(Context)

// IRoutes represents gin.IRoutes interface
type IRoutes interface {
	Use(...HandlerFunc) IRoutes
	Handle(string, string, ...HandlerFunc) IRoutes
	Any(string, ...HandlerFunc) IRoutes
	GET(string, ...HandlerFunc) IRoutes
	POST(string, ...HandlerFunc) IRoutes
	DELETE(string, ...HandlerFunc) IRoutes
	PATCH(string, ...HandlerFunc) IRoutes
	PUT(string, ...HandlerFunc) IRoutes
	OPTIONS(string, ...HandlerFunc) IRoutes
	HEAD(string, ...HandlerFunc) IRoutes
}

// Engine represents gin.Engine interface
type Engine interface {
	IRoutes
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// HTTPError represents gin's HTTP error
type HTTPError struct {
	Err  error
	Type uint32
	Meta interface{}
}

// Error returns error message
func (msg HTTPError) Error() string {
	return msg.Err.Error()
}

// GinAdapter implements logging for Gin framework
type GinAdapter struct {
	*integrations.BaseAdapter
	config Config
}

// Config holds configuration for the Gin adapter
type Config struct {
	// SkipPaths defines paths to skip in logging
	SkipPaths []string
	// SkipPathRegexp defines regexp patterns to skip in logging
	SkipPathRegexp []string
	// TimeFormat defines the time format for logging
	TimeFormat string
	// UTC defines whether to use UTC time
	UTC bool
	// LogLevel defines the minimum log level
	LogLevel core.Level
	// LogRequestBody defines whether to log request body
	LogRequestBody bool
	// LogResponseBody defines whether to log response body
	LogResponseBody bool
	// MaxBodySize defines maximum body size to log (in bytes)
	MaxBodySize int64
	// SkipClientError defines whether to skip 4xx status codes
	SkipClientError bool
	// LogLatency defines whether to log request latency
	LogLatency bool
}

// DefaultConfig returns default configuration for Gin adapter
func DefaultConfig() Config {
	return Config{
		SkipPaths:       []string{},
		SkipPathRegexp:  []string{},
		TimeFormat:      time.RFC3339,
		UTC:             true,
		LogLevel:        core.InfoLevel,
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodySize:     1024 * 1024, // 1MB
		SkipClientError: false,
		LogLatency:      true,
	}
}

// NewGinAdapter creates a new Gin adapter
func NewGinAdapter(coreLogger core.Logger) *GinAdapter {
	baseAdapter := integrations.NewBaseAdapter(coreLogger, "Gin", "v1.x")
	return &GinAdapter{
		BaseAdapter: baseAdapter,
		config:      DefaultConfig(),
	}
}

// NewGinAdapterWithConfig creates a new Gin adapter with configuration
func NewGinAdapterWithConfig(coreLogger core.Logger, config Config) *GinAdapter {
	baseAdapter := integrations.NewBaseAdapter(coreLogger, "Gin", "v1.x")
	return &GinAdapter{
		BaseAdapter: baseAdapter,
		config:      config,
	}
}

// Logger returns a Gin logging middleware
func (g *GinAdapter) Logger() HandlerFunc {
	return g.LoggerWithFormatter(g.defaultLogFormatter)
}

// LoggerWithFormatter returns a Gin logging middleware with custom formatter
func (g *GinAdapter) LoggerWithFormatter(f LogFormatter) HandlerFunc {
	return func(c Context) {
		start := time.Now()
		path := c.Request().URL.Path
		raw := c.Request().URL.RawQuery

		// Skip logging for configured paths
		if g.shouldSkipPath(path) {
			c.Next()
			return
		}

		// Process request
		c.Next()

		// Log after processing
		param := LogFormatterParams{
			Request:      c.Request(),
			TimeStamp:    time.Now(),
			Latency:      time.Since(start),
			ClientIP:     c.ClientIP(),
			Method:       c.Request().Method,
			StatusCode:   c.Writer().Status(),
			ErrorMessage: g.getErrorMessage(c),
			BodySize:     c.Writer().Size(),
			Keys:         g.getContextKeys(c),
		}

		if raw != "" {
			path = path + "?" + raw
		}
		param.Path = path

		f(param, g)
	}
}

// LogFormatterParams contains the values for formatting log entries
type LogFormatterParams struct {
	Request      *http.Request
	TimeStamp    time.Time
	StatusCode   int
	Latency      time.Duration
	ClientIP     string
	Method       string
	Path         string
	ErrorMessage string
	BodySize     int
	Keys         map[string]interface{}
}

// StatusCodeColor returns color for status code (for colored output)
func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return "\033[97;42m" // white + green bg
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return "\033[90;47m" // black + white bg
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return "\033[90;43m" // black + yellow bg
	default:
		return "\033[97;41m" // white + red bg
	}
}

// MethodColor returns color for HTTP method (for colored output)
func (p *LogFormatterParams) MethodColor() string {
	method := p.Method
	switch method {
	case http.MethodGet:
		return "\033[94m" // blue
	case http.MethodPost:
		return "\033[92m" // green
	case http.MethodPut:
		return "\033[93m" // yellow
	case http.MethodDelete:
		return "\033[91m" // red
	case http.MethodPatch:
		return "\033[95m" // magenta
	case http.MethodHead:
		return "\033[96m" // cyan
	case http.MethodOptions:
		return "\033[90m" // dark gray
	default:
		return "\033[0m" // reset
	}
}

// ResetColor returns ANSI reset color code
func (p *LogFormatterParams) ResetColor() string {
	return "\033[0m"
}

// IsOutputColor returns whether output should be colored
func (p *LogFormatterParams) IsOutputColor() bool {
	return true // This could be configurable
}

// LogFormatter defines the signature for log formatting functions
type LogFormatter func(params LogFormatterParams, adapter *GinAdapter)

// defaultLogFormatter is the default log formatter
func (g *GinAdapter) defaultLogFormatter(param LogFormatterParams, adapter *GinAdapter) {
	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	timestamp := param.TimeStamp.Format(g.config.TimeFormat)
	if g.config.UTC {
		timestamp = param.TimeStamp.UTC().Format(g.config.TimeFormat)
	}

	fields := []interface{}{
		"component", "gin",
		"operation", "http_request",
		"timestamp", timestamp,
		"status", param.StatusCode,
		"latency", param.Latency.String(),
		"client_ip", param.ClientIP,
		"method", param.Method,
		"path", param.Path,
		"body_size", param.BodySize,
	}

	// Add error message if present
	if param.ErrorMessage != "" {
		fields = append(fields, "error", param.ErrorMessage)
	}

	// Add context keys if present
	if len(param.Keys) > 0 {
		for k, v := range param.Keys {
			fields = append(fields, k, v)
		}
	}

	// Determine log level based on status code
	level := g.getLogLevelForStatusCode(param.StatusCode)

	// Skip client errors if configured
	if g.config.SkipClientError && param.StatusCode >= 400 && param.StatusCode < 500 {
		return
	}

	// Format message with colors if enabled
	msg := fmt.Sprintf("[GIN] %s |%s %3d %s| %13v | %15s |%s %-7s %s %s",
		timestamp,
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
	)

	// Log with appropriate level
	switch level {
	case core.DebugLevel:
		g.GetLogger().Debugw(msg, fields...)
	case core.InfoLevel:
		g.GetLogger().Infow(msg, fields...)
	case core.WarnLevel:
		g.GetLogger().Warnw(msg, fields...)
	case core.ErrorLevel:
		g.GetLogger().Errorw(msg, fields...)
	case core.FatalLevel:
		g.GetLogger().Fatalw(msg, fields...)
	default:
		g.GetLogger().Infow(msg, fields...)
	}
}

// LogRequest logs an HTTP request (implements HTTPAdapter interface)
func (g *GinAdapter) LogRequest(method, path string, statusCode int, duration int64, userID string) {
	fields := []interface{}{
		"component", "gin",
		"operation", "http_request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", float64(duration) / 1e6,
	}

	if userID != "" {
		fields = append(fields, "user_id", userID)
	}

	level := g.getLogLevelForStatusCode(statusCode)
	msg := fmt.Sprintf("HTTP %s %s", method, path)

	switch level {
	case core.InfoLevel:
		g.GetLogger().Infow(msg, fields...)
	case core.WarnLevel:
		g.GetLogger().Warnw(msg, fields...)
	case core.ErrorLevel:
		g.GetLogger().Errorw(msg, fields...)
	default:
		g.GetLogger().Infow(msg, fields...)
	}
}

// LogMiddleware logs middleware execution (implements HTTPAdapter interface)
func (g *GinAdapter) LogMiddleware(middlewareName string, duration int64) {
	fields := []interface{}{
		"component", "gin",
		"operation", "middleware",
		"middleware_name", middlewareName,
		"duration_ms", float64(duration) / 1e6,
	}

	g.GetLogger().Debugw("Middleware executed", fields...)
}

// LogError logs HTTP-related errors (implements HTTPAdapter interface)
func (g *GinAdapter) LogError(err error, method, path string, statusCode int) {
	fields := []interface{}{
		"component", "gin",
		"operation", "http_error",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"error", err.Error(),
	}

	g.GetLogger().Errorw("HTTP request failed", fields...)
}

// Recovery returns a Gin recovery middleware
func (g *GinAdapter) Recovery() HandlerFunc {
	return g.RecoveryWithWriter(nil)
}

// RecoveryWithWriter returns a Gin recovery middleware with custom writer
func (g *GinAdapter) RecoveryWithWriter(out io.Writer) HandlerFunc {
	return func(c Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic details
				fields := []interface{}{
					"component", "gin",
					"operation", "panic_recovery",
					"method", c.Request().Method,
					"path", c.Request().URL.Path,
					"client_ip", c.ClientIP(),
					"user_agent", c.Request().UserAgent(),
					"panic", fmt.Sprintf("%v", err),
				}

				// Add stack trace if available
				if errMsg, ok := err.(error); ok {
					fields = append(fields, "error", errMsg.Error())
				}

				g.GetLogger().Errorw("Panic recovered", fields...)

				// Abort with status 500
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// Helper methods

// shouldSkipPath checks if the path should be skipped from logging
func (g *GinAdapter) shouldSkipPath(path string) bool {
	// Check exact path matches
	for _, skipPath := range g.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}

	// Check regex patterns (implementation would use regexp package)
	// For now, just return false
	return false
}

// getErrorMessage extracts error message from context
func (g *GinAdapter) getErrorMessage(c Context) string {
	// This would typically extract errors from Gin context
	// For now, return empty string
	return ""
}

// getContextKeys extracts relevant keys from context
func (g *GinAdapter) getContextKeys(c Context) map[string]interface{} {
	keys := make(map[string]interface{})

	// Add commonly used context keys
	if userID, exists := c.Get("user_id"); exists {
		keys["user_id"] = userID
	}

	if requestID, exists := c.Get("request_id"); exists {
		keys["request_id"] = requestID
	}

	if traceID, exists := c.Get("trace_id"); exists {
		keys["trace_id"] = traceID
	}

	return keys
}

// getLogLevelForStatusCode determines the appropriate log level based on HTTP status code
func (g *GinAdapter) getLogLevelForStatusCode(statusCode int) core.Level {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return core.InfoLevel
	case statusCode >= 400 && statusCode < 500:
		return core.WarnLevel
	case statusCode >= 500:
		return core.ErrorLevel
	default:
		return core.InfoLevel
	}
}

// SetConfig updates the adapter configuration
func (g *GinAdapter) SetConfig(config Config) {
	g.config = config
}

// GetConfig returns the current adapter configuration
func (g *GinAdapter) GetConfig() Config {
	return g.config
}

// Verify that GinAdapter implements the HTTPAdapter interface
var _ integrations.HTTPAdapter = (*GinAdapter)(nil)
