package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
)

func main() {
	println("=== Gin Web Framework Integration Example ===")

	// 1. Setup unified logger with OTLP configuration
	opt := &option.LogOption{
		Engine:      "slog", // Use slog for structured logging
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(true),    // 启用 OTLP 发送到收集器
			Endpoint: "127.0.0.1:4317", // gRPC 端点（修正格式）
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Headers: map[string]string{
				"service.name":    "gin-web-service",
				"service.version": "1.0.0",
				"framework":       "gin",
				"environment":     "development",
			},
		},
	}

	coreLogger, err := logger.New(opt)
	if err != nil {
		panic("Failed to create logger: " + err.Error())
	}

	// Set as global logger
	logger.SetGlobal(coreLogger)

	// 2. Setup Gin with custom logger middleware
	gin.SetMode(gin.ReleaseMode) // Use release mode for cleaner output
	r := gin.New()

	// Add our unified logger middleware
	r.Use(GinLoggerMiddleware(coreLogger))
	r.Use(GinRecoveryMiddleware(coreLogger))

	// 3. Setup routes with different logging scenarios
	setupRoutes(r, coreLogger)

	// 4. Start server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		coreLogger.Infow("Starting Gin server", "addr", ":8080", "framework", "gin")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			coreLogger.Errorw("Server failed to start", "error", err.Error())
		}
	}()

	println("Server started on :8080")
	println("Try these endpoints:")
	println("  GET  http://localhost:8080/")
	println("  GET  http://localhost:8080/users/123")
	println("  POST http://localhost:8080/users")
	println("  GET  http://localhost:8080/health")
	println("  GET  http://localhost:8080/slow")
	println("  GET  http://localhost:8080/error")
	println("  GET  http://localhost:8080/panic")
	println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	coreLogger.Infow("Shutting down server", "signal", "received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		coreLogger.Errorw("Server forced to shutdown", "error", err.Error())
	} else {
		coreLogger.Infow("Server exited gracefully")
	}
}

// GinLoggerMiddleware creates a Gin middleware that logs HTTP requests using our unified logger
func GinLoggerMiddleware(logger core.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate processing time
		latency := time.Since(startTime)

		// Get request info
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Determine log level based on status code
		logFields := []interface{}{
			"component", "gin",
			"method", method,
			"path", path,
			"status_code", statusCode,
			"latency_ms", float64(latency.Nanoseconds()) / 1e6,
			"client_ip", clientIP,
			"user_agent", userAgent,
		}

		// Add request ID if present
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			logFields = append(logFields, "request_id", requestID)
		}

		// Add user info if authenticated
		if userID, exists := c.Get("user_id"); exists {
			logFields = append(logFields, "user_id", userID)
		}

		// Log based on status code
		message := method + " " + path
		switch {
		case statusCode >= 400 && statusCode < 500:
			logger.Warnw(message, logFields...)
		case statusCode >= 500:
			logger.Errorw(message, logFields...)
		case latency > 5*time.Second:
			logger.Warnw(message, append(logFields, "slow_request", true)...)
		default:
			logger.Infow(message, logFields...)
		}
	}
}

// GinRecoveryMiddleware creates a Gin middleware that handles panics using our unified logger
func GinRecoveryMiddleware(logger core.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorw("Panic recovered",
					"component", "gin",
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
					"error", err,
					"panic", true,
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  "PANIC_RECOVERED",
				})
			}
		}()
		c.Next()
	}
}

func setupRoutes(r *gin.Engine, logger core.Logger) {
	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		logger.Infow("Welcome endpoint accessed", "user_agent", c.Request.UserAgent())
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Gin + Unified Logger Demo",
			"version": "1.0.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// User routes group
	userRoutes := r.Group("/users")
	{
		// Get user by ID
		userRoutes.GET("/:id", func(c *gin.Context) {
			userID := c.Param("id")

			// Simulate user lookup with structured logging
			logger.Debugw("Looking up user", "user_id", userID, "operation", "get_user")

			// Simulate some processing time
			time.Sleep(50 * time.Millisecond)

			// Check if user ID is valid (for demo purposes)
			if id, err := strconv.Atoi(userID); err != nil || id <= 0 {
				logger.Warnw("Invalid user ID requested", "user_id", userID, "error", "invalid_format")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid user ID",
					"code":  "INVALID_USER_ID",
				})
				return
			}

			// Set user context for middleware logging
			c.Set("user_id", userID)

			logger.Infow("User retrieved successfully", "user_id", userID)
			c.JSON(http.StatusOK, gin.H{
				"id":    userID,
				"name":  "Demo User " + userID,
				"email": "user" + userID + "@example.com",
				"role":  "member",
			})
		})

		// Create user
		userRoutes.POST("", func(c *gin.Context) {
			var user struct {
				Name  string `json:"name" binding:"required"`
				Email string `json:"email" binding:"required"`
			}

			if err := c.ShouldBindJSON(&user); err != nil {
				logger.Warnw("Invalid user creation request", "error", err.Error(), "validation", "failed")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request body",
					"code":  "VALIDATION_ERROR",
				})
				return
			}

			// Simulate user creation
			userID := strconv.Itoa(int(time.Now().Unix() % 10000))

			logger.Infow("User created successfully",
				"user_id", userID,
				"name", user.Name,
				"email", user.Email,
				"operation", "create_user")

			c.JSON(http.StatusCreated, gin.H{
				"id":      userID,
				"name":    user.Name,
				"email":   user.Email,
				"created": time.Now().Format(time.RFC3339),
			})
		})
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		logger.Debugw("Health check requested", "endpoint", "/health")
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
			"uptime": "unknown", // In real app, calculate actual uptime
		})
	})

	// Slow endpoint for testing latency logging
	r.GET("/slow", func(c *gin.Context) {
		logger.Debugw("Slow endpoint accessed", "expected_delay", "3s")

		// Simulate slow operation
		time.Sleep(3 * time.Second)

		logger.Warnw("Slow operation completed", "operation", "slow_task", "duration", "3s")
		c.JSON(http.StatusOK, gin.H{
			"message": "This was intentionally slow",
			"delay":   "3 seconds",
		})
	})

	// Error endpoint for testing error logging
	r.GET("/error", func(c *gin.Context) {
		logger.Errorw("Simulated error endpoint", "error_type", "intentional", "test", true)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "This is an intentional error for testing",
			"code":  "SIMULATED_ERROR",
		})
	})

	// Panic endpoint for testing recovery middleware
	r.GET("/panic", func(c *gin.Context) {
		logger.Debugw("Panic endpoint accessed", "warning", "will_panic")
		panic("This is an intentional panic for testing recovery middleware")
	})

	// API documentation endpoint
	r.GET("/docs", func(c *gin.Context) {
		logger.Infow("API documentation accessed")
		c.JSON(http.StatusOK, gin.H{
			"title":       "Gin + Unified Logger API",
			"description": "Demonstration of unified logger integration with Gin framework",
			"endpoints": map[string]string{
				"GET /":          "Welcome message",
				"GET /users/:id": "Get user by ID",
				"POST /users":    "Create new user",
				"GET /health":    "Health check",
				"GET /slow":      "Slow endpoint (3s delay)",
				"GET /error":     "Error endpoint",
				"GET /panic":     "Panic endpoint (tests recovery)",
				"GET /docs":      "This documentation",
			},
			"features": []string{
				"Structured JSON logging",
				"Request/response logging",
				"Panic recovery",
				"Performance monitoring",
				"User context tracking",
			},
		})
	})
}

// Helper function to create boolean pointers
func boolPtr(b bool) *bool {
	return &b
}
