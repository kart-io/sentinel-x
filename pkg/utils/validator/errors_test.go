package validator

import (
	"fmt"
	"testing"
)

// TestValidationErrors_Error tests the Error() method.
func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want string
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: "",
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: "",
		},
		{
			name: "single_error",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "email is invalid"},
				},
			},
			want: "validation failed: email is invalid",
		},
		{
			name: "multiple_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "email is invalid"},
					{Field: "name", Tag: "required", Message: "name is required"},
				},
			},
			want: "validation failed: email is invalid; name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidationErrors_Count tests the Count() method.
func TestValidationErrors_Count(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want int
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: 0,
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: 0,
		},
		{
			name: "one_error",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "invalid"},
				},
			},
			want: 1,
		},
		{
			name: "three_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "invalid"},
					{Field: "name", Tag: "required", Message: "required"},
					{Field: "age", Tag: "gte", Message: "too low"},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.Count()
			if got != tt.want {
				t.Errorf("Count() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestValidationErrors_First tests the First() method.
func TestValidationErrors_First(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want string
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: "",
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: "",
		},
		{
			name: "has_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "first error"},
					{Field: "name", Tag: "required", Message: "second error"},
				},
			},
			want: "first error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.First()
			if got != tt.want {
				t.Errorf("First() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidationErrors_FirstField tests the FirstField() method.
func TestValidationErrors_FirstField(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want string
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: "",
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: "",
		},
		{
			name: "has_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "invalid"},
					{Field: "name", Tag: "required", Message: "required"},
				},
			},
			want: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.FirstField()
			if got != tt.want {
				t.Errorf("FirstField() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidationErrors_Messages tests the Messages() method.
func TestValidationErrors_Messages(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want []string
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: nil,
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: nil,
		},
		{
			name: "multiple_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "email invalid"},
					{Field: "name", Tag: "required", Message: "name required"},
				},
			},
			want: []string{"email invalid", "name required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.Messages()
			if len(got) != len(tt.want) {
				t.Errorf("Messages() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Messages()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestValidationErrors_ByField tests the ByField() method.
func TestValidationErrors_ByField(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want map[string][]string
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: nil,
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: nil,
		},
		{
			name: "single_field_single_error",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Message: "invalid format"},
				},
			},
			want: map[string][]string{
				"email": {"invalid format"},
			},
		},
		{
			name: "single_field_multiple_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Message: "required"},
					{Field: "email", Message: "invalid format"},
				},
			},
			want: map[string][]string{
				"email": {"required", "invalid format"},
			},
		},
		{
			name: "multiple_fields",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Message: "invalid"},
					{Field: "name", Message: "required"},
					{Field: "age", Message: "too low"},
				},
			},
			want: map[string][]string{
				"email": {"invalid"},
				"name":  {"required"},
				"age":   {"too low"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.ByField()
			if len(got) != len(tt.want) {
				t.Errorf("ByField() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for field, wantMsgs := range tt.want {
				gotMsgs, ok := got[field]
				if !ok {
					t.Errorf("ByField() missing field %q", field)
					continue
				}
				if len(gotMsgs) != len(wantMsgs) {
					t.Errorf("ByField()[%q] length = %d, want %d", field, len(gotMsgs), len(wantMsgs))
					continue
				}
				for i := range gotMsgs {
					if gotMsgs[i] != wantMsgs[i] {
						t.Errorf("ByField()[%q][%d] = %q, want %q", field, i, gotMsgs[i], wantMsgs[i])
					}
				}
			}
		})
	}
}

// TestValidationErrors_ForField tests the ForField() method.
func TestValidationErrors_ForField(t *testing.T) {
	errs := &ValidationErrors{
		Errors: []FieldError{
			{Field: "email", Message: "required"},
			{Field: "email", Message: "invalid format"},
			{Field: "name", Message: "required"},
		},
	}

	tests := []struct {
		name  string
		errs  *ValidationErrors
		field string
		want  []string
	}{
		{
			name:  "nil_errors",
			errs:  nil,
			field: "email",
			want:  nil,
		},
		{
			name:  "empty_errors",
			errs:  &ValidationErrors{Errors: []FieldError{}},
			field: "email",
			want:  nil,
		},
		{
			name:  "field_with_multiple_errors",
			errs:  errs,
			field: "email",
			want:  []string{"required", "invalid format"},
		},
		{
			name:  "field_with_single_error",
			errs:  errs,
			field: "name",
			want:  []string{"required"},
		},
		{
			name:  "field_not_found",
			errs:  errs,
			field: "age",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.ForField(tt.field)
			if len(got) != len(tt.want) {
				t.Errorf("ForField(%q) length = %d, want %d", tt.field, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ForField(%q)[%d] = %q, want %q", tt.field, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestValidationErrors_ToMap tests the ToMap() method.
func TestValidationErrors_ToMap(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want map[string]interface{}
	}{
		{
			name: "nil_errors",
			errs: nil,
			want: nil,
		},
		{
			name: "empty_errors",
			errs: &ValidationErrors{Errors: []FieldError{}},
			want: nil,
		},
		{
			name: "has_errors",
			errs: &ValidationErrors{
				Errors: []FieldError{
					{Field: "email", Tag: "email", Message: "invalid"},
					{Field: "name", Tag: "required", Message: "required"},
				},
			},
			want: map[string]interface{}{
				"count": 2,
				"errors": []FieldError{
					{Field: "email", Tag: "email", Message: "invalid"},
					{Field: "name", Tag: "required", Message: "required"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.ToMap()
			if tt.want == nil {
				if got != nil {
					t.Errorf("ToMap() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("ToMap() = nil, want %v", tt.want)
				return
			}

			if got["count"] != tt.want["count"] {
				t.Errorf("ToMap()[count] = %v, want %v", got["count"], tt.want["count"])
			}
		})
	}
}

// TestValidationErrors_String tests the String() method.
func TestValidationErrors_String(t *testing.T) {
	errs := &ValidationErrors{
		Errors: []FieldError{
			{Field: "email", Tag: "email", Message: "email is invalid"},
		},
	}

	want := "validation failed: email is invalid"
	got := errs.String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestValidationErrors_Format tests the Format() method.
func TestValidationErrors_Format(t *testing.T) {
	errs := &ValidationErrors{
		Errors: []FieldError{
			{Field: "email", Tag: "email", Message: "invalid", Value: "notanemail"},
			{Field: "name", Tag: "required", Message: "required"},
		},
	}

	tests := []struct {
		name   string
		format string
		check  func(string) bool
	}{
		{
			name:   "simple_v",
			format: "%v",
			check: func(s string) bool {
				return s == "validation failed: invalid; required"
			},
		},
		{
			name:   "detailed_v",
			format: "%+v",
			check: func(s string) bool {
				return len(s) > 50 && // Detailed format is longer
					containsAll(s, "ValidationErrors", "email", "name", "tag=")
			},
		},
		{
			name:   "string_s",
			format: "%s",
			check: func(s string) bool {
				return s == "validation failed: invalid; required"
			},
		},
		{
			name:   "quoted_q",
			format: "%q",
			check: func(s string) bool {
				return s == `"validation failed: invalid; required"`
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(tt.format, errs)
			if !tt.check(got) {
				t.Errorf("Format(%q) = %q failed check", tt.format, got)
			}
		})
	}
}

// TestValidationErrors_Append tests the Append() method.
func TestValidationErrors_Append(t *testing.T) {
	errs := NewValidationErrors()

	if errs.Count() != 0 {
		t.Errorf("New errors should have count 0, got %d", errs.Count())
	}

	errs.Append("email", "email", "invalid email")
	if errs.Count() != 1 {
		t.Errorf("After Append, count should be 1, got %d", errs.Count())
	}

	if errs.FirstField() != "email" {
		t.Errorf("FirstField() = %q, want 'email'", errs.FirstField())
	}

	errs.Append("name", "required", "name required")
	if errs.Count() != 2 {
		t.Errorf("After second Append, count should be 2, got %d", errs.Count())
	}
}

// TestValidationErrors_AppendError tests the AppendError() method.
func TestValidationErrors_AppendError(t *testing.T) {
	errs := NewValidationErrors()

	fe := FieldError{
		Field:   "email",
		Tag:     "email",
		Message: "invalid",
		Value:   "test",
	}

	errs.AppendError(fe)
	if errs.Count() != 1 {
		t.Errorf("After AppendError, count should be 1, got %d", errs.Count())
	}

	if errs.Errors[0].Value != "test" {
		t.Errorf("AppendError did not preserve Value field")
	}
}

// TestNewValidationError tests the NewValidationError() function.
func TestNewValidationError(t *testing.T) {
	errs := NewValidationError("email", "email", "invalid email")

	if errs == nil {
		t.Fatal("NewValidationError returned nil")
	}

	if errs.Count() != 1 {
		t.Errorf("NewValidationError should create 1 error, got %d", errs.Count())
	}

	if errs.FirstField() != "email" {
		t.Errorf("Field = %q, want 'email'", errs.FirstField())
	}

	if errs.First() != "invalid email" {
		t.Errorf("Message = %q, want 'invalid email'", errs.First())
	}
}

// TestNewValidationErrors tests the NewValidationErrors() function.
func TestNewValidationErrors(t *testing.T) {
	errs := NewValidationErrors()

	if errs == nil {
		t.Fatal("NewValidationErrors returned nil")
	}

	if errs.Count() != 0 {
		t.Errorf("NewValidationErrors should create empty errors, got count %d", errs.Count())
	}

	if errs.HasErrors() {
		t.Error("NewValidationErrors should not have errors initially")
	}
}

// TestRegisterTranslator tests the RegisterTranslator() method.
func TestRegisterTranslator(t *testing.T) {
	v := New()

	// Get a translator to register
	enTrans := v.GetTranslator(LangEN)
	if enTrans == nil {
		t.Fatal("Could not get English translator")
	}

	// Register it under a new language code
	v.RegisterTranslator("test-lang", enTrans)

	// Verify it was registered
	testTrans := v.GetTranslator("test-lang")
	if testTrans == nil {
		t.Error("RegisterTranslator did not register the translator")
	}
}

// TestRegisterTranslations tests the RegisterTranslations() method.
func TestRegisterTranslations(_ *testing.T) {
	v := New()

	overrides := []TranslationOverride{
		{Tag: "custom1", Message: "Custom message 1 for {0}"},
		{Tag: "custom2", Message: "Custom message 2 for {0}"},
	}

	// Register for English
	v.RegisterTranslations(LangEN, overrides)

	// Register for non-existent language (should not panic)
	v.RegisterTranslations("non-existent", overrides)
}

// TestRegisterTranslation_Single tests the RegisterTranslation() method.
func TestRegisterTranslation_Single(_ *testing.T) {
	v := New()

	// Register a custom translation
	v.RegisterTranslation(LangEN, "custom_tag", "Custom message for {0}")

	// Register for non-existent language (should not panic)
	v.RegisterTranslation("non-existent", "custom_tag", "Message")
}

// containsAll checks if string contains all substrings.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

// contains checks if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
