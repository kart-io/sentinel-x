// Package json_benchmark provides comprehensive benchmarks comparing
// sonic vs encoding/json performance across different scenarios.
package json

import (
	"bytes"
	stdjson "encoding/json"
	"io"
	"testing"

	"github.com/bytedance/sonic"
)

// Realistic API response structures matching sentinel-x patterns

type APIResponse struct {
	Code      int         `json:"code"`
	HTTPCode  int         `json:"http_code,omitempty"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

type UserData struct {
	ID        int      `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Role      string   `json:"role"`
	Active    bool     `json:"active"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
	Metadata  Metadata `json:"metadata"`
}

type Metadata struct {
	LastLogin     int64    `json:"last_login"`
	LoginCount    int      `json:"login_count"`
	Permissions   []string `json:"permissions"`
	Department    string   `json:"department"`
	PreferredLang string   `json:"preferred_lang"`
}

type PageResponse struct {
	Code     int       `json:"code"`
	Message  string    `json:"message"`
	Data     *PageData `json:"data,omitempty"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

type PageData struct {
	List       []UserData `json:"list"`
	TotalPages int        `json:"total_pages"`
}

// Benchmark data generators

func getAPIResponse() *APIResponse {
	return &APIResponse{
		Code:      0,
		HTTPCode:  200,
		Message:   "success",
		RequestID: "req-12345678-abcd-1234-efgh-123456789abc",
		Timestamp: 1703001234567,
		Data: map[string]interface{}{
			"id":       12345,
			"name":     "Test User",
			"email":    "test@example.com",
			"status":   "active",
			"score":    95.5,
			"count":    42,
			"tags":     []string{"admin", "developer", "reviewer"},
			"settings": map[string]bool{"notifications": true, "theme_dark": false},
		},
	}
}

func getUserData() *UserData {
	return &UserData{
		ID:        12345,
		Username:  "johndoe",
		Email:     "john.doe@example.com",
		Role:      "admin",
		Active:    true,
		CreatedAt: 1703001234567,
		UpdatedAt: 1703001234567,
		Metadata: Metadata{
			LastLogin:     1703001234567,
			LoginCount:    150,
			Permissions:   []string{"read", "write", "delete", "admin", "audit"},
			Department:    "Engineering",
			PreferredLang: "en",
		},
	}
}

func getPageResponse() *PageResponse {
	users := make([]UserData, 20)
	for i := range users {
		users[i] = UserData{
			ID:        10000 + i,
			Username:  "user" + string(rune('A'+i)),
			Email:     "user" + string(rune('A'+i)) + "@example.com",
			Role:      "user",
			Active:    i%2 == 0,
			CreatedAt: 1703001234567 + int64(i*1000),
			UpdatedAt: 1703001234567 + int64(i*2000),
			Metadata: Metadata{
				LastLogin:     1703001234567 + int64(i*3000),
				LoginCount:    100 + i*10,
				Permissions:   []string{"read", "write"},
				Department:    "Engineering",
				PreferredLang: "en",
			},
		}
	}

	return &PageResponse{
		Code:     0,
		Message:  "success",
		Total:    200,
		Page:     1,
		PageSize: 20,
		Data: &PageData{
			List:       users,
			TotalPages: 10,
		},
	}
}

// ============================================================================
// Marshal Benchmarks - API Response
// ============================================================================

func BenchmarkAPIResponse_Sonic(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

func BenchmarkAPIResponse_Stdlib(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(data)
	}
}

func BenchmarkAPIResponse_SonicDirect(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(data)
	}
}

// ============================================================================
// Marshal Benchmarks - User Data
// ============================================================================

func BenchmarkUserData_Sonic(b *testing.B) {
	data := getUserData()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

func BenchmarkUserData_Stdlib(b *testing.B) {
	data := getUserData()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(data)
	}
}

// ============================================================================
// Marshal Benchmarks - Page Response (Large payload)
// ============================================================================

func BenchmarkPageResponse_Sonic(b *testing.B) {
	data := getPageResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

func BenchmarkPageResponse_Stdlib(b *testing.B) {
	data := getPageResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(data)
	}
}

// ============================================================================
// Unmarshal Benchmarks
// ============================================================================

func BenchmarkAPIResponseUnmarshal_Sonic(b *testing.B) {
	data := getAPIResponse()
	jsonBytes, _ := Marshal(data)
	var result APIResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkAPIResponseUnmarshal_Stdlib(b *testing.B) {
	data := getAPIResponse()
	jsonBytes, _ := stdjson.Marshal(data)
	var result APIResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stdjson.Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkPageResponseUnmarshal_Sonic(b *testing.B) {
	data := getPageResponse()
	jsonBytes, _ := Marshal(data)
	var result PageResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkPageResponseUnmarshal_Stdlib(b *testing.B) {
	data := getPageResponse()
	jsonBytes, _ := stdjson.Marshal(data)
	var result PageResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stdjson.Unmarshal(jsonBytes, &result)
	}
}

// ============================================================================
// Encoder/Decoder Benchmarks (Streaming)
// ============================================================================

func BenchmarkAPIResponseEncoder_Sonic(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		_ = encoder.Encode(data)
	}
}

func BenchmarkAPIResponseEncoder_Stdlib(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := stdjson.NewEncoder(&buf)
		_ = encoder.Encode(data)
	}
}

func BenchmarkAPIResponseDecoder_Sonic(b *testing.B) {
	data := getAPIResponse()
	jsonBytes, _ := Marshal(data)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result APIResponse
		decoder := NewDecoder(bytes.NewReader(jsonBytes))
		_ = decoder.Decode(&result)
	}
}

func BenchmarkAPIResponseDecoder_Stdlib(b *testing.B) {
	data := getAPIResponse()
	jsonBytes, _ := stdjson.Marshal(data)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result APIResponse
		decoder := stdjson.NewDecoder(bytes.NewReader(jsonBytes))
		_ = decoder.Decode(&result)
	}
}

// ============================================================================
// Round-trip Benchmarks (Marshal + Unmarshal)
// ============================================================================

func BenchmarkRoundTripAPIResponse_Sonic(b *testing.B) {
	data := getAPIResponse()
	var result APIResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := Marshal(data)
		_ = Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkRoundTripAPIResponse_Stdlib(b *testing.B) {
	data := getAPIResponse()
	var result APIResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := stdjson.Marshal(data)
		_ = stdjson.Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkRoundTripPageResponse_Sonic(b *testing.B) {
	data := getPageResponse()
	var result PageResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := Marshal(data)
		_ = Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkRoundTripPageResponse_Stdlib(b *testing.B) {
	data := getPageResponse()
	var result PageResponse
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := stdjson.Marshal(data)
		_ = stdjson.Unmarshal(jsonBytes, &result)
	}
}

// ============================================================================
// HTTP Response Simulation (Most realistic scenario)
// ============================================================================

func BenchmarkHTTPResponse_Sonic(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		_ = encoder.Encode(data)
		// Simulate writing to HTTP connection
		_, _ = io.Copy(io.Discard, &buf)
	}
}

func BenchmarkHTTPResponse_Stdlib(b *testing.B) {
	data := getAPIResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := stdjson.NewEncoder(&buf)
		_ = encoder.Encode(data)
		// Simulate writing to HTTP connection
		_, _ = io.Copy(io.Discard, &buf)
	}
}

func BenchmarkHTTPPageResponse_Sonic(b *testing.B) {
	data := getPageResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		_ = encoder.Encode(data)
		_, _ = io.Copy(io.Discard, &buf)
	}
}

func BenchmarkHTTPPageResponse_Stdlib(b *testing.B) {
	data := getPageResponse()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := stdjson.NewEncoder(&buf)
		_ = encoder.Encode(data)
		_, _ = io.Copy(io.Discard, &buf)
	}
}
