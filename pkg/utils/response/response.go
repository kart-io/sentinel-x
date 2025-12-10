// Package response provides unified API response structures.
// This package defines standard response formats for HTTP APIs,
// ensuring consistent response structures across all endpoints.
package response

import (
	"net/http"
	"sync"

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

// responsePool is a sync.Pool for Response objects to reduce memory allocations.
// This significantly reduces GC pressure in high-throughput scenarios (10K+ RPS).
var responsePool = sync.Pool{
	New: func() interface{} {
		return &Response{}
	},
}

// Acquire retrieves a Response object from the pool.
// The returned Response should be released back to the pool using Release() after use.
func Acquire() *Response {
	return responsePool.Get().(*Response)
}

// Release returns a Response object to the pool after resetting its fields.
// This prevents data leakage between requests.
func Release(r *Response) {
	if r == nil {
		return
	}
	// Reset all fields to zero values
	r.Code = 0
	r.HTTPCode = 0
	r.Message = ""
	r.Data = nil
	r.RequestID = ""
	r.Timestamp = 0
	responsePool.Put(r)
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
// Note: The returned Response uses object pooling. It should be released
// using Release() after the response is written to avoid memory leaks.
func Success(data interface{}) *Response {
	resp := Acquire()
	resp.Code = 0
	resp.HTTPCode = http.StatusOK
	resp.Message = "success"
	resp.Data = data
	return resp
}

// SuccessWithMessage creates a successful response with custom message.
// Note: The returned Response uses object pooling. It should be released
// using Release() after the response is written to avoid memory leaks.
func SuccessWithMessage(message string, data interface{}) *Response {
	resp := Acquire()
	resp.Code = 0
	resp.HTTPCode = http.StatusOK
	resp.Message = message
	resp.Data = data
	return resp
}

// Err creates an error response from an Errno type.
// Note: The returned Response uses object pooling. It should be released
// using Release() after the response is written to avoid memory leaks.
func Err(e *errors.Errno) *Response {
	if e == nil {
		return Success(nil)
	}
	resp := Acquire()
	resp.Code = e.Code
	resp.HTTPCode = e.HTTPStatus()
	resp.Message = e.MessageEN
	return resp
}

// ErrWithLang creates an error response with language-specific message.
// Note: The returned Response uses object pooling. It should be released
// using Release() after the response is written to avoid memory leaks.
func ErrWithLang(e *errors.Errno, lang string) *Response {
	if e == nil {
		return Success(nil)
	}
	resp := Acquire()
	resp.Code = e.Code
	resp.HTTPCode = e.HTTPStatus()
	resp.Message = e.Message(lang)
	return resp
}

// ErrorWithCode creates an error response with code and message.
// Note: The returned Response uses object pooling. It should be released
// using Release() after the response is written to avoid memory leaks.
func ErrorWithCode(code int, message string) *Response {
	resp := Acquire()
	resp.Code = code
	resp.Message = message
	resp.HTTPCode = resp.HTTPStatus()
	return resp
}

// ErrorWithData creates an error response with additional data.
// Note: The returned Response uses object pooling. It should be released
// using Release() after the response is written to avoid memory leaks.
func ErrorWithData(code int, message string, data interface{}) *Response {
	resp := Acquire()
	resp.Code = code
	resp.Message = message
	resp.Data = data
	resp.HTTPCode = resp.HTTPStatus()
	return resp
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
