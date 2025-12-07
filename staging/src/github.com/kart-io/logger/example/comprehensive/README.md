# Comprehensive Logger Usage Examples

This directory contains comprehensive usage examples for the unified logger library, demonstrating all available methods and features.

## What's Included

### 1. Basic Logging Methods

- `Debug()`, `Info()`, `Warn()`, `Error()` methods
- Simple message logging with automatic caller detection
- Different log levels demonstration

### 2. Printf-style Logging Methods

- `Debugf()`, `Infof()`, `Warnf()`, `Errorf()` methods
- Template-based formatting with parameters
- Type-safe string formatting

### 3. Structured Logging Methods

- `Debugw()`, `Infow()`, `Warnw()`, `Errorw()` methods
- Key-value pair structured logging
- Rich context and metadata support

### 4. Logger Enhancement Methods

- `With()` - Create child loggers with persistent fields
- `WithCtx()` - Add context and fields to logger
- `WithCallerSkip()` - Adjust caller stack frame reporting

### 5. Global Logger Usage

- Package-level convenience functions
- Default logger configuration
- Quick logging without setup

### 6. Configuration Examples

- Slog engine with console format
- Zap engine with production settings
- Dynamic level configuration
- Development vs production settings

### 7. Error Handling and Stacktraces

- Automatic stacktrace generation for Error/Fatal levels
- Both Slog and Zap engine examples
- Rich error context logging

### 8. Context and Tracing

- Distributed tracing context propagation
- Request lifecycle logging
- Field standardization examples
- Tracing metadata integration

## Key Features Demonstrated

### Engine Support

- ✅ **Slog Engine**: Go's standard library structured logging
- ✅ **Zap Engine**: High-performance structured logging

### Output Formats

- ✅ **JSON**: Machine-readable structured format
- ✅ **Console**: Human-readable development format

### Field Standardization

- ✅ Automatic field name mapping (e.g., `ts` → `timestamp`, `msg` → `message`)
- ✅ Consistent field names across engines
- ✅ Tracing field support (`trace.id` → `trace_id`, `span.id` → `span_id`)

### Advanced Features

- ✅ **Caller Detection**: Shows exact code location of log calls
- ✅ **Engine Identification**: `engine` field shows which engine produced the log
- ✅ **Stacktraces**: Automatic stack traces for Error/Fatal levels
- ✅ **Context Integration**: Rich context and metadata support

## Running the Examples

```bash
cd /path/to/logger/example/comprehensive
go run main.go
```

## Sample Output

The examples will produce JSON-formatted log entries like:

```json
{
  "time": "2025-08-29T15:30:00.123456789+08:00",
  "level": "INFO",
  "msg": "User logged in",
  "engine": "slog",
  "caller": "comprehensive/main.go:45",
  "user_id": 12345,
  "action": "login",
  "session_id": "sess_abc123"
}
```

For Error level logs with stacktraces:

```json
{
  "time": "2025-08-29T15:30:00.123456789+08:00",
  "level": "ERROR",
  "msg": "Database connection failed",
  "engine": "slog",
  "caller": "comprehensive/main.go:78",
  "error": "connection timeout",
  "stacktrace": "main.simulateError\\n\\tcomprehensive/main.go:78\\nmain.main\\n\\tcomprehensive/main.go:45"
}
```

## Configuration Options

The examples demonstrate various configuration options:

- **Engine**: `"slog"` or `"zap"`
- **Level**: `"DEBUG"`, `"INFO"`, `"WARN"`, `"ERROR"`, `"FATAL"`
- **Format**: `"json"` or `"console"`
- **OutputPaths**: `["stdout"]`, `["stderr"]`, or file paths
- **Development**: `true` for development, `false` for production
- **DisableCaller**: Disable caller information
- **DisableStacktrace**: Disable automatic stacktraces

## Best Practices Demonstrated

1. **Use structured logging** (`*w` methods) for rich context
2. **Create child loggers** with persistent fields for related operations
3. **Include tracing information** for distributed systems
4. **Use appropriate log levels** for different scenarios
5. **Configure differently** for development vs production
6. **Handle errors properly** with contextual information
