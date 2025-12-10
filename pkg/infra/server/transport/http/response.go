package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// JSON sends a JSON response.
func (c *RequestContext) JSON(code int, v interface{}) {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.writer.WriteHeader(code)
	c.statusCode = code
	c.written = true

	if v != nil {
		if err := json.NewEncoder(c.writer).Encode(v); err != nil {
			// Log error but response is already committed
			_ = err
		}
	}
}

// String sends a string response.
func (c *RequestContext) String(code int, s string) {
	c.SetHeader("Content-Type", "text/plain; charset=utf-8")
	c.writer.WriteHeader(code)
	c.statusCode = code
	c.written = true
	_, _ = c.writer.Write([]byte(s))
}

// Bytes sends a raw bytes response.
func (c *RequestContext) Bytes(code int, contentType string, data []byte) {
	c.SetHeader("Content-Type", contentType)
	c.writer.WriteHeader(code)
	c.statusCode = code
	c.written = true
	_, _ = c.writer.Write(data)
}

// Error sends an error response as JSON.
func (c *RequestContext) Error(code int, err error) {
	c.JSON(code, map[string]string{"error": err.Error()})
}

// NoContent sends a no content response.
func (c *RequestContext) NoContent(code int) {
	c.writer.WriteHeader(code)
	c.statusCode = code
	c.written = true
}

// Redirect sends a redirect response.
func (c *RequestContext) Redirect(code int, url string) {
	http.Redirect(c.writer, c.request, url, code)
	c.statusCode = code
	c.written = true
}

// Bind binds the request body to the given struct.
// Supports JSON content type.
func (c *RequestContext) Bind(v interface{}) error {
	contentType := c.Header("Content-Type")

	// Default to JSON if no content type specified
	if contentType == "" || contentType == "application/json" ||
		len(contentType) > 16 && contentType[:16] == "application/json" {
		return json.NewDecoder(c.request.Body).Decode(v)
	}

	// For other content types, try JSON as fallback
	return json.NewDecoder(c.request.Body).Decode(v)
}

// Validate validates the given struct using the global validator.
// Returns nil if validation passes, or *validator.ValidationErrors if validation fails.
func (c *RequestContext) Validate(v interface{}) error {
	verr := validator.Global().ValidateWithLang(v, c.Lang())
	if verr == nil || !verr.HasErrors() {
		return nil
	}
	return verr
}

// ShouldBindAndValidate binds and validates the request body.
// Returns nil if both binding and validation pass.
func (c *RequestContext) ShouldBindAndValidate(v interface{}) error {
	if err := c.Bind(v); err != nil {
		return err
	}
	return c.Validate(v)
}

// MustBindAndValidate binds and validates, returning first error message if failed.
// Returns (errorMessage, false) if failed, ("", true) if succeeded.
func (c *RequestContext) MustBindAndValidate(v interface{}) (string, bool) {
	if err := c.Bind(v); err != nil {
		return "invalid request body: " + err.Error(), false
	}

	verr := validator.Global().ValidateWithLang(v, c.Lang())
	if verr != nil && verr.HasErrors() {
		return verr.First(), false
	}

	return "", true
}

// Lang returns the language preference from Accept-Language header or query param.
func (c *RequestContext) Lang() string {
	if c.lang != "" {
		return c.lang
	}

	// Check query parameter first
	if lang := c.Query("lang"); lang != "" {
		return lang
	}

	// Check Accept-Language header
	acceptLang := c.Header("Accept-Language")
	if acceptLang != "" {
		// Parse Accept-Language header (simplified)
		// Format: zh-CN,zh;q=0.9,en;q=0.8
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])
			if strings.HasPrefix(lang, "zh") {
				return validator.LangZH
			}
			if strings.HasPrefix(lang, "en") {
				return validator.LangEN
			}
		}
	}

	return validator.LangEN // Default to English
}

// SetLang sets the language for this request context.
func (c *RequestContext) SetLang(lang string) {
	c.lang = lang
}
