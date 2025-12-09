package http

import (
	"encoding/json"
	"net/http"
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
