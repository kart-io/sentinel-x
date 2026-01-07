package response_test

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// Example_automaticPooling demonstrates automatic pooling with Writer
func Example_automaticPooling() {
	// Mock context (in real code, this comes from your HTTP framework)
	var ctx *gin.Context

	// Recommended: Use Writer for automatic pooling
	w := response.NewWriter(ctx)
	w.WithTimestamp()
	w.WithRequestID("req-123")

	// Success response - automatically uses pooling
	w.OK(map[string]interface{}{
		"user_id": 12345,
		"name":    "John Doe",
	})

	// Error response - automatically uses pooling
	w.Fail(errors.ErrNotFound)

	// Paginated response - automatically uses pooling
	users := []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
	}
	w.PageOK(users, 100, 1, 10)
}

// Example_manualPooling demonstrates manual pooling
func Example_manualPooling() {
	// Acquire a Response from the pool
	resp := response.Acquire()
	defer response.Release(resp) // IMPORTANT: Always release

	// Set fields
	resp.Code = 0
	resp.Message = "success"
	resp.Data = map[string]string{
		"status": "completed",
	}
	resp.RequestID = "req-456"
	resp.Timestamp = time.Now().UnixMilli()

	// Use the response (e.g., write to client)
	fmt.Printf("Response: code=%d, message=%s\n", resp.Code, resp.Message)
	// Output: Response: code=0, message=success
}

// Example_helperFunctions demonstrates using helper functions with pooling
func Example_helperFunctions() {
	// Success response
	resp1 := response.Success(map[string]string{"key": "value"})
	defer response.Release(resp1)
	fmt.Printf("Success: %v\n", resp1.IsSuccess())

	// Error response
	resp2 := response.Err(errors.ErrInvalidParam)
	defer response.Release(resp2)
	fmt.Printf("Error code: %d\n", resp2.Code)

	// Custom error
	resp3 := response.ErrorWithCode(400, "custom error")
	defer response.Release(resp3)
	fmt.Printf("Custom error: %s\n", resp3.Message)

	// Output:
	// Success: true
	// Error code: 1001
	// Custom error: custom error
}

// Example_highThroughputHandler demonstrates pooling in a high-traffic handler
func Example_highThroughputHandler() {
	// Simulated HTTP handler for high-traffic endpoint
	handler := func(ctx *gin.Context) {
		// Option 1: Use Writer (recommended)
		response.NewWriter(ctx).OK(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})

		// Option 2: Manual pooling (advanced)
		// resp := response.Success(data)
		// defer response.Release(resp)
		// ctx.JSON(resp.HTTPStatus(), resp)
	}

	// Simulate processing
	_ = handler
}

// Example_errorHandling demonstrates proper error handling with pooling
func Example_errorHandling() {
	// Success case
	func() {
		resp := response.Success("operation completed")
		defer response.Release(resp)

		if !resp.IsSuccess() {
			// Handle error
			return
		}

		// Process success
		fmt.Println("Success:", resp.Message)
	}()

	// Error case
	func() {
		resp := response.Err(errors.ErrInternal)
		defer response.Release(resp)

		if !resp.IsSuccess() {
			fmt.Println("Error:", resp.Message)
			return
		}
	}()

	// Output:
	// Success: success
	// Error: Internal server error
}

// Example_concurrentRequests demonstrates thread-safe concurrent usage
func Example_concurrentRequests() {
	// Simulate concurrent request handling
	processRequest := func(requestID int) {
		resp := response.Acquire()
		defer response.Release(resp)

		// Simulate request processing
		resp.Code = 0
		resp.Message = "success"
		resp.Data = map[string]interface{}{
			"request_id": requestID,
			"processed":  true,
		}

		// In real code, this would write to HTTP response
		_ = resp.HTTPStatus()
	}

	// Process multiple requests concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			processRequest(id)
			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}

	fmt.Println("Processed 10 concurrent requests")
	// Output: Processed 10 concurrent requests
}

// Example_paginationWithPooling demonstrates paginated responses
func Example_paginationWithPooling() {
	// Sample data
	items := []map[string]interface{}{
		{"id": 1, "name": "Item 1"},
		{"id": 2, "name": "Item 2"},
		{"id": 3, "name": "Item 3"},
	}

	// Create paginated response
	resp := response.Page(items, 100, 1, 10)
	defer response.Release(resp)

	// Use the response
	if pageData, ok := resp.Data.(*response.PageData); ok {
		fmt.Printf("Page %d of %d\n", pageData.Page, pageData.TotalPages)
	}

	// Output: Page 1 of 10
}

// Example_customResponseWithPooling demonstrates custom response creation
func Example_customResponseWithPooling() {
	// Acquire from pool
	resp := response.Acquire()
	defer response.Release(resp)

	// Customize response
	resp.Code = 0
	resp.Message = "custom success"
	resp.Data = map[string]interface{}{
		"custom_field": "custom_value",
	}

	// Add metadata
	resp.WithRequestID("custom-req-123")
	resp.WithTimestamp(time.Now().UnixMilli())

	fmt.Printf("Custom response: %s\n", resp.Message)
	// Output: Custom response: custom success
}

// Example_errorWithLanguage demonstrates localized error responses
func Example_errorWithLanguage() {
	// Error response with English (default)
	resp1 := response.Err(errors.ErrNotFound)
	defer response.Release(resp1)
	fmt.Println("EN:", resp1.Message)

	// Error response with specific language
	resp2 := response.ErrWithLang(errors.ErrNotFound, "zh")
	defer response.Release(resp2)
	fmt.Println("ZH:", resp2.Message)

	// Output:
	// EN: Resource not found
	// ZH: 资源不存在
}

// BenchmarkExample demonstrates how to benchmark your handlers
func Example_benchmarking() {
	// In your *_test.go file:
	//
	// func BenchmarkMyHandler(b *testing.B) {
	//     b.ReportAllocs()
	//     b.RunParallel(func(pb *testing.PB) {
	//         for pb.Next() {
	//             resp := response.Success(myData)
	//             // Process response
	//             response.Release(resp)
	//         }
	//     })
	// }
	//
	// Run with: go test -bench=BenchmarkMyHandler -benchmem
}

// Example_commonMistakes demonstrates what NOT to do
func Example_commonMistakes() {
	// ❌ WRONG: Forgetting to release
	wrongExample := func() {
		resp := response.Acquire()
		resp.Code = 0
		// BUG: Forgot to release - memory leak!
		_ = resp
	}

	// ✅ CORRECT: Always release
	correctExample := func() {
		resp := response.Acquire()
		defer response.Release(resp)
		resp.Code = 0
	}

	// ❌ WRONG: Using after release
	wrongExample2 := func() {
		resp := response.Acquire()
		response.Release(resp)
		resp.Message = "test" // BUG: Using after release!
	}

	// ✅ CORRECT: Use before release
	correctExample2 := func() {
		resp := response.Acquire()
		resp.Message = "test"
		response.Release(resp) // Release at the end
	}

	wrongExample()
	correctExample()
	wrongExample2()
	correctExample2()
}
