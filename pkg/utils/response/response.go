// Package response provides unified API response structures.
// This package defines standard response formats for HTTP APIs,
// ensuring consistent response structures across all endpoints.
package response

import (
	"net/http"

	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// Response is the unified API response structure.
// All API responses should use this format for consistency.
type Response struct {
	// Code is the business error code (0 = success)
	Code int `json:"code"`

	// HTTPCode is the HTTP status code (optional, for client convenience)
	HTTPCode int `json:"http_code,omitempty"`

	// Message is a human-readable message
	Message string `json:"message"`

	// Data contains the response payload (nil for errors)
	Data interface{} `json:"data,omitempty"`

	// RequestID is the unique request identifier for tracing
	RequestID string `json:"request_id,omitempty"`

	// Timestamp is the response timestamp (Unix milliseconds)
	Timestamp int64 `json:"timestamp,omitempty"`
}

// PageData represents paginated data.
type PageData struct {
	// List contains the data items
	List interface{} `json:"list"`

	// Total is the total number of items
	Total int64 `json:"total"`

	// Page is the current page number (1-based)
	Page int `json:"page"`

	// PageSize is the number of items per page
	PageSize int `json:"page_size"`

	// TotalPages is the total number of pages
	TotalPages int `json:"total_pages"`
}

// Success creates a successful response with data.
func Success(data interface{}) *Response {
	return &Response{
		Code:     0,
		HTTPCode: http.StatusOK,
		Message:  "success",
		Data:     data,
	}
}

// SuccessWithMessage creates a successful response with custom message.
func SuccessWithMessage(message string, data interface{}) *Response {
	return &Response{
		Code:     0,
		HTTPCode: http.StatusOK,
		Message:  message,
		Data:     data,
	}
}

// Err creates an error response from an Errno type.
func Err(e *errors.Errno) *Response {
	if e == nil {
		return Success(nil)
	}
	return &Response{
		Code:     e.Code,
		HTTPCode: e.HTTPStatus(),
		Message:  e.MessageEN,
	}
}

// ErrWithLang creates an error response with language-specific message.
func ErrWithLang(e *errors.Errno, lang string) *Response {
	if e == nil {
		return Success(nil)
	}
	return &Response{
		Code:     e.Code,
		HTTPCode: e.HTTPStatus(),
		Message:  e.Message(lang),
	}
}

// ErrorWithCode creates an error response with code and message.
func ErrorWithCode(code int, message string) *Response {
	r := &Response{
		Code:    code,
		Message: message,
	}
	r.HTTPCode = r.HTTPStatus()
	return r
}

// ErrorWithData creates an error response with additional data.
func ErrorWithData(code int, message string, data interface{}) *Response {
	r := &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
	r.HTTPCode = r.HTTPStatus()
	return r
}

// Page creates a paginated response.
func Page(list interface{}, total int64, page, pageSize int) *Response {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return Success(&PageData{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// WithRequestID adds request ID to the response.
func (r *Response) WithRequestID(requestID string) *Response {
	r.RequestID = requestID
	return r
}

// WithTimestamp adds timestamp to the response.
func (r *Response) WithTimestamp(timestamp int64) *Response {
	r.Timestamp = timestamp
	return r
}

// IsSuccess returns true if the response indicates success.
func (r *Response) IsSuccess() bool {
	return r.Code == 0
}

// HTTPStatus returns the appropriate HTTP status code for this response.
// It looks up the registered errno to get the correct HTTP status.
func (r *Response) HTTPStatus() int {
	// If HTTPCode is already set, use it
	if r.HTTPCode != 0 {
		return r.HTTPCode
	}

	if r.Code == 0 {
		return http.StatusOK
	}

	// Look up in registered errors to get HTTP status
	if e, ok := errors.Lookup(r.Code); ok {
		return e.HTTPStatus()
	}

	// Fallback: determine by category from error code
	category := errors.GetCategory(r.Code)
	switch category {
	case errors.CategoryRequest:
		return http.StatusBadRequest
	case errors.CategoryAuth:
		return http.StatusUnauthorized
	case errors.CategoryPermission:
		return http.StatusForbidden
	case errors.CategoryResource:
		return http.StatusNotFound
	case errors.CategoryConflict:
		return http.StatusConflict
	case errors.CategoryRateLimit:
		return http.StatusTooManyRequests
	case errors.CategoryTimeout:
		return http.StatusGatewayTimeout
	case errors.CategoryNetwork:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
