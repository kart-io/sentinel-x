package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	println("=== Web Frameworks Comparison Example ===")
	println("Running both Gin and Echo servers simultaneously")

	// Setup unified logger
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		Development: false,
	}

	coreLogger, err := logger.New(opt)
	if err != nil {
		panic("Failed to create logger: " + err.Error())
	}

	logger.SetGlobal(coreLogger)

	// Setup servers
	ginServer := setupGinServer(coreLogger)
	echoServer := setupEchoServer(coreLogger)

	// Start both servers
	var wg sync.WaitGroup

	// Start Gin server
	wg.Add(1)
	go func() {
		defer wg.Done()
		coreLogger.Infow("Starting Gin server", "addr", ":8080", "framework", "gin")
		if err := ginServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			coreLogger.Errorw("Gin server error", "error", err.Error())
		}
	}()

	// Start Echo server
	wg.Add(1)
	go func() {
		defer wg.Done()
		coreLogger.Infow("Starting Echo server", "addr", ":8081", "framework", "echo")
		if err := echoServer.Start(":8081"); err != nil && err != http.ErrServerClosed {
			coreLogger.Errorw("Echo server error", "error", err.Error())
		}
	}()

	println("Servers started:")
	println("  Gin:  http://localhost:8080")
	println("  Echo: http://localhost:8081")
	println()
	println("Comparison endpoints:")
	println("  GET http://localhost:8080/compare - Gin framework info")
	println("  GET http://localhost:8081/compare - Echo framework info")
	println("  GET http://localhost:8080/test/123 - Gin test endpoint")
	println("  GET http://localhost:8081/test/123 - Echo test endpoint")
	println("Press Ctrl+C to stop both servers")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	coreLogger.Infow("Shutting down both servers", "signal", "received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown both servers
	go func() {
		if err := ginServer.Shutdown(ctx); err != nil {
			coreLogger.Errorw("Gin server forced to shutdown", "error", err.Error())
		} else {
			coreLogger.Infow("Gin server exited gracefully")
		}
	}()

	go func() {
		if err := echoServer.Shutdown(ctx); err != nil {
			coreLogger.Errorw("Echo server forced to shutdown", "error", err.Error())
		} else {
			coreLogger.Infow("Echo server exited gracefully")
		}
	}()

	wg.Wait()
	coreLogger.Infow("All servers stopped")
}

func setupGinServer(logger core.Logger) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Custom Gin middleware
	r.Use(func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latency := time.Since(startTime)

		logger.Infow("Gin Request",
			"framework", "gin",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status_code", c.Writer.Status(),
			"latency_ms", float64(latency.Nanoseconds())/1e6,
			"client_ip", c.ClientIP(),
		)
	})

	// Routes
	r.GET("/", func(c *gin.Context) {
		logger.Infow("Gin root endpoint accessed")
		c.JSON(http.StatusOK, gin.H{
			"framework": "Gin",
			"message":   "Hello from Gin with Unified Logger!",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   gin.Version,
		})
	})

	r.GET("/compare", func(c *gin.Context) {
		logger.Infow("Gin comparison endpoint accessed")
		c.JSON(http.StatusOK, gin.H{
			"framework": "Gin",
			"features": []string{
				"Fast HTTP router",
				"Middleware support",
				"JSON validation",
				"Route groups",
				"Custom recovery",
			},
			"performance":    "Very high",
			"learning_curve": "Easy",
		})
	})

	r.GET("/test/:id", func(c *gin.Context) {
		id := c.Param("id")
		logger.Infow("Gin test endpoint", "id", id)
		c.JSON(http.StatusOK, gin.H{
			"framework": "Gin",
			"id":        id,
			"message":   "Test data from Gin",
			"logged_by": "unified_logger",
		})
	})

	return &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
}

func setupEchoServer(logger core.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Custom Echo middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startTime := time.Now()
			err := next(c)
			latency := time.Since(startTime)

			logger.Infow("Echo Request",
				"framework", "echo",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"route", c.Path(),
				"status_code", c.Response().Status,
				"latency_ms", float64(latency.Nanoseconds())/1e6,
				"client_ip", c.RealIP(),
			)
			return err
		}
	})

	e.Use(middleware.CORS())

	// Routes
	e.GET("/", func(c echo.Context) error {
		logger.Infow("Echo root endpoint accessed")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"framework": "Echo",
			"message":   "Hello from Echo with Unified Logger!",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "v4",
		})
	})

	e.GET("/compare", func(c echo.Context) error {
		logger.Infow("Echo comparison endpoint accessed")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"framework": "Echo",
			"features": []string{
				"High performance",
				"Extensible middleware",
				"Route-level middleware",
				"Data binding",
				"Built-in HTTP/2 support",
			},
			"performance":    "Excellent",
			"learning_curve": "Moderate",
		})
	})

	e.GET("/test/:id", func(c echo.Context) error {
		id := c.Param("id")
		logger.Infow("Echo test endpoint", "id", id)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"framework": "Echo",
			"id":        id,
			"message":   "Test data from Echo",
			"logged_by": "unified_logger",
		})
	})

	return e
}
