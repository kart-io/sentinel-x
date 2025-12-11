package jwt

import (
	"context"
	"strings"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

func TestJWT_Verify_Reproduction(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	// Create a valid token first
	token, err := j.Sign(ctx, "user-123")
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}
	validToken := token.GetAccessToken()

	tests := []struct {
		name        string
		token       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Token with space in middle",
			token:       validToken[:20] + " " + validToken[20:],
			wantErr:     true,
			errContains: "illegal base64 data",
		},
		{
			name:        "Token with standard base64 chars (+)",
			token:       strings.ReplaceAll(validToken, "-", "+"), // Replace URL-safe - with +
			wantErr:     true,
			errContains: "illegal base64 data",
		},
		{
			name:        "Token with standard base64 chars (/)",
			token:       strings.ReplaceAll(validToken, "_", "/"), // Replace URL-safe _ with /
			wantErr:     true,
			errContains: "illegal base64 data",
		},
		{
			name:        "Token with padding (=)",
			token:       validToken + "=",
			wantErr:     true,
			errContains: "illegal base64 data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := j.Verify(ctx, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Logf("Error: %v", err)
					// We expect the error to be wrapped in ErrInvalidToken
					// But the Cause should contain the message
					errno := errors.FromError(err)
					if !strings.Contains(errno.Error(), tt.errContains) && !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("Expected error containing '%s', got '%v'", tt.errContains, err)
					}
				}
			}
		})
	}
}
