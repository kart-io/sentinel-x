package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/infra/tracing"
)

func main() {
	// Create tracing configuration
	opts := tracing.NewOptions()
	opts.Enabled = true
	opts.ServiceName = "tracing-example"
	opts.ServiceVersion = "1.0.0"
	opts.Environment = "development"
	opts.ExporterType = tracing.ExporterStdout // Use stdout for demo
	opts.SamplerType = tracing.SamplerAlwaysOn

	// Initialize provider
	provider, err := tracing.NewProvider(opts)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracing: %v", err)
		}
	}()

	// Create a tracer
	tracer := provider.Tracer("example")

	// Example 1: Simple span
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "main-operation")
	defer span.End()

	// Add attributes
	tracing.AddSpanAttributes(ctx,
		tracing.String("example.type", "demonstration"),
		tracing.Int("example.version", 1),
	)

	// Example 2: Nested spans
	doWork(ctx)

	// Example 3: Error handling
	if err := doWorkWithError(ctx); err != nil {
		tracing.RecordError(ctx, err)
		log.Printf("Work failed: %v", err)
	}

	// Example 4: Extract trace information
	traceID := tracing.TraceIDFromContext(ctx)
	spanID := tracing.SpanIDFromContext(ctx)
	fmt.Printf("Trace ID: %s\n", traceID)
	fmt.Printf("Span ID: %s\n", spanID)

	// Mark main operation as successful
	tracing.SetSpanOK(ctx)

	// Force flush to ensure all spans are exported
	if err := provider.ForceFlush(context.Background()); err != nil {
		log.Printf("Failed to flush spans: %v", err)
	}

	fmt.Println("\nAll spans exported. Check stdout for JSON output.")
}

func doWork(ctx context.Context) {
	// Start a child span
	ctx, span := tracing.StartSpan(ctx, "example", "do-work")
	defer span.End()

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	// Add event
	tracing.AddSpanEvent(ctx, "processing-started",
		tracing.String("status", "in-progress"),
	)

	// More work
	processData(ctx)

	// Add success attributes
	tracing.AddSpanAttributes(ctx,
		tracing.String("work.status", "completed"),
		tracing.Int("work.items", 42),
	)

	tracing.SetSpanOK(ctx)
}

func processData(ctx context.Context) {
	// Another nested span
	ctx, span := tracing.StartSpan(ctx, "example", "process-data")
	defer span.End()

	// Simulate processing
	time.Sleep(5 * time.Millisecond)

	tracing.AddSpanAttributes(ctx,
		tracing.String("data.type", "json"),
		tracing.Int64("data.size", 1024),
	)

	tracing.SetSpanOK(ctx)
}

func doWorkWithError(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "example", "work-with-error")
	defer span.End()

	// Simulate an error
	err := fmt.Errorf("simulated error for demonstration")

	// Record the error
	tracing.RecordErrorWithStatus(ctx, err, "operation failed")

	return err
}
