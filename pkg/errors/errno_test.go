package errors

import (
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestMakeCode(t *testing.T) {
	tests := []struct {
		service  int
		category int
		sequence int
		expected int
	}{
		{0, 0, 0, 0},
		{0, 1, 1, 1001},
		{0, 2, 0, 2000},
		{2, 4, 1, 204001},
		{25, 1, 1, 2501001},
		{90, 7, 1, 9007001},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_%d_%d", tt.service, tt.category, tt.sequence), func(t *testing.T) {
			got := MakeCode(tt.service, tt.category, tt.sequence)
			if got != tt.expected {
				t.Errorf("MakeCode(%d, %d, %d) = %d, want %d",
					tt.service, tt.category, tt.sequence, got, tt.expected)
			}
		})
	}
}

func TestParseCode(t *testing.T) {
	tests := []struct {
		code             int
		expectedService  int
		expectedCategory int
		expectedSequence int
	}{
		{0, 0, 0, 0},
		{1001, 0, 1, 1},
		{2000, 0, 2, 0},
		{204001, 2, 4, 1},
		{2501001, 25, 1, 1},
		{9007001, 90, 7, 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.code), func(t *testing.T) {
			service, category, sequence := ParseCode(tt.code)
			if service != tt.expectedService || category != tt.expectedCategory || sequence != tt.expectedSequence {
				t.Errorf("ParseCode(%d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.code, service, category, sequence,
					tt.expectedService, tt.expectedCategory, tt.expectedSequence)
			}
		})
	}
}

func TestGetService(t *testing.T) {
	if got := GetService(2501001); got != 25 {
		t.Errorf("GetService(2501001) = %d, want 25", got)
	}
}

func TestGetCategory(t *testing.T) {
	if got := GetCategory(2501001); got != 1 {
		t.Errorf("GetCategory(2501001) = %d, want 1", got)
	}
}

func TestGetSequence(t *testing.T) {
	if got := GetSequence(2501001); got != 1 {
		t.Errorf("GetSequence(2501001) = %d, want 1", got)
	}
}

func TestIsSuccess(t *testing.T) {
	if !IsSuccess(0) {
		t.Error("IsSuccess(0) should be true")
	}
	if IsSuccess(1001) {
		t.Error("IsSuccess(1001) should be false")
	}
}

func TestIsClientError(t *testing.T) {
	// Request errors (category 1)
	if !IsClientError(1001) {
		t.Error("IsClientError(1001) should be true")
	}
	// Rate limit errors (category 6)
	if !IsClientError(6000) {
		t.Error("IsClientError(6000) should be true")
	}
	// Internal errors (category 7)
	if IsClientError(7000) {
		t.Error("IsClientError(7000) should be false")
	}
}

func TestIsServerError(t *testing.T) {
	// Internal errors (category 7)
	if !IsServerError(7000) {
		t.Error("IsServerError(7000) should be true")
	}
	// Config errors (category 12)
	if !IsServerError(12000) {
		t.Error("IsServerError(12000) should be true")
	}
	// Request errors (category 1)
	if IsServerError(1001) {
		t.Error("IsServerError(1001) should be false")
	}
}

func TestErrnoError(t *testing.T) {
	err := ErrInvalidParam
	expected := "errno 1001: Invalid parameter"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestErrnoErrorWithCause(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := ErrInvalidParam.WithCause(cause)

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}

	if err.Code != ErrInvalidParam.Code {
		t.Error("WithCause should preserve the code")
	}
}

func TestErrnoWithMessage(t *testing.T) {
	err := ErrInvalidParam.WithMessage("custom message")

	if err.MessageEN != "custom message" {
		t.Errorf("WithMessage should set MessageEN, got %q", err.MessageEN)
	}

	if err.Code != ErrInvalidParam.Code {
		t.Error("WithMessage should preserve the code")
	}
}

func TestErrnoWithMessagef(t *testing.T) {
	err := ErrInvalidParam.WithMessagef("param %s is invalid", "username")
	expected := "param username is invalid"

	if err.MessageEN != expected {
		t.Errorf("WithMessagef should set MessageEN to %q, got %q", expected, err.MessageEN)
	}
}

func TestErrnoMessage(t *testing.T) {
	err := &Errno{
		Code:      1001,
		MessageEN: "English message",
		MessageZH: "中文消息",
	}

	// Test English
	if got := err.Message("en"); got != "English message" {
		t.Errorf("Message(en) = %q, want %q", got, "English message")
	}

	// Test Chinese
	if got := err.Message("zh"); got != "中文消息" {
		t.Errorf("Message(zh) = %q, want %q", got, "中文消息")
	}

	if got := err.Message("zh-CN"); got != "中文消息" {
		t.Errorf("Message(zh-CN) = %q, want %q", got, "中文消息")
	}
}

func TestErrnoHTTPStatus(t *testing.T) {
	if got := ErrInvalidParam.HTTPStatus(); got != http.StatusBadRequest {
		t.Errorf("HTTPStatus() = %d, want %d", got, http.StatusBadRequest)
	}

	if got := ErrUnauthorized.HTTPStatus(); got != http.StatusUnauthorized {
		t.Errorf("HTTPStatus() = %d, want %d", got, http.StatusUnauthorized)
	}

	if got := ErrNotFound.HTTPStatus(); got != http.StatusNotFound {
		t.Errorf("HTTPStatus() = %d, want %d", got, http.StatusNotFound)
	}
}

func TestErrnoGRPCStatus(t *testing.T) {
	if got := ErrInvalidParam.GRPCStatus(); got != codes.InvalidArgument {
		t.Errorf("GRPCStatus() = %v, want %v", got, codes.InvalidArgument)
	}

	if got := ErrUnauthorized.GRPCStatus(); got != codes.Unauthenticated {
		t.Errorf("GRPCStatus() = %v, want %v", got, codes.Unauthenticated)
	}

	if got := ErrNotFound.GRPCStatus(); got != codes.NotFound {
		t.Errorf("GRPCStatus() = %v, want %v", got, codes.NotFound)
	}
}

func TestErrnoIs(t *testing.T) {
	err1 := ErrInvalidParam.WithMessage("custom")

	if !err1.Is(ErrInvalidParam) {
		t.Error("Is() should return true for same code")
	}

	if err1.Is(ErrNotFound) {
		t.Error("Is() should return false for different code")
	}
}

func TestIsCode(t *testing.T) {
	err := ErrInvalidParam.WithMessage("test")

	if !IsCode(err, ErrInvalidParam.Code) {
		t.Error("IsCode should return true")
	}

	if IsCode(err, ErrNotFound.Code) {
		t.Error("IsCode should return false")
	}
}

func TestGetCode(t *testing.T) {
	err := ErrInvalidParam.WithMessage("test")

	if got := GetCode(err); got != ErrInvalidParam.Code {
		t.Errorf("GetCode() = %d, want %d", got, ErrInvalidParam.Code)
	}

	// Test with non-Errno error
	if got := GetCode(fmt.Errorf("plain error")); got != -1 {
		t.Errorf("GetCode() for plain error = %d, want -1", got)
	}
}

func TestFromError(t *testing.T) {
	// Test with nil
	if got := FromError(nil); got != nil {
		t.Error("FromError(nil) should return nil")
	}

	// Test with Errno
	err := ErrInvalidParam.WithMessage("test")
	if got := FromError(err); got != err {
		t.Error("FromError should return Errno as-is")
	}

	// Test with plain error
	plainErr := fmt.Errorf("plain error")
	result := FromError(plainErr)
	if result.Code != ErrInternal.Code {
		t.Errorf("FromError(plain) should wrap as ErrInternal, got code %d", result.Code)
	}
	if result.Unwrap() != plainErr {
		t.Error("FromError should preserve the cause")
	}
}

func TestLookup(t *testing.T) {
	// Test existing code
	if e, ok := Lookup(ErrInvalidParam.Code); !ok || e != ErrInvalidParam {
		t.Error("Lookup should find registered errno")
	}

	// Test non-existing code
	if _, ok := Lookup(9999999); ok {
		t.Error("Lookup should return false for non-existing code")
	}
}

func TestRegistrySize(t *testing.T) {
	size := RegistrySize()
	if size == 0 {
		t.Error("RegistrySize should not be 0 after init")
	}
}

func TestGetAllRegistered(t *testing.T) {
	all := GetAllRegistered()
	if len(all) == 0 {
		t.Error("GetAllRegistered should return non-empty map")
	}

	// Verify it's a copy
	all[9999999] = &Errno{Code: 9999999}
	if _, ok := Lookup(9999999); ok {
		t.Error("GetAllRegistered should return a copy")
	}
}
