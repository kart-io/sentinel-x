package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHelper provides utilities for e2e tests
type TestHelper struct {
	t       *testing.T
	tempDir string
	cleanup []func()
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir, err := os.MkdirTemp("", "logger-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	helper := &TestHelper{
		t:       t,
		tempDir: tempDir,
		cleanup: []func(){},
	}

	// Register cleanup for temp directory
	helper.AddCleanup(func() {
		os.RemoveAll(tempDir)
	})

	return helper
}

// GetTempDir returns the temporary directory for this test
func (h *TestHelper) GetTempDir() string {
	return h.tempDir
}

// CreateTempFile creates a temporary file with given content
func (h *TestHelper) CreateTempFile(name, content string) string {
	filePath := filepath.Join(h.tempDir, name)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		h.t.Fatalf("Failed to write file %s: %v", filePath, err)
	}

	return filePath
}

// ReadFile reads the content of a file
func (h *TestHelper) ReadFile(filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		h.t.Fatalf("Failed to read file %s: %v", filepath, err)
	}
	return string(content)
}

// FileExists checks if a file exists
func (h *TestHelper) FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

// GetFileSize returns the size of a file in bytes
func (h *TestHelper) GetFileSize(filepath string) int64 {
	info, err := os.Stat(filepath)
	if err != nil {
		h.t.Fatalf("Failed to get file info for %s: %v", filepath, err)
	}
	return info.Size()
}

// CountLines counts the number of lines in a file
func (h *TestHelper) CountLines(filepath string) int {
	file, err := os.Open(filepath)
	if err != nil {
		h.t.Fatalf("Failed to open file %s: %v", filepath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		h.t.Fatalf("Error scanning file %s: %v", filepath, err)
	}

	return count
}

// ParseJSONLogs parses JSON logs from a file and returns parsed entries
func (h *TestHelper) ParseJSONLogs(filepath string) []map[string]interface{} {
	file, err := os.Open(filepath)
	if err != nil {
		h.t.Fatalf("Failed to open log file %s: %v", filepath, err)
	}
	defer file.Close()

	var logs []map[string]interface{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			h.t.Logf("Failed to parse JSON log line: %s, error: %v", line, err)
			continue
		}

		logs = append(logs, logEntry)
	}

	if err := scanner.Err(); err != nil {
		h.t.Fatalf("Error scanning log file %s: %v", filepath, err)
	}

	return logs
}

// WaitForFile waits for a file to exist or times out
func (h *TestHelper) WaitForFile(filepath string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.FileExists(filepath) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// WaitForFileSize waits for a file to reach a minimum size or times out
func (h *TestHelper) WaitForFileSize(filepath string, minSize int64, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.FileExists(filepath) && h.GetFileSize(filepath) >= minSize {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// ListFiles lists all files in a directory (non-recursive)
func (h *TestHelper) ListFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		h.t.Fatalf("Failed to read directory %s: %v", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
}

// FindRotatedFiles finds rotated log files (with patterns like .log.1, .log.2, etc.)
func (h *TestHelper) FindRotatedFiles(baseName string) []string {
	dir := filepath.Dir(baseName)
	baseFile := filepath.Base(baseName)

	entries, err := os.ReadDir(dir)
	if err != nil {
		h.t.Fatalf("Failed to read directory %s: %v", dir, err)
	}

	var rotatedFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// Look for patterns like app.log.1, app.log.2, app.log.2023-01-01, etc.
			if strings.HasPrefix(name, baseFile+".") && name != baseFile {
				rotatedFiles = append(rotatedFiles, filepath.Join(dir, name))
			}
		}
	}
	return rotatedFiles
}

// CopyFile copies a file from src to dst
func (h *TestHelper) CopyFile(src, dst string) {
	source, err := os.Open(src)
	if err != nil {
		h.t.Fatalf("Failed to open source file %s: %v", src, err)
	}
	defer source.Close()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		h.t.Fatalf("Failed to create directory %s: %v", dstDir, err)
	}

	destination, err := os.Create(dst)
	if err != nil {
		h.t.Fatalf("Failed to create destination file %s: %v", dst, err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		h.t.Fatalf("Failed to copy file from %s to %s: %v", src, dst, err)
	}
}

// AddCleanup adds a cleanup function to be called during test cleanup
func (h *TestHelper) AddCleanup(fn func()) {
	h.cleanup = append(h.cleanup, fn)
}

// Cleanup runs all registered cleanup functions
func (h *TestHelper) Cleanup() {
	for i := len(h.cleanup) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					h.t.Logf("Cleanup function panicked: %v", r)
				}
			}()
			h.cleanup[i]()
		}()
	}
}

// SafeFlush safely flushes a logger, handling sync errors for stdout/stderr
func (h *TestHelper) SafeFlush(logger interface{}) error {
	type Flusher interface {
		Flush() error
	}

	if flusher, ok := logger.(Flusher); ok {
		err := flusher.Flush()
		if err != nil {
			// Ignore sync errors for stdout/stderr and file closing errors in test environment
			if strings.Contains(err.Error(), "sync /dev/stdout") ||
				strings.Contains(err.Error(), "sync /dev/stderr") ||
				strings.Contains(err.Error(), "bad file descriptor") ||
				strings.Contains(err.Error(), "file already closed") {
				h.t.Logf("Ignoring expected flush error in test environment: %v", err)
				return nil
			}
		}
		return err
	}
	return nil
}

// AssertLogContains checks that a log file contains expected entries
func (h *TestHelper) AssertLogContains(filepath string, expectedEntries []LogAssertion) {
	logs := h.ParseJSONLogs(filepath)

	for _, assertion := range expectedEntries {
		found := false
		for _, log := range logs {
			if assertion.Matches(log) {
				found = true
				break
			}
		}
		if !found {
			h.t.Errorf("Expected log entry not found: %+v", assertion)
		}
	}
}

// LogAssertion represents an assertion for log content
type LogAssertion struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

// Matches checks if a log entry matches this assertion
func (a LogAssertion) Matches(logEntry map[string]interface{}) bool {
	// Check level
	if a.Level != "" {
		if level, ok := logEntry["level"].(string); !ok || !strings.EqualFold(level, a.Level) {
			return false
		}
	}

	// Check message (partial match)
	if a.Message != "" {
		if msg, ok := logEntry["message"].(string); !ok || !strings.Contains(msg, a.Message) {
			if msg, ok := logEntry["msg"].(string); !ok || !strings.Contains(msg, a.Message) {
				return false
			}
		}
	}

	// Check fields
	for key, expectedValue := range a.Fields {
		if actualValue, ok := logEntry[key]; !ok {
			return false
		} else {
			// Simple equality check - could be enhanced for complex types
			if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
				return false
			}
		}
	}

	return true
}
