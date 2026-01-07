package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Normal Bearer token",
			header:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
		},
		{
			name:     "Token with spaces",
			header:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9 . payload . signature",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
		},
		{
			name:     "Token with standard base64 (+ and /)",
			header:   "Bearer header.pay/load.sig+nature",
			expected: "header.pay_load.sig-nature",
		},
		{
			name:     "Token with padding (=)",
			header:   "Bearer header.payload.signature==",
			expected: "header.payload.signature",
		},
		{
			name:     "Mixed issues",
			header:   "Bearer header . pay/load . sig+nature ==",
			expected: "header.pay_load.sig-nature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.header)
			c.Request = req

			lookup := tokenLookup{source: "header", name: "Authorization"}

			got := extractToken(c, lookup, "Bearer")
			if got != tt.expected {
				t.Errorf("extractToken() = %v, want %v", got, tt.expected)
			}
		})
	}
}
