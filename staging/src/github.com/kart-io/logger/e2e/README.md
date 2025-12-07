# End-to-End (E2E) Tests

This directory contains end-to-end tests for the unified logger library, including tests for log rotation, YAML configuration parsing, and integration scenarios.

## Test Structure

- `rotation_test.go` - Tests log file rotation functionality with real file I/O
- `yaml_config_test.go` - Tests YAML configuration file parsing and application
- `integration_test.go` - Tests integration between rotation, OTLP, and other features
- `helpers.go` - Common test utilities and helpers
- `testdata/` - Test configuration files and fixtures

## Running E2E Tests

```bash
# Run all e2e tests
go test ./e2e -v

# Run specific test suites
go test ./e2e -run TestRotation -v
go test ./e2e -run TestYAMLConfig -v
go test ./e2e -run TestIntegration -v

# Run with cleanup (removes test files)
go test ./e2e -cleanup

# Run with verbose output and coverage
go test ./e2e -v -cover
```

## Test Scenarios

### Log Rotation Tests
- File size-based rotation
- Time-based rotation  
- Backup file management
- Compression functionality
- Mixed output scenarios (console + file)

### YAML Configuration Tests
- Complete configuration parsing
- Default value application
- Validation error handling
- Environment variable overrides
- Multi-source configuration resolution

### Integration Tests
- Rotation + OTLP combined
- Rotation with different engines (Zap/Slog)
- Complex multi-output scenarios
- Error recovery and fallback behavior

## Test Data

The `testdata/` directory contains:
- Sample YAML configuration files
- Expected log output samples
- Test certificates for OTLP testing (if needed)

## Cleanup

E2E tests create temporary files and directories. The test framework automatically cleans up after each test, but you can force cleanup with:

```bash
rm -rf ./e2e/testdata/tmp/
```