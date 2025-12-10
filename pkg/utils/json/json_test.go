package json

import (
	"bytes"
	stdjson "encoding/json"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
)

// Test data structures
type SimpleStruct struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type ResponseStruct struct {
	Code     int                    `json:"code"`
	Message  string                 `json:"message"`
	Data     map[string]interface{} `json:"data,omitempty"`
	List     []string               `json:"list,omitempty"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

type ComplexStruct struct {
	ID        int                    `json:"id"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	Active    bool                   `json:"active"`
	Score     float64                `json:"score"`
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
	Nested    *SimpleStruct          `json:"nested,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "simple struct",
			data: SimpleStruct{
				ID:      1,
				Name:    "test",
				Message: "hello world",
			},
		},
		{
			name: "map with mixed types",
			data: map[string]interface{}{
				"code":    0,
				"message": "success",
				"data": map[string]interface{}{
					"id":   123,
					"name": "test",
					"tags": []string{"a", "b", "c"},
				},
			},
		},
		{
			name: "response struct",
			data: ResponseStruct{
				Code:    0,
				Message: "success",
				Data: map[string]interface{}{
					"id":    123,
					"name":  "test",
					"count": 42,
				},
				List: []string{"item1", "item2", "item3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.data)
			if err != nil {
				t.Errorf("Marshal() error = %v", err)
				return
			}

			// Verify it's valid JSON by unmarshaling with standard library
			var result interface{}
			if err := stdjson.Unmarshal(got, &result); err != nil {
				t.Errorf("Marshal() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		target  interface{}
		wantErr bool
	}{
		{
			name:   "simple struct",
			json:   `{"id":1,"name":"test","message":"hello"}`,
			target: &SimpleStruct{},
		},
		{
			name:   "response struct",
			json:   `{"code":0,"message":"success","data":{"id":123}}`,
			target: &ResponseStruct{},
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			target:  &SimpleStruct{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.json), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncoder(t *testing.T) {
	data := SimpleStruct{
		ID:      1,
		Name:    "test",
		Message: "hello",
	}

	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		t.Errorf("Encoder.Encode() error = %v", err)
	}

	// Verify output is valid JSON
	var result SimpleStruct
	if err := stdjson.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Encoder produced invalid JSON: %v", err)
	}

	if result.ID != data.ID || result.Name != data.Name {
		t.Errorf("Encoder output mismatch: got %+v, want %+v", result, data)
	}
}

func TestDecoder(t *testing.T) {
	json := `{"id":1,"name":"test","message":"hello"}`
	reader := strings.NewReader(json)

	decoder := NewDecoder(reader)
	var result SimpleStruct
	if err := decoder.Decode(&result); err != nil {
		t.Errorf("Decoder.Decode() error = %v", err)
	}

	if result.ID != 1 || result.Name != "test" {
		t.Errorf("Decoder output mismatch: got %+v", result)
	}
}

func TestConfigFastestMode(t *testing.T) {
	// Save original state
	origMarshal := Marshal
	defer func() { Marshal = origMarshal }()

	ConfigFastestMode()

	// Test that it still works
	data := SimpleStruct{ID: 1, Name: "test", Message: "hello"}
	_, err := Marshal(data)
	if err != nil {
		t.Errorf("Marshal() after ConfigFastestMode() error = %v", err)
	}
}

func TestConfigStandardMode(t *testing.T) {
	// Save original state
	origMarshal := Marshal
	defer func() { Marshal = origMarshal }()

	ConfigStandardMode()

	// Test that it still works
	data := SimpleStruct{ID: 1, Name: "test", Message: "hello"}
	_, err := Marshal(data)
	if err != nil {
		t.Errorf("Marshal() after ConfigStandardMode() error = %v", err)
	}
}

func TestIsUsingSonic(t *testing.T) {
	result := IsUsingSonic()
	// Just verify it returns a boolean without error
	t.Logf("Using sonic: %v (arch: %s)", result, "amd64/arm64 expected")
}

// ============================================================================
// Benchmarks
// ============================================================================

// getTestData returns a realistic API response structure
func getTestData() interface{} {
	return ResponseStruct{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"id":         12345,
			"name":       "John Doe",
			"email":      "john.doe@example.com",
			"age":        30,
			"active":     true,
			"score":      98.5,
			"role":       "admin",
			"department": "Engineering",
		},
		List: []string{
			"permission1",
			"permission2",
			"permission3",
			"permission4",
			"permission5",
		},
		Metadata: map[string]string{
			"version":     "1.0.0",
			"api_version": "v1",
			"region":      "us-west-2",
		},
	}
}

func getComplexTestData() interface{} {
	return ComplexStruct{
		ID:     12345,
		Name:   "John Doe",
		Email:  "john.doe@example.com",
		Active: true,
		Score:  98.5,
		Tags:   []string{"go", "performance", "json", "optimization"},
		Metadata: map[string]interface{}{
			"version":     "1.0.0",
			"environment": "production",
			"region":      "us-west-2",
			"count":       42,
			"ratio":       3.14159,
		},
		Nested: &SimpleStruct{
			ID:      999,
			Name:    "nested",
			Message: "nested message",
		},
		Timestamp: 1703001234567,
	}
}

// BenchmarkMarshal compares our wrapper against standard library
func BenchmarkMarshal(b *testing.B) {
	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

func BenchmarkMarshalStdlib(b *testing.B) {
	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(data)
	}
}

func BenchmarkMarshalSonic(b *testing.B) {
	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(data)
	}
}

// BenchmarkMarshalComplex tests with complex nested structures
func BenchmarkMarshalComplex(b *testing.B) {
	data := getComplexTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

func BenchmarkMarshalComplexStdlib(b *testing.B) {
	data := getComplexTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(data)
	}
}

func BenchmarkMarshalComplexSonic(b *testing.B) {
	data := getComplexTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(data)
	}
}

// BenchmarkUnmarshal tests deserialization performance
func BenchmarkUnmarshal(b *testing.B) {
	data := getTestData()
	jsonBytes, _ := Marshal(data)
	var result ResponseStruct
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkUnmarshalStdlib(b *testing.B) {
	data := getTestData()
	jsonBytes, _ := stdjson.Marshal(data)
	var result ResponseStruct
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stdjson.Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkUnmarshalSonic(b *testing.B) {
	data := getTestData()
	jsonBytes, _ := sonic.Marshal(data)
	var result ResponseStruct
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sonic.Unmarshal(jsonBytes, &result)
	}
}

// BenchmarkEncoder tests streaming encoding performance
func BenchmarkEncoder(b *testing.B) {
	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		_ = encoder.Encode(data)
	}
}

func BenchmarkEncoderStdlib(b *testing.B) {
	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := stdjson.NewEncoder(&buf)
		_ = encoder.Encode(data)
	}
}

func BenchmarkEncoderSonic(b *testing.B) {
	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := sonic.ConfigDefault.NewEncoder(&buf)
		_ = encoder.Encode(data)
	}
}

// BenchmarkMarshalSmall tests with small payloads
func BenchmarkMarshalSmall(b *testing.B) {
	data := SimpleStruct{
		ID:      1,
		Name:    "test",
		Message: "hello",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

func BenchmarkMarshalSmallStdlib(b *testing.B) {
	data := SimpleStruct{
		ID:      1,
		Name:    "test",
		Message: "hello",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(data)
	}
}

// BenchmarkMarshalFastestMode tests sonic's fastest configuration
func BenchmarkMarshalFastestMode(b *testing.B) {
	if !IsUsingSonic() {
		b.Skip("Sonic not available on this architecture")
	}

	// Switch to fastest mode
	origMarshal := Marshal
	defer func() { Marshal = origMarshal }()
	ConfigFastestMode()

	data := getTestData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(data)
	}
}

// BenchmarkRoundtrip tests full marshal->unmarshal cycle
func BenchmarkRoundtrip(b *testing.B) {
	data := getTestData()
	var result ResponseStruct
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := Marshal(data)
		_ = Unmarshal(jsonBytes, &result)
	}
}

func BenchmarkRoundtripStdlib(b *testing.B) {
	data := getTestData()
	var result ResponseStruct
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := stdjson.Marshal(data)
		_ = stdjson.Unmarshal(jsonBytes, &result)
	}
}
