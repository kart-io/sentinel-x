package validator

import (
	"fmt"
	"strings"
)

// ValidationErrors represents a collection of validation errors.
type ValidationErrors struct {
	Errors []FieldError `json:"errors"`
}

// FieldError represents a single field validation error.
type FieldError struct {
	Field   string      `json:"field"`           // Field name (from JSON/form tag)
	Tag     string      `json:"tag"`             // Validation tag that failed
	Value   interface{} `json:"value,omitempty"` // Actual value that failed
	Param   string      `json:"param,omitempty"` // Validation parameter
	Message string      `json:"message"`         // Human-readable error message
}

// Error implements the error interface.
func (v *ValidationErrors) Error() string {
	if v == nil || len(v.Errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("validation failed: ")

	for i, fe := range v.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fe.Message)
	}

	return sb.String()
}

// HasErrors returns true if there are validation errors.
func (v *ValidationErrors) HasErrors() bool {
	return v != nil && len(v.Errors) > 0
}

// Count returns the number of validation errors.
func (v *ValidationErrors) Count() int {
	if v == nil {
		return 0
	}
	return len(v.Errors)
}

// First returns the first error message, or empty string if no errors.
func (v *ValidationErrors) First() string {
	if v == nil || len(v.Errors) == 0 {
		return ""
	}
	return v.Errors[0].Message
}

// FirstField returns the first error's field name, or empty string if no errors.
func (v *ValidationErrors) FirstField() string {
	if v == nil || len(v.Errors) == 0 {
		return ""
	}
	return v.Errors[0].Field
}

// Messages returns all error messages as a slice.
func (v *ValidationErrors) Messages() []string {
	if v == nil || len(v.Errors) == 0 {
		return nil
	}

	messages := make([]string, len(v.Errors))
	for i, fe := range v.Errors {
		messages[i] = fe.Message
	}
	return messages
}

// ByField returns errors grouped by field name.
func (v *ValidationErrors) ByField() map[string][]string {
	if v == nil || len(v.Errors) == 0 {
		return nil
	}

	result := make(map[string][]string)
	for _, fe := range v.Errors {
		result[fe.Field] = append(result[fe.Field], fe.Message)
	}
	return result
}

// ForField returns all error messages for a specific field.
func (v *ValidationErrors) ForField(field string) []string {
	if v == nil || len(v.Errors) == 0 {
		return nil
	}

	var messages []string
	for _, fe := range v.Errors {
		if fe.Field == field {
			messages = append(messages, fe.Message)
		}
	}
	return messages
}

// ToMap converts validation errors to a map suitable for JSON response.
func (v *ValidationErrors) ToMap() map[string]interface{} {
	if v == nil || len(v.Errors) == 0 {
		return nil
	}

	return map[string]interface{}{
		"errors": v.Errors,
		"count":  len(v.Errors),
	}
}

// String implements fmt.Stringer interface.
func (v *ValidationErrors) String() string {
	return v.Error()
}

// Format implements fmt.Formatter interface for custom formatting.
func (v *ValidationErrors) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			// Detailed format with all fields
			_, _ = fmt.Fprintf(f, "ValidationErrors(%d):\n", v.Count())
			for i, fe := range v.Errors {
				_, _ = fmt.Fprintf(f, "  [%d] %s: %s (tag=%s", i, fe.Field, fe.Message, fe.Tag)
				if fe.Param != "" {
					_, _ = fmt.Fprintf(f, ", param=%s", fe.Param)
				}
				if fe.Value != nil {
					_, _ = fmt.Fprintf(f, ", value=%v", fe.Value)
				}
				_, _ = fmt.Fprint(f, ")\n")
			}
		} else {
			_, _ = fmt.Fprint(f, v.Error())
		}
	case 's':
		_, _ = fmt.Fprint(f, v.Error())
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", v.Error())
	}
}

// Append adds a field error to the collection.
func (v *ValidationErrors) Append(field, tag, message string) {
	v.Errors = append(v.Errors, FieldError{
		Field:   field,
		Tag:     tag,
		Message: message,
	})
}

// AppendError adds a FieldError to the collection.
func (v *ValidationErrors) AppendError(fe FieldError) {
	v.Errors = append(v.Errors, fe)
}

// NewValidationError creates a new ValidationErrors with a single error.
func NewValidationError(field, tag, message string) *ValidationErrors {
	return &ValidationErrors{
		Errors: []FieldError{
			{
				Field:   field,
				Tag:     tag,
				Message: message,
			},
		},
	}
}

// NewValidationErrors creates a new empty ValidationErrors.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]FieldError, 0),
	}
}
