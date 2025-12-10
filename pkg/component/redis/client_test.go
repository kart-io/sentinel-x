package redis

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOptionsJSONMarshal_PasswordRedacted(t *testing.T) {
	opts := &Options{
		Host:     "localhost",
		Port:     6379,
		Password: "supersecret",
		Database: 0,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	if strings.Contains(jsonStr, "supersecret") {
		t.Error("password should be redacted in JSON output")
	}

	if !strings.Contains(jsonStr, "[REDACTED]") {
		t.Error("JSON output should contain [REDACTED] placeholder")
	}
}

func TestOptionsJSONMarshal_EmptyPassword(t *testing.T) {
	opts := &Options{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	if strings.Contains(jsonStr, "[REDACTED]") {
		t.Error("empty password should not be replaced with [REDACTED]")
	}

	if !strings.Contains(jsonStr, `"password":""`) {
		t.Error("empty password should remain empty in JSON output")
	}
}

func TestOptionsString_PasswordRedacted(t *testing.T) {
	opts := &Options{
		Host:     "localhost",
		Port:     6379,
		Password: "supersecret",
		Database: 0,
	}

	str := opts.String()

	if strings.Contains(str, "supersecret") {
		t.Error("password should be redacted in String() output")
	}

	if !strings.Contains(str, "[REDACTED]") {
		t.Error("String() output should contain [REDACTED] placeholder")
	}
}

func TestNewOptions_Defaults(t *testing.T) {
	opts := NewOptions()

	if opts.Host != "127.0.0.1" {
		t.Errorf("expected default host 127.0.0.1, got %s", opts.Host)
	}

	if opts.Port != 6379 {
		t.Errorf("expected default port 6379, got %d", opts.Port)
	}

	if opts.MaxRetries != 3 {
		t.Errorf("expected default max retries 3, got %d", opts.MaxRetries)
	}

	if opts.PoolSize != 50 {
		t.Errorf("expected default pool size 50, got %d", opts.PoolSize)
	}

	if opts.MinIdleConns != 10 {
		t.Errorf("expected default min idle conns 10, got %d", opts.MinIdleConns)
	}

	if opts.DialTimeout != 5*time.Second {
		t.Errorf("expected default dial timeout 5s, got %v", opts.DialTimeout)
	}
}
