package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/internal/user-center/handler"
	v1 "github.com/kart-io/sentinel-x/pkg/api/user-center/v1"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/stretchr/testify/assert"
)

// TestAuthHandler_Login_Validation verifies that validation works correctly
// using the global validator and protoc generated rules.
func TestAuthHandler_Login_Validation(t *testing.T) {
	// Initialize handler with nil service - we only want to test validation phase
	// used in ShouldBindAndValidate before service logic is called.
	h := handler.NewAuthHandler(nil)

	tests := []struct {
		name       string
		req        *v1.LoginRequest
		wantStatus int
		wantCode   int // internal business error code
		errMsg     string
	}{
		{
			name: "Valid Request",
			req: &v1.LoginRequest{
				Username: "validuser",
				Password: "validpassword123", // min_len: 6 defined in proto
			},
			// Expecting panic because validation passes and svc is nil
			// We catch this panic to confirm validation passed
			wantStatus: -1,
		},
		{
			name: "Invalid Username - Too Short",
			req: &v1.LoginRequest{
				Username: "ab", // min_len: 3
				Password: "validpassword123",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   errors.ErrBadRequest.Code,
			errMsg:     "Username", // Error message should contain field name
		},
		{
			name: "Invalid Password - Too Short",
			req: &v1.LoginRequest{
				Username: "validuser",
				Password: "123", // min_len: 6
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   errors.ErrBadRequest.Code,
			errMsg:     "Password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Request
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Run Handler
			defer func() {
				if r := recover(); r != nil {
					// Logic: If status is -1, we EXPECT a panic (meaning validation passed and it hit the nil svc)
					if tt.wantStatus == -1 {
						return // Test Passed
					}
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			h.Login(c)

			// Check Status Code (only if no panic)
			if tt.wantStatus != -1 {
				assert.Equal(t, tt.wantStatus, w.Code)

				// Optional: Check Response Body structure for error
				var resp map[string]interface{}
				_ = json.NewDecoder(w.Body).Decode(&resp)

				// Assuming standard error response format
				if code, ok := resp["code"]; ok {
					assert.Equal(t, float64(tt.wantCode), code)
				}
				if msg, ok := resp["message"]; ok {
					assert.Contains(t, msg, tt.errMsg)
				}
			} else {
				t.Error("Expected panic for valid request (nil service) but function returned normally")
			}
		})
	}
}
