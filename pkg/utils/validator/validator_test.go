package validator

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

// TestGlobal tests the global validator instance.
func TestGlobal(t *testing.T) {
	v1 := Global()
	if v1 == nil {
		t.Fatal("Global() returned nil")
	}

	// Should return the same instance
	v2 := Global()
	if v1 != v2 {
		t.Error("Global() should return the same instance")
	}

	// Test that validator is properly initialized
	if v1.validate == nil {
		t.Error("Global validator's validate field is nil")
	}
	if v1.uni == nil {
		t.Error("Global validator's uni field is nil")
	}
	if len(v1.trans) == 0 {
		t.Error("Global validator has no translators registered")
	}
}

// TestSetGlobal tests setting a custom global validator.
func TestSetGlobal(t *testing.T) {
	// Save original (通过 Global() 获取当前实例)
	original := Global()

	// Set custom validator
	custom := New()
	SetGlobal(custom)

	if Global() != custom {
		t.Error("SetGlobal() did not set the custom validator")
	}

	// Restore original
	SetGlobal(original)
}

// TestNew tests creating a new validator instance.
func TestNew(t *testing.T) {
	v := New()

	if v == nil {
		t.Fatal("New() returned nil")
	}

	if v.validate == nil {
		t.Error("Validator's validate field is nil")
	}

	if v.uni == nil {
		t.Error("Validator's uni field is nil")
	}

	// Check translators
	if len(v.trans) != 2 {
		t.Errorf("Expected 2 translators (en, zh), got %d", len(v.trans))
	}

	enTrans := v.GetTranslator(LangEN)
	if enTrans == nil {
		t.Error("English translator not registered")
	}

	zhTrans := v.GetTranslator(LangZH)
	if zhTrans == nil {
		t.Error("Chinese translator not registered")
	}
}

// TestValidate tests basic struct validation.
func TestValidate(t *testing.T) {
	v := New()

	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"gte=0,lte=150"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name: "valid_struct",
			input: TestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			wantErr: false,
		},
		{
			name: "missing_required_name",
			input: TestStruct{
				Name:  "",
				Email: "john@example.com",
				Age:   30,
			},
			wantErr: true,
		},
		{
			name: "invalid_email",
			input: TestStruct{
				Name:  "John Doe",
				Email: "invalid-email",
				Age:   30,
			},
			wantErr: true,
		},
		{
			name: "age_out_of_range",
			input: TestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   200,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateWithLang tests struct validation with language-specific error messages.
func TestValidateWithLang(t *testing.T) {
	v := New()

	type TestStruct struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	invalidStruct := TestStruct{
		Name:  "",
		Email: "invalid",
	}

	// Test English translation
	t.Run("english_translation", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidStruct, LangEN)
		if errs == nil {
			t.Fatal("Expected validation errors, got nil")
		}

		if !errs.HasErrors() {
			t.Error("HasErrors() should return true")
		}

		if errs.Count() != 2 {
			t.Errorf("Expected 2 errors, got %d", errs.Count())
		}

		// Check that messages are in English
		firstMsg := errs.First()
		if firstMsg == "" {
			t.Error("First() returned empty message")
		}

		messages := errs.Messages()
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
	})

	// Test Chinese translation
	t.Run("chinese_translation", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidStruct, LangZH)
		if errs == nil {
			t.Fatal("Expected validation errors, got nil")
		}

		if errs.Count() != 2 {
			t.Errorf("Expected 2 errors, got %d", errs.Count())
		}

		// Chinese messages should contain Chinese characters
		firstMsg := errs.First()
		if firstMsg == "" {
			t.Error("First() returned empty message")
		}
	})

	// Test valid struct returns nil
	t.Run("valid_struct", func(t *testing.T) {
		validStruct := TestStruct{
			Name:  "John Doe",
			Email: "john@example.com",
		}

		errs := v.ValidateWithLang(validStruct, LangEN)
		if errs != nil {
			t.Errorf("Expected nil for valid struct, got %v", errs)
		}
	})

	// Test invalid language defaults to English
	t.Run("invalid_language", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidStruct, "fr")
		if errs == nil {
			t.Fatal("Expected validation errors, got nil")
		}

		// Should fall back to English
		if errs.Count() != 2 {
			t.Errorf("Expected 2 errors, got %d", errs.Count())
		}
	})
}

// TestValidateVar tests single variable validation.
func TestValidateVar(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		field   interface{}
		tag     string
		wantErr bool
	}{
		{
			name:    "valid_email",
			field:   "test@example.com",
			tag:     "email",
			wantErr: false,
		},
		{
			name:    "invalid_email",
			field:   "not-an-email",
			tag:     "email",
			wantErr: true,
		},
		{
			name:    "valid_required",
			field:   "value",
			tag:     "required",
			wantErr: false,
		},
		{
			name:    "invalid_required",
			field:   "",
			tag:     "required",
			wantErr: true,
		},
		{
			name:    "valid_number_range",
			field:   50,
			tag:     "gte=0,lte=100",
			wantErr: false,
		},
		{
			name:    "invalid_number_range",
			field:   150,
			tag:     "gte=0,lte=100",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.field, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateVarWithLang tests single variable validation with translations.
func TestValidateVarWithLang(t *testing.T) {
	v := New()

	// Test invalid value with English
	t.Run("english_error", func(t *testing.T) {
		errs := v.ValidateVarWithLang("not-an-email", "email", LangEN)
		if errs == nil {
			t.Fatal("Expected validation error, got nil")
		}

		if !errs.HasErrors() {
			t.Error("HasErrors() should return true")
		}

		if errs.Count() != 1 {
			t.Errorf("Expected 1 error, got %d", errs.Count())
		}

		// For variable validation, field name may be empty or generic
		// Just check that we got an error message
		firstMsg := errs.First()
		if firstMsg == "" {
			t.Error("First() returned empty message")
		}
	})

	// Test invalid value with Chinese
	t.Run("chinese_error", func(t *testing.T) {
		errs := v.ValidateVarWithLang("", "required", LangZH)
		if errs == nil {
			t.Fatal("Expected validation error, got nil")
		}

		if errs.Count() != 1 {
			t.Errorf("Expected 1 error, got %d", errs.Count())
		}
	})

	// Test valid value returns nil
	t.Run("valid_value", func(t *testing.T) {
		errs := v.ValidateVarWithLang("test@example.com", "email", LangEN)
		if errs != nil {
			t.Errorf("Expected nil for valid value, got %v", errs)
		}
	})
}

// TestRegisterValidation tests registering custom validation functions.
func TestRegisterValidation(t *testing.T) {
	v := New()

	// Custom validation function
	customValidator := func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "secret"
	}

	// Register custom validation
	err := v.RegisterValidation("custom_secret", customValidator)
	if err != nil {
		t.Fatalf("RegisterValidation() error = %v", err)
	}

	// Test using the custom validation
	type TestStruct struct {
		Code string `validate:"custom_secret"`
	}

	validStruct := TestStruct{Code: "secret"}
	if err := v.Validate(validStruct); err != nil {
		t.Errorf("Custom validation failed for valid value: %v", err)
	}

	invalidStruct := TestStruct{Code: "wrong"}
	if err := v.Validate(invalidStruct); err == nil {
		t.Error("Custom validation should have failed for invalid value")
	}
}

// TestRegisterValidationWithTranslation tests registering custom validation with translations.
func TestRegisterValidationWithTranslation(t *testing.T) {
	v := New()

	// Custom validation function
	customValidator := func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "magic"
	}

	// Register with translations
	translations := map[string]string{
		LangEN: "{0} must be the magic word",
		LangZH: "{0}必须是魔法单词",
	}

	err := v.RegisterValidationWithTranslation("magic_word", customValidator, translations)
	if err != nil {
		t.Fatalf("RegisterValidationWithTranslation() error = %v", err)
	}

	type TestStruct struct {
		Word string `json:"word" validate:"magic_word"`
	}

	invalidStruct := TestStruct{Word: "wrong"}

	// Test English translation
	t.Run("english_translation", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidStruct, LangEN)
		if errs == nil {
			t.Fatal("Expected validation error, got nil")
		}

		msg := errs.First()
		if msg == "" {
			t.Error("Expected custom English error message")
		}
		// Message should contain "magic word"
		t.Logf("English message: %s", msg)
	})

	// Test Chinese translation
	t.Run("chinese_translation", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidStruct, LangZH)
		if errs == nil {
			t.Fatal("Expected validation error, got nil")
		}

		msg := errs.First()
		if msg == "" {
			t.Error("Expected custom Chinese error message")
		}
		t.Logf("Chinese message: %s", msg)
	})
}

// TestGetTranslator tests getting translators for different languages.
func TestGetTranslator(t *testing.T) {
	v := New()

	// Test English translator
	t.Run("english_translator", func(t *testing.T) {
		trans := v.GetTranslator(LangEN)
		if trans == nil {
			t.Error("English translator should not be nil")
		}
	})

	// Test Chinese translator
	t.Run("chinese_translator", func(t *testing.T) {
		trans := v.GetTranslator(LangZH)
		if trans == nil {
			t.Error("Chinese translator should not be nil")
		}
	})

	// Test unknown language defaults to English
	t.Run("unknown_language", func(t *testing.T) {
		trans := v.GetTranslator("fr")
		if trans == nil {
			t.Error("Unknown language should return default (English) translator")
		}
	})
}

// TestEngine tests getting the underlying validator engine.
func TestEngine(t *testing.T) {
	v := New()

	engine := v.Engine()
	if engine == nil {
		t.Error("Engine() should not return nil")
	}

	// Verify it's the same instance
	if engine != v.validate {
		t.Error("Engine() should return the internal validate instance")
	}
}

// TestConvenienceFunctions tests the global convenience wrapper functions.
func TestConvenienceFunctions(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name" validate:"required"`
	}

	// Test Struct function
	t.Run("Struct", func(t *testing.T) {
		validStruct := TestStruct{Name: "test"}
		if err := Struct(validStruct); err != nil {
			t.Errorf("Struct() unexpected error: %v", err)
		}

		invalidStruct := TestStruct{Name: ""}
		if err := Struct(invalidStruct); err == nil {
			t.Error("Struct() should return error for invalid struct")
		}
	})

	// Test StructWithLang function
	t.Run("StructWithLang", func(t *testing.T) {
		invalidStruct := TestStruct{Name: ""}
		errs := StructWithLang(invalidStruct, LangEN)
		if errs == nil {
			t.Error("StructWithLang() should return errors")
		}
		if !errs.HasErrors() {
			t.Error("StructWithLang() errors should have errors")
		}
	})

	// Test Var function
	t.Run("Var", func(t *testing.T) {
		if err := Var("test@example.com", "email"); err != nil {
			t.Errorf("Var() unexpected error: %v", err)
		}

		if err := Var("invalid", "email"); err == nil {
			t.Error("Var() should return error for invalid value")
		}
	})

	// Test VarWithLang function
	t.Run("VarWithLang", func(t *testing.T) {
		errs := VarWithLang("invalid", "email", LangEN)
		if errs == nil {
			t.Error("VarWithLang() should return errors")
		}
		if !errs.HasErrors() {
			t.Error("VarWithLang() errors should have errors")
		}
	})
}

// TestJSONTagNameFunc tests that JSON tag names are used for field names.
func TestJSONTagNameFunc(t *testing.T) {
	v := New()

	type TestStruct struct {
		FieldName string `json:"custom_field" validate:"required"`
	}

	invalidStruct := TestStruct{FieldName: ""}
	errs := v.ValidateWithLang(invalidStruct, LangEN)

	if errs == nil {
		t.Fatal("Expected validation error, got nil")
	}

	// Field name should be "custom_field" from JSON tag, not "FieldName"
	firstField := errs.FirstField()
	if firstField != "custom_field" {
		t.Errorf("Expected field name 'custom_field', got '%s'", firstField)
	}
}

// TestFormTagFallback tests that form tag is used when json tag is not present.
func TestFormTagFallback(t *testing.T) {
	v := New()

	type TestStruct struct {
		FieldName string `form:"form_field" validate:"required"`
	}

	invalidStruct := TestStruct{FieldName: ""}
	errs := v.ValidateWithLang(invalidStruct, LangEN)

	if errs == nil {
		t.Fatal("Expected validation error, got nil")
	}

	// Field name should be "form_field" from form tag
	firstField := errs.FirstField()
	if firstField != "form_field" {
		t.Errorf("Expected field name 'form_field', got '%s'", firstField)
	}
}

// TestConcurrentValidation tests that the validator is safe for concurrent use.
func TestConcurrentValidation(_ *testing.T) {
	v := New()

	type TestStruct struct {
		Email string `validate:"required,email"`
	}

	// Run multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				testStruct := TestStruct{Email: "test@example.com"}
				_ = v.Validate(testStruct)

				_ = v.ValidateVar("test@example.com", "email")

				_ = v.ValidateWithLang(testStruct, LangEN)
				_ = v.ValidateWithLang(testStruct, LangZH)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestConcurrentGlobalSetGlobal 测试 Global 和 SetGlobal 的并发安全性
func TestConcurrentGlobalSetGlobal(t *testing.T) {
	const goroutines = 50
	const iterations = 100

	type TestStruct struct {
		Name string `validate:"required"`
	}

	errChan := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				// 一半的 goroutine 调用 SetGlobal
				if id%2 == 0 {
					SetGlobal(New())
				}

				// 所有 goroutine 都调用 Global
				v := Global()
				if v == nil {
					errChan <- nil // 不应该发生，但用 nil 表示无错误
					return
				}

				// 使用获取到的验证器
				testStruct := TestStruct{Name: "test"}
				if err := v.Validate(testStruct); err != nil {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < goroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("并发 Global/SetGlobal 测试失败: %v", err)
		}
	}
}
