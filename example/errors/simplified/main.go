// Package example demonstrates the simplified error API.
//
// This example shows how to use the new simplified error creation functions.
package main

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"

	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

const ServiceExample = 50

func init() {
	errors.RegisterService(ServiceExample, "example-service")
}

// ============================================================================
// Error Definitions
// ============================================================================

// Standard category errors - simple and clean!
var (
	ErrNotFound = errors.NewNotFoundErr(ServiceExample, 1,
		"Resource not found", "资源未找到")

	ErrInvalidInput = errors.NewRequestErr(ServiceExample, 2,
		"Invalid input", "输入无效")

	ErrConflict = errors.NewConflictErr(ServiceExample, 3,
		"Resource conflict", "资源冲突")

	ErrInternal = errors.NewInternalErr(ServiceExample, 4,
		"Internal error", "内部错误")
)

// Custom HTTP/gRPC codes - also simple!
var ErrCustom = errors.NewError(ServiceExample, errors.CategoryResource, 5,
	http.StatusGone, codes.NotFound,
	"Resource has expired", "资源已过期")

// ============================================================================
// Demo
// ============================================================================

func main() {
	fmt.Println("=== Error Definitions ===")
	fmt.Printf("ErrNotFound: Code=%d, Message=%s\n", ErrNotFound.Code, ErrNotFound.MessageEN)
	fmt.Printf("ErrInvalidInput: Code=%d, Message=%s\n", ErrInvalidInput.Code, ErrInvalidInput.MessageEN)
	fmt.Printf("ErrConflict: Code=%d, Message=%s\n", ErrConflict.Code, ErrConflict.MessageEN)
	fmt.Printf("ErrInternal: Code=%d, Message=%s\n", ErrInternal.Code, ErrInternal.MessageEN)
	fmt.Printf("ErrCustom: Code=%d, Message=%s\n", ErrCustom.Code, ErrCustom.MessageEN)

	fmt.Println("\n=== Error Creation Benefits ===")
	fmt.Println("- Single function call")
	fmt.Println("- Less code")
	fmt.Println("- Direct creation")
	fmt.Println("- Type-safe")

	fmt.Println("\n=== Runtime Usage ===")

	// Use errors with dynamic messages
	err1 := ErrNotFound.WithMessagef("user %s not found", "alice")
	err2 := ErrInvalidInput.WithCause(fmt.Errorf("invalid email format"))

	fmt.Printf("Error 1: %s\n", err1.Error())
	fmt.Printf("Error 2: %s\n", err2.Error())

	// Access error properties
	fmt.Printf("HTTP Status: %d\n", ErrNotFound.HTTPStatus())
	fmt.Printf("gRPC Code: %s\n", ErrNotFound.GRPCStatus().String())
	fmt.Printf("Chinese Message: %s\n", ErrNotFound.Message("zh"))
}
