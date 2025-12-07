package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	gin_adapter "github.com/kart-io/logger/integrations/gin"
	"github.com/kart-io/logger/option"
)

// User represents a user in our API
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func main() {
	// Initialize logger with configuration
	logConfig := option.LogOption{
		Level:  core.InfoLevel,
		Format: core.JSONFormat,
		Output: option.OutputOption{
			Console: &option.ConsoleOption{
				Enable: true,
				Color:  true,
			},
		},
	}

	coreLogger, err := logger.NewLoggerFromOption(logConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Create Gin adapter with custom configuration
	ginConfig := gin_adapter.Config{
		LogLevel:        core.InfoLevel,
		LogRequestBody:  true,  // Enable request body logging for demo
		LogResponseBody: false, // Disable response body logging for production
		MaxBodySize:     1024,  // 1KB max body size for logging
		SkipClientError: false, // Log client errors for debugging
		LogLatency:      true,  // Enable latency logging
		TimeFormat:      time.RFC3339,
		UTC:             true,
		SkipPaths:       []string{"/ping", "/metrics"}, // Skip health check endpoints
	}

	ginAdapter := gin_adapter.NewGinAdapterWithConfig(coreLogger, ginConfig)

	// Set Gin to release mode for production-like behavior
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()

	// Add logging middleware (choose one approach)

	// Approach 1: Use default middleware set
	// for _, middleware := range ginAdapter.DefaultMiddleware() {
	//     router.Use(middleware)
	// }

	// Approach 2: Use production middleware set with health check skipping
	// for _, middleware := range ginAdapter.ProductionMiddleware("/ping", "/health", "/metrics") {
	//     router.Use(middleware)
	// }

	// Approach 3: Custom middleware setup (recommended for flexibility)
	router.Use(ginAdapter.RequestIDMiddleware("X-Request-ID"))
	router.Use(ginAdapter.UserContextMiddleware("X-User-ID"))
	router.Use(ginAdapter.HealthCheckSkipper("/ping", "/health", "/metrics"))
	router.Use(ginAdapter.RequestBodyLogger(1024))
	router.Use(ginAdapter.Logger())
	router.Use(ginAdapter.MetricsMiddleware())
	router.Use(ginAdapter.Recovery())

	// Define routes

	// Health check endpoint
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "gin-logger-example"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Get all users
		api.GET("/users", func(c *gin.Context) {
			// Add some context for logging
			c.Set("operation", "list_users")

			users := []User{
				{ID: 1, Name: "Alice", Email: "alice@example.com"},
				{ID: 2, Name: "Bob", Email: "bob@example.com"},
				{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
			}

			// Log additional information
			ginAdapter.GetLogger().Infow("Users retrieved successfully",
				"component", "api",
				"operation", "list_users",
				"user_count", len(users),
				"request_id", c.GetString("request_id"),
			)

			c.JSON(http.StatusOK, gin.H{"users": users, "count": len(users)})
		})

		// Get user by ID
		api.GET("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.Set("operation", "get_user")
			c.Set("user_id", id)

			// Simulate user lookup
			if id == "1" {
				user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}
				c.JSON(http.StatusOK, user)
			} else if id == "404" {
				// Demonstrate error logging
				err := fmt.Errorf("user not found")
				ginAdapter.LogError(err, c.Request.Method, c.Request.URL.Path, http.StatusNotFound)

				c.JSON(http.StatusNotFound, ErrorResponse{
					Error:   "user_not_found",
					Message: "User with specified ID was not found",
					Code:    http.StatusNotFound,
				})
			} else {
				user := User{ID: 2, Name: "Unknown", Email: "unknown@example.com"}
				c.JSON(http.StatusOK, user)
			}
		})

		// Create user
		api.POST("/users", func(c *gin.Context) {
			var user User
			c.Set("operation", "create_user")

			if err := c.ShouldBindJSON(&user); err != nil {
				ginAdapter.LogError(err, c.Request.Method, c.Request.URL.Path, http.StatusBadRequest)

				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "invalid_input",
					Message: "Invalid user data provided",
					Code:    http.StatusBadRequest,
				})
				return
			}

			// Simulate user creation
			user.ID = int(time.Now().Unix()) // Simple ID generation

			ginAdapter.GetLogger().Infow("User created successfully",
				"component", "api",
				"operation", "create_user",
				"user_id", user.ID,
				"user_name", user.Name,
				"request_id", c.GetString("request_id"),
			)

			c.JSON(http.StatusCreated, user)
		})

		// Update user
		api.PUT("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.Set("operation", "update_user")
			c.Set("user_id", id)

			var user User
			if err := c.ShouldBindJSON(&user); err != nil {
				ginAdapter.LogError(err, c.Request.Method, c.Request.URL.Path, http.StatusBadRequest)

				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "invalid_input",
					Message: "Invalid user data provided",
					Code:    http.StatusBadRequest,
				})
				return
			}

			ginAdapter.GetLogger().Infow("User updated successfully",
				"component", "api",
				"operation", "update_user",
				"user_id", id,
				"request_id", c.GetString("request_id"),
			)

			c.JSON(http.StatusOK, user)
		})

		// Delete user
		api.DELETE("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.Set("operation", "delete_user")
			c.Set("user_id", id)

			ginAdapter.GetLogger().Warnw("User deleted",
				"component", "api",
				"operation", "delete_user",
				"user_id", id,
				"request_id", c.GetString("request_id"),
			)

			c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully", "user_id": id})
		})

		// Simulate server error
		api.GET("/error", func(c *gin.Context) {
			c.Set("operation", "simulate_error")

			err := fmt.Errorf("simulated internal server error")
			ginAdapter.LogError(err, c.Request.Method, c.Request.URL.Path, http.StatusInternalServerError)

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "An internal server error occurred",
				Code:    http.StatusInternalServerError,
			})
		})

		// Simulate panic for recovery testing
		api.GET("/panic", func(c *gin.Context) {
			c.Set("operation", "simulate_panic")
			panic("This is a simulated panic for testing recovery middleware")
		})
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		coreLogger.Infow("Starting Gin server",
			"component", "server",
			"address", server.Addr,
			"pid", os.Getpid(),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			coreLogger.Errorw("Server failed to start",
				"component", "server",
				"error", err.Error(),
			)
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Print example API calls
	printExampleUsage()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	coreLogger.Infow("Server is shutting down...", "component", "server")

	// Give outstanding requests a deadline of 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		coreLogger.Errorw("Server forced to shutdown",
			"component", "server",
			"error", err.Error(),
		)
	}

	coreLogger.Infow("Server shutdown completed", "component", "server")
}

func printExampleUsage() {
	fmt.Println("\n" + "="*60)
	fmt.Println("🚀 Gin Logger Integration Example")
	fmt.Println("=" * 60)
	fmt.Println("Server is running on: http://localhost:8080")
	fmt.Println("\n📋 Example API calls to test logging:")
	fmt.Println("\n🟢 Health checks (will be skipped in logs):")
	fmt.Println("  curl http://localhost:8080/ping")
	fmt.Println("  curl http://localhost:8080/health")

	fmt.Println("\n🔵 Basic CRUD operations (will be logged):")
	fmt.Println("  curl http://localhost:8080/api/v1/users")
	fmt.Println("  curl http://localhost:8080/api/v1/users/1")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/users -H \"Content-Type: application/json\" -d '{\"name\":\"Dave\",\"email\":\"dave@example.com\"}'")
	fmt.Println("  curl -X PUT http://localhost:8080/api/v1/users/1 -H \"Content-Type: application/json\" -d '{\"name\":\"Alice Updated\",\"email\":\"alice.new@example.com\"}'")
	fmt.Println("  curl -X DELETE http://localhost:8080/api/v1/users/1")

	fmt.Println("\n🟡 Error scenarios:")
	fmt.Println("  curl http://localhost:8080/api/v1/users/404  # User not found")
	fmt.Println("  curl http://localhost:8080/api/v1/error     # Server error")
	fmt.Println("  curl http://localhost:8080/api/v1/panic     # Panic recovery")

	fmt.Println("\n🎯 Request with custom headers (for context):")
	fmt.Println("  curl -H \"X-Request-ID: my-custom-req-123\" -H \"X-User-ID: user-456\" http://localhost:8080/api/v1/users")

	fmt.Println("\n📝 All requests will generate structured logs showing:")
	fmt.Println("  • Request ID tracking")
	fmt.Println("  • User context (if provided)")
	fmt.Println("  • Request/response metrics")
	fmt.Println("  • Error logging and recovery")
	fmt.Println("  • Middleware execution timing")
	fmt.Println("  • Request body logging (for POST/PUT)")

	fmt.Println("\n🛑 Press Ctrl+C to gracefully shutdown the server")
	fmt.Println("="*60 + "\n")
}
