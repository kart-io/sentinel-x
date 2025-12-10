package mysql

import (
	"encoding/json"
	"strings"
	"testing"

	mysqlOpts "github.com/kart-io/sentinel-x/pkg/options/mysql"
)

func TestBuildDSN_Basic(t *testing.T) {
	opts := &mysqlOpts.Options{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "secret",
		Database: "testdb",
	}

	dsn := BuildDSN(opts)

	if !strings.Contains(dsn, "root:secret@tcp(localhost:3306)/testdb") {
		t.Errorf("unexpected DSN: %s", dsn)
	}
}

func TestBuildDSN_PasswordEscaping(t *testing.T) {
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
		{
			name:     "password with colon",
			password: "pass:word",
			expected: "pass%3Aword",
		},
		{
			name:     "complex password",
			password: "p@ss:w/rd!#$",
			expected: "p%40ss%3Aw%2Frd%21%23%24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &mysqlOpts.Options{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: tt.password,
				Database: "testdb",
			}

			dsn := BuildDSN(opts)

			expectedPart := "root:" + tt.expected + "@tcp"
			if !strings.Contains(dsn, expectedPart) {
				t.Errorf("DSN password not properly escaped: got %s, expected to contain %s", dsn, expectedPart)
			}
		})
	}
}

func TestOptionsJSONMarshal_PasswordRedacted(t *testing.T) {
	opts := &mysqlOpts.Options{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "supersecret",
		Database: "testdb",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Password should be redacted
	if strings.Contains(jsonStr, "supersecret") {
		t.Error("password should be redacted in JSON output")
	}

	if !strings.Contains(jsonStr, "[REDACTED]") {
		t.Error("JSON output should contain [REDACTED] placeholder")
	}
}

func TestOptionsJSONMarshal_EmptyPassword(t *testing.T) {
	opts := &mysqlOpts.Options{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "",
		Database: "testdb",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Empty password should show empty string, not [REDACTED]
	if strings.Contains(jsonStr, "[REDACTED]") {
		t.Error("empty password should not be replaced with [REDACTED]")
	}

	if !strings.Contains(jsonStr, `"password":""`) {
		t.Error("empty password should remain empty in JSON output")
	}
}

func TestOptionsString_PasswordRedacted(t *testing.T) {
	opts := &mysqlOpts.Options{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "supersecret",
		Database: "testdb",
	}

	str := opts.String()

	if strings.Contains(str, "supersecret") {
		t.Error("password should be redacted in String() output")
	}

	if !strings.Contains(str, "[REDACTED]") {
		t.Error("String() output should contain [REDACTED] placeholder")
	}
}
