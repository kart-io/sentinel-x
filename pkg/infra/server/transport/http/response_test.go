package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// testFormContentType is the content type for form-urlencoded requests.
const testFormContentType = "application/x-www-form-urlencoded"

// testFormRequest creates a form-urlencoded POST request with the given form data.
func testFormRequest(t *testing.T, formData url.Values) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", testFormContentType)
	return req
}

// TestBindForm_TagSupport tests binding with different tag types.
func TestBindForm_TagSupport(t *testing.T) {
	tests := []struct {
		name     string
		formData map[string]string
		bindFunc func(*http.Request) (interface{}, error)
		validate func(t *testing.T, result interface{})
	}{
		{
			name: "form tags",
			formData: map[string]string{
				"username": "testuser",
				"password": "secret123",
				"age":      "25",
			},
			bindFunc: func(req *http.Request) (interface{}, error) {
				type FormRequest struct {
					Username string `form:"username"`
					Password string `form:"password"`
					Age      int    `form:"age"`
				}
				var target FormRequest
				return &target, bindForm(req, &target)
			},
			validate: func(t *testing.T, result interface{}) {
				r := result.(*struct {
					Username string `form:"username"`
					Password string `form:"password"`
					Age      int    `form:"age"`
				})
				if r.Username != "testuser" {
					t.Errorf("expected username 'testuser', got '%s'", r.Username)
				}
			},
		},
		{
			name: "json tags (protobuf scenario)",
			formData: map[string]string{
				"username": "protouser",
				"password": "protopass",
				"email":    "test@example.com",
			},
			bindFunc: func(req *http.Request) (interface{}, error) {
				type ProtoLikeRequest struct {
					Username string `json:"username,omitempty"`
					Password string `json:"password,omitempty"`
					Email    string `json:"email,omitempty"`
				}
				var target ProtoLikeRequest
				return &target, bindForm(req, &target)
			},
			validate: func(t *testing.T, result interface{}) {
				r := result.(*struct {
					Username string `json:"username,omitempty"`
					Password string `json:"password,omitempty"`
					Email    string `json:"email,omitempty"`
				})
				if r.Username != "protouser" {
					t.Errorf("expected username 'protouser', got '%s'", r.Username)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			for k, v := range tt.formData {
				form.Set(k, v)
			}
			req := testFormRequest(t, form)
			_, err := tt.bindFunc(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestBindForm_FormTagPriority tests that form tag takes priority over json tag.
func TestBindForm_FormTagPriority(t *testing.T) {
	type MixedRequest struct {
		Name string `form:"form_name" json:"json_name"`
	}

	form := url.Values{}
	form.Set("form_name", "from_form")
	form.Set("json_name", "from_json")

	req := testFormRequest(t, form)

	var target MixedRequest
	err := bindForm(req, &target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "from_form" {
		t.Errorf("form tag should take priority: expected 'from_form', got '%s'", target.Name)
	}
}

// TestBindForm_VariousTypes tests binding of various field types.
func TestBindForm_VariousTypes(t *testing.T) {
	type TypesRequest struct {
		String  string  `json:"string_field"`
		Int     int     `json:"int_field"`
		Int64   int64   `json:"int64_field"`
		Uint    uint    `json:"uint_field"`
		Float64 float64 `json:"float64_field"`
		Bool    bool    `json:"bool_field"`
	}

	form := url.Values{}
	form.Set("string_field", "hello")
	form.Set("int_field", "42")
	form.Set("int64_field", "9999999999")
	form.Set("uint_field", "100")
	form.Set("float64_field", "3.14")
	form.Set("bool_field", "true")

	req := testFormRequest(t, form)

	var target TypesRequest
	err := bindForm(req, &target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.String != "hello" {
		t.Errorf("expected string 'hello', got '%s'", target.String)
	}
	if target.Int != 42 {
		t.Errorf("expected int 42, got %d", target.Int)
	}
	if target.Int64 != 9999999999 {
		t.Errorf("expected int64 9999999999, got %d", target.Int64)
	}
	if target.Uint != 100 {
		t.Errorf("expected uint 100, got %d", target.Uint)
	}
	if target.Float64 < 3.13 || target.Float64 > 3.15 {
		t.Errorf("expected float64 ~3.14, got %f", target.Float64)
	}
	if !target.Bool {
		t.Error("expected bool true")
	}
}

// TestBindForm_BoolVariants tests different boolean value representations.
func TestBindForm_BoolVariants(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"off", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			type BoolRequest struct {
				Flag bool `json:"flag"`
			}
			form := url.Values{}
			form.Set("flag", tt.value)

			req := testFormRequest(t, form)
			var target BoolRequest
			if err := bindForm(req, &target); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if target.Flag != tt.expected {
				t.Errorf("value '%s': expected %v, got %v", tt.value, tt.expected, target.Flag)
			}
		})
	}
}

// TestBindForm_MissingFields tests that missing form fields don't cause errors.
func TestBindForm_MissingFields(t *testing.T) {
	type PartialRequest struct {
		Required string `json:"required"`
		Optional string `json:"optional"`
	}

	form := url.Values{}
	form.Set("required", "present")

	req := testFormRequest(t, form)

	var target PartialRequest
	err := bindForm(req, &target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Required != "present" {
		t.Errorf("expected required 'present', got '%s'", target.Required)
	}
	if target.Optional != "" {
		t.Errorf("expected optional to be empty, got '%s'", target.Optional)
	}
}

// TestBindForm_NonPointerTarget tests error when target is not a pointer.
func TestBindForm_NonPointerTarget(t *testing.T) {
	type Request struct {
		Field string `json:"field"`
	}

	form := url.Values{}
	form.Set("field", "value")

	req := testFormRequest(t, form)

	var target Request
	err := bindForm(req, target) // Note: not a pointer

	if err == nil {
		t.Error("expected error for non-pointer target")
	}
}

// TestBind_FormURLEncoded tests the main Bind method with form-urlencoded content.
func TestBind_FormURLEncoded(t *testing.T) {
	type RegisterRequest struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Email    string `json:"email,omitempty"`
	}

	form := url.Values{}
	form.Set("username", "newuser")
	form.Set("password", "newpass123")
	form.Set("email", "new@example.com")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", testFormContentType)

	rec := httptest.NewRecorder()
	ctx := NewRequestContext(req, rec)

	var target RegisterRequest
	err := ctx.Bind(&target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Username != "newuser" {
		t.Errorf("expected username 'newuser', got '%s'", target.Username)
	}
	if target.Password != "newpass123" {
		t.Errorf("expected password 'newpass123', got '%s'", target.Password)
	}
	if target.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got '%s'", target.Email)
	}
}

// TestBind_JSON tests that JSON binding still works correctly.
func TestBind_JSON(t *testing.T) {
	const testName = "test"

	type JSONRequest struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	body := `{"name":"` + testName + `","value":123}`

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := NewRequestContext(req, rec)

	var target JSONRequest
	err := ctx.Bind(&target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != testName {
		t.Errorf("expected name '%s', got '%s'", testName, target.Name)
	}
	if target.Value != 123 {
		t.Errorf("expected value 123, got %d", target.Value)
	}
}
