// Package example demonstrates the simplified error API.
//
// This example shows how to use the new simplified error creation functions
// that replace the old Builder pattern.
package main

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"

	"github.com/kart-io/sentinel-x/pkg/errors"
)

const ServiceExample = 50

func init() {
	errors.RegisterService(ServiceExample, "example-service")
}

// ============================================================================
// Old Style (Deprecated but still works)
// ============================================================================

var OldStyleError = errors.NewNotFoundError(ServiceExample, 1).
	Message("Old style error", "旧风格错误").
	MustBuild()

var OldStyleCustom = errors.NewBuilder(ServiceExample, errors.CategoryRequest, 2).
	HTTP(http.StatusBadRequest).
	GRPC(codes.InvalidArgument).
	Message("Old style custom", "旧风格自定义").
	MustBuild()

// ============================================================================
// New Style (Recommended)
// ============================================================================

// Standard category errors - much simpler!
var (
	NewStyleNotFound = errors.NewNotFoundErr(ServiceExample, 10,
		"New style not found", "新风格未找到")

	NewStyleInvalid = errors.NewRequestErr(ServiceExample, 11,
		"New style invalid", "新风格无效")

	NewStyleConflict = errors.NewConflictErr(ServiceExample, 12,
		"New style conflict", "新风格冲突")

	NewStyleInternal = errors.NewInternalErr(ServiceExample, 13,
		"New style internal", "新风格内部")
)

// Custom HTTP/gRPC codes - also simpler!
var NewStyleCustom = errors.NewError(ServiceExample, errors.CategoryResource, 14,
	http.StatusGone, codes.NotFound,
	"New style custom", "新风格自定义")

// ============================================================================
// Comparison
// ============================================================================

func main() {
	fmt.Println("=== Old Style (Deprecated) ===")
	fmt.Printf("Code: %d, Message: %s\n", OldStyleError.Code, OldStyleError.MessageEN)
	fmt.Printf("Code: %d, Message: %s\n", OldStyleCustom.Code, OldStyleCustom.MessageEN)

	fmt.Println("\n=== New Style (Recommended) ===")
	fmt.Printf("Code: %d, Message: %s\n", NewStyleNotFound.Code, NewStyleNotFound.MessageEN)
	fmt.Printf("Code: %d, Message: %s\n", NewStyleInvalid.Code, NewStyleInvalid.MessageEN)
	fmt.Printf("Code: %d, Message: %s\n", NewStyleConflict.Code, NewStyleConflict.MessageEN)
	fmt.Printf("Code: %d, Message: %s\n", NewStyleInternal.Code, NewStyleInternal.MessageEN)
	fmt.Printf("Code: %d, Message: %s\n", NewStyleCustom.Code, NewStyleCustom.MessageEN)

	fmt.Println("\n=== Comparison ===")
	fmt.Println("Old Style:")
	fmt.Println("  - Requires method chaining")
	fmt.Println("  - More verbose")
	fmt.Println("  - Creates intermediate objects")
	fmt.Println("  - Still works but deprecated")
	fmt.Println("\nNew Style:")
	fmt.Println("  - Single function call")
	fmt.Println("  - Less code")
	fmt.Println("  - Direct creation")
	fmt.Println("  - Recommended for all new code")

	fmt.Println("\n=== Runtime Usage ===")

	// Both styles produce identical runtime behavior
	err1 := NewStyleNotFound.WithMessagef("user %s not found", "alice")
	err2 := NewStyleInvalid.WithCause(fmt.Errorf("invalid input"))

	fmt.Printf("Error 1: %s\n", err1.Error())
	fmt.Printf("Error 2: %s\n", err2.Error())

	// All existing functionality works
	fmt.Printf("HTTP Status: %d\n", NewStyleNotFound.HTTPStatus())
	fmt.Printf("gRPC Code: %s\n", NewStyleNotFound.GRPCStatus().String())
	fmt.Printf("Chinese Message: %s\n", NewStyleNotFound.Message("zh"))
}
