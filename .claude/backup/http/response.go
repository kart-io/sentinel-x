package http

import (
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/kart-io/sentinel-x/pkg/utils/json"
	"github.com/kart-io/sentinel-x/pkg/utils/validator"
)

// JSON sends a JSON response.
// Uses high-performance sonic JSON encoder when available.
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

// Bind binds the request to the given struct.
// For GET/DELETE/HEAD requests, binds from query parameters.
// For POST/PUT/PATCH requests, binds from body (JSON/Form) based on Content-Type.
func (c *RequestContext) Bind(v interface{}) error {
	method := c.request.Method
	contentType := c.Header("Content-Type")

	// For GET, DELETE, HEAD requests, bind from query parameters
	if method == "GET" || method == "DELETE" || method == "HEAD" {
		return bindForm(c.request, v)
	}

	// Handle Multipart Form
	if strings.Contains(contentType, "multipart/form-data") {
		return binding.FormMultipart.Bind(c.request, v)
	}

	// Handle URL Encoded Form - use custom bindForm to support json tags (for protobuf structs)
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return bindForm(c.request, v)
	}

	// Default to JSON
	if err := json.NewDecoder(c.request.Body).Decode(v); err != nil {
		if err == io.EOF {
			return &bindingError{msg: "request body is empty"}
		}
		return err
	}
	return nil
}

// Validator is an interface for types that can validate themselves.
type Validator interface {
	Validate() error
}

// Validate validates the given struct using the global validator or the struct's Validate method.
// Returns nil if validation passes, or error/ValidationErrors if validation fails.
func (c *RequestContext) Validate(v interface{}) error {
	// Check if the struct implements the Validator interface (e.g. Proto messages)
	if validator, ok := v.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	// Fallback to tag-based validation
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

// bindForm binds form data to the given struct.
// It supports both 'form' and 'json' struct tags for field name lookup.
// Priority: form tag > json tag > field name (lowercase).
// This is useful for protobuf-generated structs which only have json tags.
func bindForm(r *http.Request, v interface{}) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return &bindingError{msg: "bind target must be a non-nil pointer"}
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return &bindingError{msg: "bind target must be a pointer to struct"}
	}

	return bindFormToStruct(r.Form, val)
}

// bindFormToStruct recursively binds form values to struct fields.
func bindFormToStruct(form map[string][]string, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle embedded/anonymous structs
		if structField.Anonymous && field.Kind() == reflect.Struct {
			if err := bindFormToStruct(form, field); err != nil {
				return err
			}
			continue
		}

		// Skip protobuf internal fields
		if strings.HasPrefix(structField.Name, "XXX_") ||
			structField.Name == "state" ||
			structField.Name == "sizeCache" ||
			structField.Name == "unknownFields" {
			continue
		}

		// Get field name from tags (form > json > field name)
		fieldName := getFormFieldName(structField)

		// Get value from form
		values, ok := form[fieldName]
		if !ok || len(values) == 0 {
			continue
		}

		// Set field value
		if err := setFieldValue(field, values[0]); err != nil {
			return err
		}
	}

	return nil
}

// getFormFieldName returns the form field name for a struct field.
// Priority: form tag > json tag > lowercase field name.
func getFormFieldName(field reflect.StructField) string {
	// Check form tag first
	if tag := field.Tag.Get("form"); tag != "" && tag != "-" {
		return strings.Split(tag, ",")[0]
	}

	// Check json tag
	if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
		return strings.Split(tag, ",")[0]
	}

	// Fallback to lowercase field name
	return strings.ToLower(field.Name)
}

// setFieldValue sets a struct field value from a string.
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			return nil
		}
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			return nil
		}
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		if value == "" {
			return nil
		}
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		boolVal := value == "true" || value == "1" || value == "on"
		field.SetBool(boolVal)

	case reflect.Ptr:
		// Handle pointer types - create new value and set it
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValue(field.Elem(), value)
	}

	return nil
}

// bindingError represents a form binding error.
type bindingError struct {
	msg string
}

func (e *bindingError) Error() string {
	return e.msg
}
