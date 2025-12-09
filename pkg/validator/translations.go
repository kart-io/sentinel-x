package validator

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// registerCustomTranslations registers translations for custom validation rules.
func (v *Validator) registerCustomTranslations() {
	// Register English translations
	enTrans := v.GetTranslator(LangEN)
	if enTrans != nil {
		v.registerEnglishTranslations(enTrans)
	}

	// Register Chinese translations
	zhTrans := v.GetTranslator(LangZH)
	if zhTrans != nil {
		v.registerChineseTranslations(zhTrans)
	}
}

// registerEnglishTranslations registers English translations for custom rules.
func (v *Validator) registerEnglishTranslations(trans ut.Translator) {
	translations := map[string]string{
		TagMobile:       "{0} must be a valid mobile phone number",
		TagIDCard:       "{0} must be a valid ID card number",
		TagUsername:     "{0} must start with a letter and contain only letters, numbers, and underscores (3-32 characters)",
		TagPassword:     "{0} must be at least 8 characters and contain at least one letter and one number",
		TagStrongPwd:    "{0} must be at least 8 characters and contain uppercase, lowercase, number, and special character",
		TagSafeString:   "{0} contains potentially unsafe content",
		TagNoWhitespace: "{0} must not contain whitespace characters",
		TagTrimmed:      "{0} must not have leading or trailing spaces",
		TagSlug:         "{0} must be a valid URL slug (lowercase letters, numbers, and hyphens)",
	}

	for tag, message := range translations {
		registerTranslation(v.validate, trans, tag, message)
	}
}

// registerChineseTranslations registers Chinese translations for custom rules.
func (v *Validator) registerChineseTranslations(trans ut.Translator) {
	translations := map[string]string{
		TagMobile:       "{0}必须是有效的手机号码",
		TagIDCard:       "{0}必须是有效的身份证号码",
		TagUsername:     "{0}必须以字母开头，只能包含字母、数字和下划线（3-32个字符）",
		TagPassword:     "{0}必须至少8个字符，且包含至少一个字母和一个数字",
		TagStrongPwd:    "{0}必须至少8个字符，且包含大写字母、小写字母、数字和特殊字符",
		TagSafeString:   "{0}包含潜在的不安全内容",
		TagNoWhitespace: "{0}不能包含空白字符",
		TagTrimmed:      "{0}不能有前导或尾随空格",
		TagSlug:         "{0}必须是有效的URL别名（小写字母、数字和连字符）",
	}

	for tag, message := range translations {
		registerTranslation(v.validate, trans, tag, message)
	}
}

// registerTranslation registers a single translation.
func registerTranslation(validate *validator.Validate, trans ut.Translator, tag, message string) {
	_ = validate.RegisterTranslation(tag, trans,
		func(ut ut.Translator) error {
			return ut.Add(tag, message, true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T(tag, fe.Field())
			return t
		},
	)
}

// TranslationOverride represents a translation override for a specific tag.
type TranslationOverride struct {
	Tag     string
	Message string
}

// RegisterTranslations registers multiple translation overrides for a language.
func (v *Validator) RegisterTranslations(lang string, overrides []TranslationOverride) {
	trans := v.GetTranslator(lang)
	if trans == nil {
		return
	}

	for _, override := range overrides {
		registerTranslation(v.validate, trans, override.Tag, override.Message)
	}
}

// RegisterTranslation registers a single translation override.
func (v *Validator) RegisterTranslation(lang, tag, message string) {
	trans := v.GetTranslator(lang)
	if trans == nil {
		return
	}

	registerTranslation(v.validate, trans, tag, message)
}
