package validator

import (
	"testing"
)

// TestMobileValidation tests Chinese mobile phone number validation.
func TestMobileValidation(t *testing.T) {
	tests := []validationTestCase{
		// Valid mobile numbers
		{"valid_13x", "13800138000", false},
		{"valid_14x", "14512345678", false},
		{"valid_15x", "15912345678", false},
		{"valid_16x", "16612345678", false},
		{"valid_17x", "17712345678", false},
		{"valid_18x", "18600000000", false},
		{"valid_19x", "19912345678", false},

		// Invalid mobile numbers
		{"invalid_12x", "12345678901", true},
		{"invalid_10x", "10800138000", true},
		{"invalid_too_short", "1380013800", true},
		{"invalid_too_long", "138001380000", true},
		{"invalid_start_with_23", "23800138000", true},
		{"invalid_start_with_0", "03800138000", true},
		{"invalid_letters", "1380013800a", true},
		{"invalid_special_chars", "138-0013-8000", true},
		{"invalid_spaces", "138 0013 8000", true},

		// Edge cases
		{"empty_string", "", false}, // Empty is valid (let 'required' handle it)
		{"only_digits_wrong_length", "123456789", true},
		{"valid_boundary_13000000000", "13000000000", false},
		{"valid_boundary_19999999999", "19999999999", false},
	}

	runValidationTests(t, TagMobile, tests)
}

// TestMobileValidationWithStruct tests mobile validation in struct context.
func TestMobileValidationWithStruct(t *testing.T) {
	v := New()

	type UserInfo struct {
		Phone string `json:"phone" validate:"mobile"`
	}

	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"valid_mobile", "13800138000", false},
		{"invalid_mobile", "12345678901", true},
		{"empty_optional", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := UserInfo{Phone: tt.phone}
			err := v.Validate(user)
			if (err != nil) != tt.wantErr {
				t.Errorf("Struct validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIDCardValidation tests Chinese ID card number validation.
func TestIDCardValidation(t *testing.T) {
	tests := []validationTestCase{
		// Valid ID card numbers (18-digit format)
		{"valid_18_digit_1980", "110101198001011234", false},
		{"valid_18_digit_1990", "110101199012311234", false},
		{"valid_18_digit_2000", "110101200001011234", false},
		{"valid_18_digit_2020", "110101202012311234", false},
		{"valid_with_X", "11010119800101123X", false},
		{"valid_with_lowercase_x", "11010119800101123x", false},

		// Invalid ID card numbers
		{"invalid_17_digits", "11010119800101123", true},
		{"invalid_19_digits", "1101011980010112345", true},
		{"invalid_start_with_0", "010101198001011234", true},
		{"invalid_year_17", "110101178001011234", true},
		{"invalid_year_21", "110101218001011234", true},
		{"invalid_month_00", "110101198000011234", true},
		{"invalid_month_13", "110101198013011234", true},
		{"invalid_day_00", "110101198001001234", true},
		{"invalid_day_32", "110101198001321234", true},
		{"invalid_letters", "11010119800101123A", true},
		{"invalid_special_chars", "110101-1980-0101-1234", true},

		// Edge cases
		{"empty_string", "", false}, // Empty is valid (let 'required' handle it)
		{"valid_leap_year_feb_29", "110101200002291234", false},
		{"valid_jan_31", "110101198001311234", false},
		{"valid_dec_31", "110101198012311234", false},
	}

	runValidationTests(t, TagIDCard, tests)
}

// TestUsernameValidation tests username format validation.
func TestUsernameValidation(t *testing.T) {
	tests := []validationTestCase{
		// Valid usernames
		{"valid_simple", "user123", false},
		{"valid_with_underscore", "Admin_test", false},
		{"valid_min_length", "abc", false},
		{"valid_max_length", "a123456789012345678901234567890b", false},
		{"valid_all_letters", "Username", false},
		{"valid_letter_number", "user1", false},
		{"valid_multiple_underscores", "user_name_test", false},

		// Invalid usernames
		{"invalid_start_with_number", "123user", true},
		{"invalid_start_with_underscore", "_admin", true},
		{"invalid_too_short", "ab", true},
		{"invalid_too_long", "a1234567890123456789012345678901234", true},
		{"invalid_special_char_dash", "user-name", true},
		{"invalid_special_char_at", "user@name", true},
		{"invalid_special_char_dot", "user.name", true},
		{"invalid_special_char_asterisk", "user*name", true},
		{"invalid_space", "user name", true},
		{"invalid_only_numbers", "123456", true},
		{"invalid_only_underscore", "___", true},

		// Edge cases
		{"empty_string", "", false}, // Empty is valid (let 'required' handle it)
		{"valid_exactly_3_chars", "a12", false},
		{"valid_exactly_32_chars", "a1234567890123456789012345678901", false},
	}

	runValidationTests(t, TagUsername, tests)
}

// TestPasswordValidation tests basic password validation.
func TestPasswordValidation(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid passwords
		{"valid_basic", "password123", false},
		{"valid_min_length", "pass1234", false},
		{"valid_with_special", "Pass@123", false},
		{"valid_long", "MyLongPassword123456", false},
		{"valid_letter_first", "a1234567", false},
		{"valid_number_first", "1abcdefg", false},

		// Invalid passwords
		{"invalid_too_short", "pass123", true},
		{"invalid_only_letters", "password", true},
		{"invalid_only_numbers", "12345678", true},
		{"invalid_only_special", "!@#$%^&*", true},
		{"invalid_no_letter", "12345678", true},
		{"invalid_no_number", "password", true},
		{"invalid_7_chars", "pass123", true},

		// Edge cases
		{"empty_string", "", false}, // Empty is valid (let 'required' handle it)
		{"valid_exactly_8_chars", "pass1234", false},
		{"valid_unicode_letter", "密码123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.value, TagPassword)
			if (err != nil) != tt.wantErr {
				t.Errorf("Password validation for '%s': error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestStrongPasswordValidation tests strong password validation.
func TestStrongPasswordValidation(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid strong passwords
		{"valid_all_requirements", "Pass@123", false},
		{"valid_complex", "MyP@ssw0rd!", false},
		{"valid_with_symbols", "Abc123!@#$", false},
		{"valid_min_length", "Aa1!bcde", false},

		// Invalid passwords
		{"invalid_no_uppercase", "pass@123", true},
		{"invalid_no_lowercase", "PASS@123", true},
		{"invalid_no_digit", "Pass@word", true},
		{"invalid_no_special", "Password123", true},
		{"invalid_too_short", "Aa1!", true},
		{"invalid_only_letters", "PasswordABC", true},
		{"invalid_only_numbers", "12345678", true},
		{"invalid_no_lower_and_upper", "password123!", true},
		{"invalid_no_upper_and_digit", "Password!!!", true},

		// Edge cases
		{"empty_string", "", false}, // Empty is valid (let 'required' handle it)
		{"valid_exactly_8_chars", "Aa1!bcde", false},
		{"valid_long_password", "MyVeryStr0ng!Password", false},
		{"valid_with_punctuation", "Test123.Pass", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.value, TagStrongPwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrongPassword validation for '%s': error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestSafeStringValidation tests safe string validation (no SQL injection, XSS patterns).
func TestSafeStringValidation(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Safe strings
		{"safe_simple", "Hello World", false},
		{"safe_with_numbers", "User123", false},
		{"safe_with_punctuation", "Hello, how are you?", false},
		{"safe_email", "user@example.com", false},
		{"safe_url", "https://example.com", false},

		// Dangerous SQL patterns
		{"dangerous_select", "SELECT * FROM users", true},
		{"dangerous_select_lowercase", "select * from users", true},
		{"dangerous_insert", "INSERT INTO users", true},
		{"dangerous_update", "UPDATE users SET", true},
		{"dangerous_delete", "DELETE FROM users", true},
		{"dangerous_drop", "DROP TABLE users", true},
		{"dangerous_union", "1 UNION SELECT", true},
		{"dangerous_or_1_equals_1", "admin' OR 1=1-- ", true},
		{"dangerous_or_quote", "' OR 'a'='a", true},
		{"dangerous_comment", "admin'-- ", true},
		{"dangerous_multiline_comment", "admin'/**/", true},

		// Dangerous XSS patterns
		{"dangerous_script_tag", "<script>alert('XSS')</script>", true},
		{"dangerous_script_tag_uppercase", "<SCRIPT>alert('XSS')</SCRIPT>", true},
		{"dangerous_script_close", "</script>", true},
		{"dangerous_javascript", "javascript:alert('XSS')", true},
		{"dangerous_javascript_uppercase", "JAVASCRIPT:void(0)", true},

		// Edge cases
		{"empty_string", "", false},
		{"safe_text_no_keywords", "This is a normal description", false},
		{"safe_word_selected", "You have selected this item", false},      // "selected" not "SELECT "
		{"safe_word_scripts", "The scripts folder contains files", false}, // "scripts" not "<script"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.value, TagSafeString)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeString validation for '%s': error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestNoWhitespaceValidation tests no whitespace validation.
func TestNoWhitespaceValidation(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid (no whitespace)
		{"valid_simple", "HelloWorld", false},
		{"valid_with_underscore", "Hello_World", false},
		{"valid_with_dash", "Hello-World", false},
		{"valid_alphanumeric", "User123", false},
		{"valid_special_chars", "user@example.com", false},

		// Invalid (contains whitespace)
		{"invalid_space", "Hello World", true},
		{"invalid_leading_space", " HelloWorld", true},
		{"invalid_trailing_space", "HelloWorld ", true},
		{"invalid_multiple_spaces", "Hello  World", true},
		{"invalid_tab", "Hello\tWorld", true},
		{"invalid_newline", "Hello\nWorld", true},
		{"invalid_carriage_return", "Hello\rWorld", true},
		{"invalid_mixed_whitespace", "Hello \t\n World", true},

		// Edge cases
		{"empty_string", "", false},
		{"valid_no_space", "NoSpaceHere", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.value, TagNoWhitespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("NoWhitespace validation for '%s': error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestTrimmedValidation tests trimmed string validation.
func TestTrimmedValidation(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid (trimmed)
		{"valid_simple", "Hello World", false},
		{"valid_no_spaces", "HelloWorld", false},
		{"valid_internal_spaces", "Hello World Test", false},
		{"valid_single_char", "a", false},

		// Invalid (not trimmed)
		{"invalid_leading_space", " HelloWorld", true},
		{"invalid_trailing_space", "HelloWorld ", true},
		{"invalid_both_spaces", " HelloWorld ", true},
		{"invalid_leading_tab", "\tHelloWorld", true},
		{"invalid_trailing_tab", "HelloWorld\t", true},
		{"invalid_leading_newline", "\nHelloWorld", true},
		{"invalid_trailing_newline", "HelloWorld\n", true},
		{"invalid_multiple_leading", "  HelloWorld", true},
		{"invalid_multiple_trailing", "HelloWorld  ", true},

		// Edge cases
		{"empty_string", "", false},
		{"valid_internal_only", "Hello  World", false}, // Internal spaces are OK
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.value, TagTrimmed)
			if (err != nil) != tt.wantErr {
				t.Errorf("Trimmed validation for '%s': error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestSlugValidation tests URL slug validation.
func TestSlugValidation(t *testing.T) {
	tests := []validationTestCase{
		// Valid slugs
		{"valid_simple", "hello-world", false},
		{"valid_single_word", "hello", false},
		{"valid_with_numbers", "test123", false},
		{"valid_multiple_hyphens", "this-is-a-test", false},
		{"valid_numbers_only", "123", false},
		{"valid_alphanumeric", "abc123def", false},
		{"valid_with_hyphen_numbers", "test-123-slug", false},

		// Invalid slugs
		{"invalid_uppercase", "Hello-World", true},
		{"invalid_uppercase_single", "Hello", true},
		{"invalid_start_with_hyphen", "-hello", true},
		{"invalid_end_with_hyphen", "hello-", true},
		{"invalid_double_hyphen", "hello--world", true},
		{"invalid_underscore", "hello_world", true},
		{"invalid_space", "hello world", true},
		{"invalid_special_char_at", "hello@world", true},
		{"invalid_special_char_dot", "hello.world", true},
		{"invalid_special_char_slash", "hello/world", true},

		// Edge cases
		{"empty_string", "", false},
		{"valid_single_char", "a", false},
		{"valid_long_slug", "this-is-a-very-long-slug-with-many-words", false},
	}

	runValidationTests(t, TagSlug, tests)
}

// TestCustomRulesWithTranslations tests that custom rules have proper translations.
func TestCustomRulesWithTranslations(t *testing.T) {
	v := New()

	type TestStruct struct {
		Mobile   string `json:"mobile" validate:"mobile"`
		IDCard   string `json:"id_card" validate:"idcard"`
		Username string `json:"username" validate:"username"`
		Password string `json:"password" validate:"password"`
	}

	invalidData := TestStruct{
		Mobile:   "invalid",
		IDCard:   "invalid",
		Username: "123invalid",
		Password: "short",
	}

	// Test English translations
	t.Run("english_translations", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidData, LangEN)
		if errs == nil {
			t.Fatal("Expected validation errors, got nil")
		}

		if errs.Count() != 4 {
			t.Errorf("Expected 4 errors, got %d", errs.Count())
		}

		// Check that all messages are non-empty
		for _, err := range errs.Errors {
			if err.Message == "" {
				t.Errorf("Empty message for field %s, tag %s", err.Field, err.Tag)
			}
			t.Logf("EN - %s: %s", err.Field, err.Message)
		}
	})

	// Test Chinese translations
	t.Run("chinese_translations", func(t *testing.T) {
		errs := v.ValidateWithLang(invalidData, LangZH)
		if errs == nil {
			t.Fatal("Expected validation errors, got nil")
		}

		if errs.Count() != 4 {
			t.Errorf("Expected 4 errors, got %d", errs.Count())
		}

		// Check that all messages are non-empty
		for _, err := range errs.Errors {
			if err.Message == "" {
				t.Errorf("Empty message for field %s, tag %s", err.Field, err.Tag)
			}
			t.Logf("ZH - %s: %s", err.Field, err.Message)
		}
	})
}

// TestCombinedValidationRules tests combining multiple validation rules.
func TestCombinedValidationRules(t *testing.T) {
	v := New()

	type UserRegistration struct {
		Username string `json:"username" validate:"required,username"`
		Password string `json:"password" validate:"required,strongpwd"`
		Mobile   string `json:"mobile" validate:"required,mobile"`
		Email    string `json:"email" validate:"required,email"`
	}

	tests := []struct {
		name    string
		user    UserRegistration
		wantErr bool
		errCnt  int
	}{
		{
			name: "all_valid",
			user: UserRegistration{
				Username: "validUser123",
				Password: "StrongP@ss1",
				Mobile:   "13800138000",
				Email:    "user@example.com",
			},
			wantErr: false,
			errCnt:  0,
		},
		{
			name: "all_invalid",
			user: UserRegistration{
				Username: "123invalid",
				Password: "weak",
				Mobile:   "12345",
				Email:    "invalid-email",
			},
			wantErr: true,
			errCnt:  4,
		},
		{
			name: "missing_required",
			user: UserRegistration{
				Username: "",
				Password: "",
				Mobile:   "",
				Email:    "",
			},
			wantErr: true,
			errCnt:  4,
		},
		{
			name: "valid_username_invalid_others",
			user: UserRegistration{
				Username: "validUser",
				Password: "weak",
				Mobile:   "invalid",
				Email:    "invalid",
			},
			wantErr: true,
			errCnt:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := v.ValidateWithLang(tt.user, LangEN)
			hasErr := errs != nil && errs.HasErrors()

			if hasErr != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if hasErr && errs.Count() != tt.errCnt {
				t.Errorf("Expected %d errors, got %d", tt.errCnt, errs.Count())
				for _, err := range errs.Errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
			}
		})
	}
}
