package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/factory"
	"github.com/kart-io/logger/option"
	"github.com/kart-io/logger/reload"
)

func main() {
	fmt.Println("=== Gin + Dynamic Config Reload Example ===")

	// 1. Setup initial configuration
	configFile := "logger-config.yaml"
	if err := createInitialConfig(configFile); err != nil {
		panic(fmt.Sprintf("Failed to create initial config: %v", err))
	}

	initialOpt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
	}

	// 2. Create logger factory and initial logger
	factory := factory.NewLoggerFactory(initialOpt)
	coreLogger, err := factory.CreateLogger()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	// Set as global logger
	logger.SetGlobal(coreLogger)

	// 3. Setup configuration reloader
	reloadConfig := &reload.ReloadConfig{
		ConfigFile:           configFile,
		Triggers:             reload.TriggerAll, // File watch + Signal + API
		Signals:              []os.Signal{syscall.SIGUSR1, syscall.SIGHUP},
		ValidateBeforeReload: true,
		ReloadTimeout:        10 * time.Second,
		BackupOnReload:       true,
		BackupRetention:      5,
		Callback: func(oldConfig, newConfig *option.LogOption) error {
			fmt.Printf("🔄 配置已重载: %s→%s, %s→%s, %s→%s\n",
				oldConfig.Engine, newConfig.Engine,
				oldConfig.Level, newConfig.Level,
				oldConfig.Format, newConfig.Format,
			)
			return nil
		},
		Logger: coreLogger,
	}

	reloader, err := reload.NewConfigReloader(reloadConfig, initialOpt, factory)
	if err != nil {
		panic(fmt.Sprintf("Failed to create reloader: %v", err))
	}

	// Start the reloader
	if err := reloader.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start reloader: %v", err))
	}
	defer reloader.Stop()

	// 4. Setup Gin with unified logger
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Add logger middleware
	r.Use(GinLoggerMiddleware(coreLogger))
	r.Use(GinRecoveryMiddleware(coreLogger))

	// 5. Setup routes
	setupRoutes(r, coreLogger, reloader)

	// 6. Start HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		coreLogger.Infow("Starting Gin server with dynamic config reload",
			"addr", ":8080",
			"config_file", configFile,
			"triggers", "file_watch,signal,api")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			coreLogger.Errorw("Server failed to start", "error", err.Error())
		}
	}()

	// 7. Print usage instructions
	printUsageInstructions(configFile)

	// 8. Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	coreLogger.Infow("Shutting down server", "signal", "received")

	// 9. Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		coreLogger.Errorw("Server forced to shutdown", "error", err.Error())
	} else {
		coreLogger.Infow("Server exited gracefully")
	}

	// Cleanup config file
	os.Remove(configFile)
}

// GinLoggerMiddleware creates a Gin middleware that logs HTTP requests
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

		// Structured logging fields
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

		// Log based on status code with appropriate level
		message := fmt.Sprintf("%s %s", method, path)
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

// GinRecoveryMiddleware handles panics with structured logging
func GinRecoveryMiddleware(logger core.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorw("Panic recovered in Gin handler",
					"component", "gin",
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
					"panic_error", fmt.Sprintf("%v", err),
					"recovered", true,
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

func setupRoutes(r *gin.Engine, logger core.Logger, reloader *reload.ConfigReloader) {
	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		logger.Infow("Welcome endpoint accessed", "feature", "gin_reload_demo")
		c.JSON(http.StatusOK, gin.H{
			"message":        "Gin + Dynamic Config Reload Demo",
			"version":        "1.0.0",
			"time":           time.Now().Format(time.RFC3339),
			"current_config": getCurrentConfigSummary(reloader),
		})
	})

	// Configuration management endpoints
	configAPI := r.Group("/config")
	{
		// Get current configuration
		configAPI.GET("/current", func(c *gin.Context) {
			currentConfig := reloader.GetCurrentConfig()
			logger.Infow("Current config requested", "operation", "get_config")

			c.JSON(http.StatusOK, gin.H{
				"current_config": currentConfig,
				"backup_count":   len(reloader.GetBackupConfigs()),
			})
		})

		// Trigger config reload via API
		configAPI.POST("/reload", func(c *gin.Context) {
			var newConfig option.LogOption
			if err := c.ShouldBindJSON(&newConfig); err != nil {
				logger.Warnw("Invalid config reload request", "error", err.Error())
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid configuration",
					"code":  "INVALID_CONFIG",
				})
				return
			}

			logger.Infow("API config reload triggered",
				"old_engine", reloader.GetCurrentConfig().Engine,
				"new_engine", newConfig.Engine)

			if err := reloader.TriggerReload(&newConfig); err != nil {
				logger.Errorw("Failed to trigger config reload", "error", err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to reload configuration",
					"code":  "RELOAD_FAILED",
				})
				return
			}

			// Wait a bit for reload to complete
			time.Sleep(200 * time.Millisecond)

			c.JSON(http.StatusOK, gin.H{
				"message":    "Configuration reloaded successfully",
				"new_config": getCurrentConfigSummary(reloader),
			})
		})

		// Rollback to previous configuration
		configAPI.POST("/rollback", func(c *gin.Context) {
			logger.Infow("Config rollback requested", "operation", "rollback")

			if err := reloader.RollbackToPrevious(); err != nil {
				logger.Errorw("Failed to rollback configuration", "error", err.Error())
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
					"code":  "ROLLBACK_FAILED",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":        "Configuration rolled back successfully",
				"current_config": getCurrentConfigSummary(reloader),
			})
		})

		// Get backup configurations
		configAPI.GET("/backups", func(c *gin.Context) {
			backups := reloader.GetBackupConfigs()
			logger.Debugw("Config backups requested", "backup_count", len(backups))

			c.JSON(http.StatusOK, gin.H{
				"backup_count": len(backups),
				"backups":      backups,
			})
		})
	}

	// User routes with different logging patterns
	userRoutes := r.Group("/users")
	{
		userRoutes.GET("/:id", func(c *gin.Context) {
			userID := c.Param("id")

			logger.Debugw("User lookup started", "user_id", userID, "operation", "get_user")

			// Validate user ID
			if id, err := strconv.Atoi(userID); err != nil || id <= 0 {
				logger.Warnw("Invalid user ID format", "user_id", userID, "validation", "failed")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid user ID format",
					"code":  "INVALID_USER_ID",
				})
				return
			}

			// Set context for middleware
			c.Set("user_id", userID)

			// Simulate processing
			time.Sleep(50 * time.Millisecond)

			logger.Infow("User retrieved successfully", "user_id", userID)
			c.JSON(http.StatusOK, gin.H{
				"id":    userID,
				"name":  "Demo User " + userID,
				"email": "user" + userID + "@example.com",
				"role":  "member",
			})
		})

		userRoutes.POST("", func(c *gin.Context) {
			var user struct {
				Name  string `json:"name" binding:"required"`
				Email string `json:"email" binding:"required"`
			}

			if err := c.ShouldBindJSON(&user); err != nil {
				logger.Warnw("User creation validation failed",
					"error", err.Error(),
					"operation", "create_user")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Validation failed",
					"code":  "VALIDATION_ERROR",
				})
				return
			}

			userID := strconv.Itoa(int(time.Now().Unix() % 10000))

			logger.Infow("New user created",
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

	// Health and monitoring endpoints
	r.GET("/health", func(c *gin.Context) {
		logger.Debugw("Health check", "endpoint", "/health")
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
			"config": getCurrentConfigSummary(reloader),
		})
	})

	// Slow endpoint for latency testing
	r.GET("/slow", func(c *gin.Context) {
		logger.Debugw("Slow endpoint accessed", "expected_delay", "3s")
		time.Sleep(3 * time.Second)
		logger.Warnw("Slow operation completed", "operation", "slow_task")

		c.JSON(http.StatusOK, gin.H{
			"message": "Intentionally slow endpoint",
			"delay":   "3 seconds",
		})
	})

	// Error endpoint
	r.GET("/error", func(c *gin.Context) {
		logger.Errorw("Simulated error endpoint", "error_type", "intentional")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Simulated error for testing",
			"code":  "SIMULATED_ERROR",
		})
	})

	// Panic endpoint for recovery testing
	r.GET("/panic", func(c *gin.Context) {
		logger.Debugw("Panic endpoint accessed", "warning", "will_panic")
		panic("Intentional panic for testing recovery middleware")
	})
}

func createInitialConfig(filename string) error {
	configContent := `# Kart Logger 配置文件 - 支持动态重载
# 修改此文件后会自动重载配置

engine: "slog"              # 日志引擎: "slog" | "zap"
level: "INFO"               # 日志级别: "DEBUG" | "INFO" | "WARN" | "ERROR"
format: "json"              # 输出格式: "json" | "console"
output_paths: ["stdout"]    # 输出路径
development: false          # 开发模式

# OTLP 配置 (可选)
# otlp_endpoint: "http://localhost:4317"
# otlp:
#   enabled: true
#   protocol: "grpc"
#   timeout: "10s"
`

	return os.WriteFile(filename, []byte(configContent), 0644)
}

func getCurrentConfigSummary(reloader *reload.ConfigReloader) map[string]interface{} {
	config := reloader.GetCurrentConfig()
	return map[string]interface{}{
		"engine":       config.Engine,
		"level":        config.Level,
		"format":       config.Format,
		"development":  config.Development,
		"otlp_enabled": config.OTLP != nil && config.OTLP.IsEnabled(),
	}
}

func printUsageInstructions(configFile string) {
	absPath, _ := filepath.Abs(configFile)

	fmt.Println("\n🚀 Server started on :8080")
	fmt.Println("\n📡 API Endpoints:")
	fmt.Println("  GET  http://localhost:8080/                     - Welcome + current config")
	fmt.Println("  GET  http://localhost:8080/users/123            - Get user")
	fmt.Println("  POST http://localhost:8080/users                - Create user")
	fmt.Println("  GET  http://localhost:8080/health               - Health check")
	fmt.Println("  GET  http://localhost:8080/slow                 - Slow endpoint (3s)")
	fmt.Println("  GET  http://localhost:8080/error                - Error endpoint")
	fmt.Println("  GET  http://localhost:8080/panic                - Panic endpoint")
	fmt.Println("\n🔧 Configuration Management:")
	fmt.Println("  GET  http://localhost:8080/config/current       - Get current config")
	fmt.Println("  POST http://localhost:8080/config/reload        - Trigger config reload")
	fmt.Println("  POST http://localhost:8080/config/rollback      - Rollback to previous")
	fmt.Println("  GET  http://localhost:8080/config/backups       - Get backup configs")

	fmt.Println("\n🔄 Dynamic Config Reload Options:")
	fmt.Printf("  1. 📁 File Watch: Edit %s\n", absPath)
	fmt.Printf("  2. 📡 Signal: kill -USR1 %d\n", os.Getpid())
	fmt.Println("  3. 🌐 HTTP API: POST /config/reload")

	fmt.Println("\n💡 Try these config changes:")
	fmt.Println(`  # Switch to Zap engine with DEBUG level
  engine: "zap"
  level: "DEBUG"
  format: "console"
  development: true`)

	fmt.Println("\n❌ Press Ctrl+C to stop")
}
