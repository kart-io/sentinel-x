// Package validator provides a unified validation component based on go-playground/validator.
// It offers global validator initialization, custom validation rules, i18n error messages,
// and deep integration with HTTP/gRPC frameworks.
package validator

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

// Language constants for i18n support.
const (
	LangEN = "en"
	LangZH = "zh"
)

// Validator wraps go-playground/validator with additional features.
type Validator struct {
	validate *validator.Validate
	uni      *ut.UniversalTranslator
	trans    map[string]ut.Translator
	mu       sync.RWMutex
}

var (
	globalValidator *Validator
	once            sync.Once
)

// Global returns the global validator instance.
// It initializes the validator on first call with default settings.
func Global() *Validator {
	once.Do(func() {
		globalValidator = New()
	})
	return globalValidator
}

// SetGlobal sets the global validator instance.
// This should be called during application initialization if custom configuration is needed.
func SetGlobal(v *Validator) {
	globalValidator = v
}

// New creates a new Validator instance with default configuration.
func New() *Validator {
	v := &Validator{
		validate: validator.New(),
		trans:    make(map[string]ut.Translator),
	}

	// Use JSON tag names for error field names
	v.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			name = strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	// Initialize universal translator
	enLocale := en.New()
	zhLocale := zh.New()
	v.uni = ut.New(enLocale, enLocale, zhLocale)

	// Register English translations
	enTrans, _ := v.uni.GetTranslator(LangEN)
	_ = en_translations.RegisterDefaultTranslations(v.validate, enTrans)
	v.trans[LangEN] = enTrans

	// Register Chinese translations
	zhTrans, _ := v.uni.GetTranslator(LangZH)
	_ = zh_translations.RegisterDefaultTranslations(v.validate, zhTrans)
	v.trans[LangZH] = zhTrans

	// Register custom validation rules
	v.registerCustomRules()

	// Register custom translations
	v.registerCustomTranslations()

	return v
}

// Validate validates a struct and returns validation errors.
func (v *Validator) Validate(s interface{}) error {
	return v.validate.Struct(s)
}

// ValidateWithLang validates a struct and returns translated validation errors.
func (v *Validator) ValidateWithLang(s interface{}, lang string) *ValidationErrors {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return &ValidationErrors{
			Errors: []FieldError{
				{
					Field:   "unknown",
					Tag:     "unknown",
					Message: err.Error(),
				},
			},
		}
	}

	trans := v.GetTranslator(lang)
	return v.translateErrors(validationErrors, trans)
}

// ValidateVar validates a single variable.
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// ValidateVarWithLang validates a single variable and returns translated error.
func (v *Validator) ValidateVarWithLang(field interface{}, tag string, lang string) *ValidationErrors {
	err := v.validate.Var(field, tag)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return &ValidationErrors{
			Errors: []FieldError{
				{
					Field:   "value",
					Tag:     tag,
					Message: err.Error(),
				},
			},
		}
	}

	trans := v.GetTranslator(lang)
	return v.translateErrors(validationErrors, trans)
}

// GetTranslator returns a translator for the specified language.
func (v *Validator) GetTranslator(lang string) ut.Translator {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if trans, ok := v.trans[lang]; ok {
		return trans
	}
	// Default to English
	return v.trans[LangEN]
}

// RegisterTranslator registers a custom translator for a language.
func (v *Validator) RegisterTranslator(lang string, trans ut.Translator) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.trans[lang] = trans
}

// RegisterValidation registers a custom validation function.
func (v *Validator) RegisterValidation(tag string, fn validator.Func, callValidationEvenIfNull ...bool) error {
	return v.validate.RegisterValidation(tag, fn, callValidationEvenIfNull...)
}

// RegisterValidationWithTranslation registers a custom validation with translation.
func (v *Validator) RegisterValidationWithTranslation(
	tag string,
	fn validator.Func,
	translations map[string]string,
) error {
	if err := v.validate.RegisterValidation(tag, fn); err != nil {
		return err
	}

	for lang, message := range translations {
		trans := v.GetTranslator(lang)
		if trans == nil {
			continue
		}

		_ = v.validate.RegisterTranslation(tag, trans,
			func(ut ut.Translator) error {
				return ut.Add(tag, message, true)
			},
			func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T(tag, fe.Field())
				return t
			},
		)
	}

	return nil
}

// Engine returns the underlying validator.Validate instance.
// Use this only when direct access is absolutely necessary.
func (v *Validator) Engine() *validator.Validate {
	return v.validate
}

// translateErrors translates validation errors to human-readable messages.
func (v *Validator) translateErrors(errs validator.ValidationErrors, trans ut.Translator) *ValidationErrors {
	result := &ValidationErrors{
		Errors: make([]FieldError, 0, len(errs)),
	}

	for _, err := range errs {
		fe := FieldError{
			Field:   err.Field(),
			Tag:     err.Tag(),
			Value:   err.Value(),
			Param:   err.Param(),
			Message: err.Translate(trans),
		}
		result.Errors = append(result.Errors, fe)
	}

	return result
}

// Struct validates a struct (convenience wrapper for global validator).
func Struct(s interface{}) error {
	return Global().Validate(s)
}

// StructWithLang validates a struct with language support (convenience wrapper).
func StructWithLang(s interface{}, lang string) *ValidationErrors {
	return Global().ValidateWithLang(s, lang)
}

// Var validates a single variable (convenience wrapper for global validator).
func Var(field interface{}, tag string) error {
	return Global().ValidateVar(field, tag)
}

// VarWithLang validates a single variable with language support (convenience wrapper).
func VarWithLang(field interface{}, tag string, lang string) *ValidationErrors {
	return Global().ValidateVarWithLang(field, tag, lang)
}
