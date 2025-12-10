// Package main demonstrates how to use the sonic-powered JSON utilities
// in sentinel-x handlers. No code changes required - it works automatically!
package main

import (
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/json"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// Example 1: Standard API response (Automatic sonic usage)
// No code changes needed - response.OK() uses sonic automatically
func HandleGetUser(ctx transport.Context) {
	user := map[string]interface{}{
		"id":       123,
		"username": "johndoe",
		"email":    "john@example.com",
		"role":     "admin",
	}

	// This now uses sonic automatically! 2-4x faster than before
	response.OK(ctx, user)
}

// Example 2: Request parsing (Automatic sonic usage)
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func HandleCreateUser(ctx transport.Context) {
	var req CreateUserRequest

	// Bind now uses sonic decoder - 4x faster deserialization!
	if err := ctx.ShouldBindAndValidate(&req); err != nil {
		response.FailWithBindOrValidation(ctx, err)
		return
	}

	// Process user creation...
	user := map[string]interface{}{
		"id":       124,
		"username": req.Username,
		"email":    req.Email,
	}

	response.OK(ctx, user)
}

// Example 3: Paginated response (Even bigger improvement for large payloads)
func HandleListUsers(ctx transport.Context) {
	// Fetch users from database...
	users := []map[string]interface{}{
		{"id": 1, "username": "user1", "email": "user1@example.com"},
		{"id": 2, "username": "user2", "email": "user2@example.com"},
		// ... 20 items
	}

	// Page responses are 2.6x faster with sonic!
	response.PageOK(ctx, users, 100, 1, 20)
}

// Example 4: Direct JSON usage (if you need it)
func HandleCustomJSON(ctx transport.Context) {
	data := map[string]interface{}{
		"timestamp": 1703001234567,
		"metrics": map[string]int{
			"requests":  1000,
			"errors":    5,
			"successes": 995,
		},
	}

	// Direct JSON marshal using sonic
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		response.Fail(ctx, errors.ErrInternal)
		return
	}

	ctx.SetHeader("Content-Type", "application/json")
	ctx.ResponseWriter().Write(jsonBytes)
}

// Example 5: Check if sonic is active (for logging/monitoring)
func CheckJSONImplementation() string {
	if json.IsUsingSonic() {
		return "Using high-performance sonic JSON (2-4x faster)"
	}
	return "Using standard library JSON (architecture fallback)"
}

// Example 6: Using fastest mode for internal services
// WARNING: Only use for trusted data!
func InitInternalService() {
	// For maximum performance in internal service-to-service communication
	json.ConfigFastestMode()
}

// Example 7: Complex nested structure
type UserDetailResponse struct {
	ID       int                    `json:"id"`
	Username string                 `json:"username"`
	Profile  map[string]interface{} `json:"profile"`
	Settings map[string]bool        `json:"settings"`
	Tags     []string               `json:"tags"`
}

func HandleGetUserDetail(ctx transport.Context) {
	detail := UserDetailResponse{
		ID:       123,
		Username: "johndoe",
		Profile: map[string]interface{}{
			"first_name": "John",
			"last_name":  "Doe",
			"age":        30,
			"department": "Engineering",
		},
		Settings: map[string]bool{
			"notifications": true,
			"dark_mode":     false,
		},
		Tags: []string{"admin", "developer", "reviewer"},
	}

	// Sonic handles complex nested structures efficiently
	response.OK(ctx, detail)
}

// Example 8: Error handling still works the same
func HandleWithError(ctx transport.Context) {
	// All error responses also benefit from sonic
	response.Fail(ctx, errors.ErrUnauthorized)
}

// Example 9: Stream processing (for very large responses)
func HandleStreamJSON(ctx transport.Context) {
	// For streaming scenarios, use encoder directly
	encoder := json.NewEncoder(ctx.ResponseWriter())

	for i := 0; i < 100; i++ {
		item := map[string]interface{}{
			"id":   i,
			"data": "item data",
		}
		encoder.Encode(item)
	}
}

func main() {
	// Check JSON implementation
	fmt.Println(CheckJSONImplementation())

	// Example: Direct JSON usage
	data := map[string]interface{}{
		"name": "Sonic Integration Example",
		"features": []string{
			"2-4x faster serialization",
			"46% less memory",
			"83% fewer allocations",
			"Automatic fallback",
		},
		"supported_architectures": []string{"amd64", "arm64"},
	}

	jsonBytes, _ := json.Marshal(data)
	fmt.Printf("\nExample JSON output:\n%s\n", string(jsonBytes))

	fmt.Println("\nPerformance Tips:")
	fmt.Println("1. No code changes needed - response helpers use sonic automatically")
	fmt.Println("2. For high-throughput internal APIs, consider ConfigFastestMode()")
	fmt.Println("3. Sonic is most effective for:")
	fmt.Println("   - Medium to large payloads (>100 bytes)")
	fmt.Println("   - High request volume (>100 req/s)")
	fmt.Println("   - Complex nested structures")
	fmt.Println("4. Automatic fallback ensures compatibility on all architectures")
	fmt.Println("5. Monitor with json.IsUsingSonic() for verification")
}
