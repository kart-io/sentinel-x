package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
)

func main() {
	println("=== Echo Web Framework Integration Example ===")

	// 1. Setup unified logger with OTLP configuration
	opt := &option.LogOption{
		Engine:      "zap", // Use zap for high performance
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
		OTLP: &option.OTLPOption{
			Enabled:  boolPtr(false),          // Enable if you have an OTLP collector running
			Endpoint: "http://127.0.0.1:4317", // gRPC endpoint for web service
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Headers: map[string]string{
				"service.name":    "echo-web-service",
				"service.version": "1.0.0",
				"framework":       "echo",
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

	// 2. Setup Echo with custom logger middleware
	e := echo.New()
	e.HideBanner = true // Hide Echo banner for cleaner output

	// Add our unified logger middleware
	e.Use(EchoLoggerMiddleware(coreLogger))
	e.Use(EchoRecoveryMiddleware(coreLogger))

	// Add CORS middleware for API access
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Request-ID"},
	}))

	// 3. Setup routes with different logging scenarios
	setupEchoRoutes(e, coreLogger)

	// 4. Start server
	coreLogger.Infow("Starting Echo server", "addr", ":8081", "framework", "echo")

	println("Server started on :8081")
	println("Try these endpoints:")
	println("  GET  http://localhost:8081/")
	println("  GET  http://localhost:8081/api/products/123")
	println("  POST http://localhost:8081/api/products")
	println("  GET  http://localhost:8081/health")
	println("  GET  http://localhost:8081/metrics")
	println("  GET  http://localhost:8081/slow")
	println("  GET  http://localhost:8081/error")
	println("  GET  http://localhost:8081/panic")
	println("Press Ctrl+C to stop")

	// Start server in goroutine
	go func() {
		if err := e.Start(":8081"); err != nil && err != http.ErrServerClosed {
			coreLogger.Errorw("Echo server failed to start", "error", err.Error())
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	coreLogger.Infow("Shutting down Echo server", "signal", "received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		coreLogger.Errorw("Echo server forced to shutdown", "error", err.Error())
	} else {
		coreLogger.Infow("Echo server exited gracefully")
	}
}

// EchoLoggerMiddleware creates an Echo middleware that logs HTTP requests using our unified logger
func EchoLoggerMiddleware(logger core.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startTime := time.Now()

			// Process request
			err := next(c)

			// Calculate processing time
			latency := time.Since(startTime)

			// Get request info
			req := c.Request()
			res := c.Response()
			method := req.Method
			path := c.Path()
			if path == "" {
				path = req.URL.Path
			}
			statusCode := res.Status
			clientIP := c.RealIP()
			userAgent := req.UserAgent()

			// Build log fields
			logFields := []interface{}{
				"component", "echo",
				"method", method,
				"path", path,
				"route", c.Path(), // Echo route pattern
				"status_code", statusCode,
				"latency_ms", float64(latency.Nanoseconds()) / 1e6,
				"client_ip", clientIP,
				"user_agent", userAgent,
				"bytes_in", req.ContentLength,
				"bytes_out", res.Size,
			}

			// Add request ID if present
			if requestID := req.Header.Get("X-Request-ID"); requestID != "" {
				logFields = append(logFields, "request_id", requestID)
			}

			// Add trace info if present
			if traceID := req.Header.Get("X-Trace-ID"); traceID != "" {
				logFields = append(logFields, "trace_id", traceID)
			}

			// Add user info if authenticated
			if userID := c.Get("user_id"); userID != nil {
				logFields = append(logFields, "user_id", userID)
			}

			// Add error info if present
			if err != nil {
				if echoErr, ok := err.(*echo.HTTPError); ok {
					logFields = append(logFields, "error_code", echoErr.Code, "error_message", echoErr.Message)
				} else {
					logFields = append(logFields, "error", err.Error())
				}
			}

			// Log based on status code and errors
			message := method + " " + path
			switch {
			case err != nil:
				logger.Errorw(message, logFields...)
			case statusCode >= 400 && statusCode < 500:
				logger.Warnw(message, logFields...)
			case statusCode >= 500:
				logger.Errorw(message, logFields...)
			case latency > 5*time.Second:
				logger.Warnw(message, append(logFields, "slow_request", true)...)
			default:
				logger.Infow(message, logFields...)
			}

			return err
		}
	}
}

// EchoRecoveryMiddleware creates an Echo middleware that handles panics using our unified logger
func EchoRecoveryMiddleware(logger core.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorw("Panic recovered",
						"component", "echo",
						"method", c.Request().Method,
						"path", c.Request().URL.Path,
						"route", c.Path(),
						"client_ip", c.RealIP(),
						"error", r,
						"panic", true,
					)

					// Return JSON error response
					c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"error":     "Internal server error",
						"code":      "PANIC_RECOVERED",
						"timestamp": time.Now().Format(time.RFC3339),
					})
				}
			}()
			return next(c)
		}
	}
}

func setupEchoRoutes(e *echo.Echo, logger core.Logger) {
	// Root endpoint
	e.GET("/", func(c echo.Context) error {
		logger.Infow("Welcome endpoint accessed", "user_agent", c.Request().UserAgent())
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":   "Welcome to Echo + Unified Logger Demo",
			"framework": "Echo v4",
			"version":   "1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// API group with prefix
	api := e.Group("/api")

	// Product routes
	products := api.Group("/products")
	{
		// Get product by ID
		products.GET("/:id", func(c echo.Context) error {
			productID := c.Param("id")

			// Simulate product lookup with structured logging
			logger.Debugw("Looking up product", "product_id", productID, "operation", "get_product")

			// Simulate some processing time
			time.Sleep(30 * time.Millisecond)

			// Check if product ID is valid (for demo purposes)
			if id, err := strconv.Atoi(productID); err != nil || id <= 0 {
				logger.Warnw("Invalid product ID requested", "product_id", productID, "error", "invalid_format")
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
			}

			// Set context for middleware logging
			c.Set("user_id", "demo_user_"+productID)

			logger.Infow("Product retrieved successfully", "product_id", productID)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"id":          productID,
				"name":        "Demo Product " + productID,
				"description": "This is a demo product for testing",
				"price":       99.99,
				"category":    "electronics",
				"in_stock":    true,
			})
		})

		// Create product
		products.POST("", func(c echo.Context) error {
			var product struct {
				Name        string  `json:"name" validate:"required"`
				Description string  `json:"description"`
				Price       float64 `json:"price" validate:"required,min=0"`
				Category    string  `json:"category" validate:"required"`
			}

			if err := c.Bind(&product); err != nil {
				logger.Warnw("Invalid product creation request", "error", err.Error(), "validation", "bind_failed")
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
			}

			// Basic validation
			if product.Name == "" || product.Price <= 0 || product.Category == "" {
				logger.Warnw("Product validation failed", "product", product, "validation", "failed")
				return echo.NewHTTPError(http.StatusBadRequest, "Missing required fields")
			}

			// Simulate product creation
			productID := strconv.Itoa(int(time.Now().Unix() % 10000))

			logger.Infow("Product created successfully",
				"product_id", productID,
				"name", product.Name,
				"price", product.Price,
				"category", product.Category,
				"operation", "create_product")

			return c.JSON(http.StatusCreated, map[string]interface{}{
				"id":          productID,
				"name":        product.Name,
				"description": product.Description,
				"price":       product.Price,
				"category":    product.Category,
				"created_at":  time.Now().Format(time.RFC3339),
			})
		})

		// Update product
		products.PUT("/:id", func(c echo.Context) error {
			productID := c.Param("id")

			var updates map[string]interface{}
			if err := c.Bind(&updates); err != nil {
				logger.Warnw("Invalid product update request", "error", err.Error(), "product_id", productID)
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
			}

			logger.Infow("Product updated successfully",
				"product_id", productID,
				"updates", updates,
				"operation", "update_product")

			return c.JSON(http.StatusOK, map[string]interface{}{
				"id":         productID,
				"message":    "Product updated successfully",
				"updates":    updates,
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})

		// Delete product
		products.DELETE("/:id", func(c echo.Context) error {
			productID := c.Param("id")

			logger.Infow("Product deleted",
				"product_id", productID,
				"operation", "delete_product")

			return c.JSON(http.StatusOK, map[string]interface{}{
				"message":    "Product deleted successfully",
				"product_id": productID,
				"deleted_at": time.Now().Format(time.RFC3339),
			})
		})
	}

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		logger.Debugw("Health check requested", "endpoint", "/health")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"framework": "echo",
			"timestamp": time.Now().Format(time.RFC3339),
			"uptime":    "unknown", // In real app, calculate actual uptime
		})
	})

	// Metrics endpoint
	e.GET("/metrics", func(c echo.Context) error {
		logger.Debugw("Metrics endpoint accessed")

		// Simulate gathering metrics
		metrics := map[string]interface{}{
			"requests_total":    12345,
			"requests_per_sec":  67.8,
			"avg_response_time": "45ms",
			"error_rate":        "0.2%",
			"uptime_seconds":    3600,
		}

		logger.Infow("Metrics retrieved", "metrics", metrics)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"metrics":   metrics,
		})
	})

	// Slow endpoint for testing latency logging
	e.GET("/slow", func(c echo.Context) error {
		delay := c.QueryParam("delay")
		if delay == "" {
			delay = "2"
		}

		delayInt, _ := strconv.Atoi(delay)
		if delayInt <= 0 || delayInt > 10 {
			delayInt = 2
		}

		duration := time.Duration(delayInt) * time.Second

		logger.Debugw("Slow endpoint accessed", "expected_delay", duration.String())

		// Simulate slow operation
		time.Sleep(duration)

		logger.Warnw("Slow operation completed", "operation", "slow_task", "actual_duration", duration.String())
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "This was intentionally slow",
			"delay":   duration.String(),
		})
	})

	// Error endpoint for testing error logging
	e.GET("/error", func(c echo.Context) error {
		errorType := c.QueryParam("type")
		if errorType == "" {
			errorType = "generic"
		}

		logger.Errorw("Simulated error endpoint", "error_type", errorType, "test", true)

		switch errorType {
		case "bad_request":
			return echo.NewHTTPError(http.StatusBadRequest, "This is a bad request error")
		case "unauthorized":
			return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
		case "forbidden":
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		case "not_found":
			return echo.NewHTTPError(http.StatusNotFound, "Resource not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "This is an intentional error for testing")
		}
	})

	// Panic endpoint for testing recovery middleware
	e.GET("/panic", func(c echo.Context) error {
		logger.Debugw("Panic endpoint accessed", "warning", "will_panic")
		panic("This is an intentional panic for testing recovery middleware")
	})

	// API documentation endpoint
	e.GET("/docs", func(c echo.Context) error {
		logger.Infow("API documentation accessed")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"title":       "Echo + Unified Logger API",
			"framework":   "Echo v4",
			"description": "Demonstration of unified logger integration with Echo framework",
			"endpoints": map[string]string{
				"GET /":                    "Welcome message",
				"GET /api/products/:id":    "Get product by ID",
				"POST /api/products":       "Create new product",
				"PUT /api/products/:id":    "Update product",
				"DELETE /api/products/:id": "Delete product",
				"GET /health":              "Health check",
				"GET /metrics":             "Application metrics",
				"GET /slow?delay=N":        "Slow endpoint (N seconds delay)",
				"GET /error?type=TYPE":     "Error endpoint (various error types)",
				"GET /panic":               "Panic endpoint (tests recovery)",
				"GET /docs":                "This documentation",
			},
			"features": []string{
				"Structured JSON logging",
				"Request/response logging with route patterns",
				"Panic recovery",
				"Performance monitoring",
				"User context tracking",
				"Error categorization",
				"Metrics collection",
			},
		})
	})
}

// Helper function to create boolean pointers
func boolPtr(b bool) *bool {
	return &b
}
