package postgres

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildDSN_Basic(t *testing.T) {
	opts := &Options{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "secret",
		Database: "testdb",
		SSLMode:  "disable",
	}

	dsn := BuildDSN(opts)

	expectedParts := []string{
		"host=localhost",
		"port=5432",
		"user=postgres",
		"dbname=testdb",
		"sslmode=disable",
	}

	for _, part := range expectedParts {
		if !strings.Contains(dsn, part) {
			t.Errorf("DSN missing expected part: %s, got: %s", part, dsn)
		}
	}
}

func TestBuildDSN_PasswordEscaping(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		wantQuoted bool
	}{
		{
			name:       "simple password",
			password:   "secret",
			wantQuoted: false,
		},
		{
			name:       "password with space",
			password:   "pass word",
			wantQuoted: true,
		},
		{
			name:       "password with single quote",
			password:   "pass'word",
			wantQuoted: true,
		},
		{
			name:       "password with backslash",
			password:   "pass\\word",
			wantQuoted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: tt.password,
				Database: "testdb",
				SSLMode:  "disable",
			}

			dsn := BuildDSN(opts)

			// If password needs quoting, it should be wrapped in single quotes
			if tt.wantQuoted {
				if !strings.Contains(dsn, "password='") {
					t.Errorf("password should be quoted: %s", dsn)
				}
			}

			// DSN should not contain unescaped password that could break parsing
			if tt.password == "pass word" && strings.Contains(dsn, "password=pass word") {
				t.Error("password with space should be quoted")
			}
		})
	}
}

func TestBuildURI_PasswordEscaping(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{
			name:     "simple password",
			password: "secret",
			expected: "secret",
		},
		{
			name:     "password with at sign",
			password: "pass@word",
			expected: "pass%40word",
		},
		{
			name:     "password with slash",
			password: "pass/word",
			expected: "pass%2Fword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: tt.password,
				Database: "testdb",
				SSLMode:  "disable",
			}

			uri := BuildURI(opts)

			expectedPart := "postgres:" + tt.expected + "@"
			if !strings.Contains(uri, expectedPart) {
				t.Errorf("URI password not properly escaped: got %s, expected to contain %s", uri, expectedPart)
			}
		})
	}
}

func TestEscapePostgresValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"", "''"},
		{"with space", "'with space'"},
		{"with'quote", "'with''quote'"},
		{"with\\backslash", "'with\\\\backslash'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapePostgresValue(tt.input)
			if result != tt.expected {
				t.Errorf("escapePostgresValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOptionsJSONMarshal_PasswordRedacted(t *testing.T) {
	opts := &Options{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "supersecret",
		Database: "testdb",
		SSLMode:  "disable",
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

func TestOptionsString_PasswordRedacted(t *testing.T) {
	opts := &Options{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "supersecret",
		Database: "testdb",
		SSLMode:  "disable",
	}

	str := opts.String()

	if strings.Contains(str, "supersecret") {
		t.Error("password should be redacted in String() output")
	}

	if !strings.Contains(str, "[REDACTED]") {
		t.Error("String() output should contain [REDACTED] placeholder")
	}
}
