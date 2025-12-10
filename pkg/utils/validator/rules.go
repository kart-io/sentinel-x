package validator

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// Custom validation tags
const (
	TagMobile       = "mobile"       // Chinese mobile phone number
	TagIDCard       = "idcard"       // Chinese ID card number
	TagUsername     = "username"     // Username (alphanumeric, underscore, 3-32 chars)
	TagPassword     = "password"     // Password (min 8 chars, at least 1 letter and 1 number)
	TagStrongPwd    = "strongpwd"    // Strong password (min 8, upper+lower+digit+special)
	TagSafeString   = "safestring"   // Safe string (no SQL injection, XSS patterns)
	TagNoWhitespace = "nowhitespace" // No whitespace characters
	TagTrimmed      = "trimmed"      // String should be trimmed (no leading/trailing spaces)
	TagSlug         = "slug"         // URL slug (lowercase alphanumeric and hyphens)
)

var (
	// Regex patterns
	mobileRegex   = regexp.MustCompile(`^1[3-9]\d{9}$`)
	idCardRegex   = regexp.MustCompile(`^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,31}$`)
	slugRegex     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	// Dangerous patterns for safe string validation
	dangerousPatterns = []string{
		"<script", "</script>", "javascript:",
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "DROP ",
		"UNION ", "OR 1=1", "' OR '", "-- ", "/*", "*/",
	}
)

// registerCustomRules registers all custom validation rules.
func (v *Validator) registerCustomRules() {
	// Mobile phone number (Chinese)
	_ = v.validate.RegisterValidation(TagMobile, validateMobile)

	// ID card number (Chinese)
	_ = v.validate.RegisterValidation(TagIDCard, validateIDCard)

	// Username
	_ = v.validate.RegisterValidation(TagUsername, validateUsername)

	// Password (basic)
	_ = v.validate.RegisterValidation(TagPassword, validatePassword)

	// Strong password
	_ = v.validate.RegisterValidation(TagStrongPwd, validateStrongPassword)

	// Safe string
	_ = v.validate.RegisterValidation(TagSafeString, validateSafeString)

	// No whitespace
	_ = v.validate.RegisterValidation(TagNoWhitespace, validateNoWhitespace)

	// Trimmed string
	_ = v.validate.RegisterValidation(TagTrimmed, validateTrimmed)

	// URL slug
	_ = v.validate.RegisterValidation(TagSlug, validateSlug)
}

// validateMobile validates Chinese mobile phone numbers.
func validateMobile(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Let 'required' handle empty values
	}
	return mobileRegex.MatchString(value)
}

// validateIDCard validates Chinese ID card numbers.
func validateIDCard(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	return idCardRegex.MatchString(value)
}

// validateUsername validates username format.
// Must start with a letter, contain only alphanumeric and underscore, 3-32 chars.
func validateUsername(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	return usernameRegex.MatchString(value)
}

// validatePassword validates basic password requirements.
// At least 8 characters, containing at least 1 letter and 1 number.
func validatePassword(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	if len(value) < 8 {
		return false
	}

	hasLetter := false
	hasNumber := false

	for _, char := range value {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsDigit(char) {
			hasNumber = true
		}
		if hasLetter && hasNumber {
			return true
		}
	}

	return hasLetter && hasNumber
}

// validateStrongPassword validates strong password requirements.
// At least 8 characters, containing uppercase, lowercase, digit, and special character.
func validateStrongPassword(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	if len(value) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range value {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// validateSafeString checks for potentially dangerous patterns.
func validateSafeString(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	upperValue := strings.ToUpper(value)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(upperValue, strings.ToUpper(pattern)) {
			return false
		}
	}

	return true
}

// validateNoWhitespace validates that string contains no whitespace.
func validateNoWhitespace(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	for _, char := range value {
		if unicode.IsSpace(char) {
			return false
		}
	}

	return true
}

// validateTrimmed validates that string has no leading/trailing whitespace.
func validateTrimmed(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	return value == strings.TrimSpace(value)
}

// validateSlug validates URL slug format.
func validateSlug(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	return slugRegex.MatchString(value)
}
