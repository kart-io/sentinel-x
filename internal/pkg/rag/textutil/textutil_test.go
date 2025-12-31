package textutil_test

import (
	"testing"

	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/stretchr/testify/assert"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "相同向量",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "正交向量",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "相反向量",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{-1.0, 0.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "空向量",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "长度不匹配",
			a:        []float32{1.0, 2.0},
			b:        []float32{1.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutil.CosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestNormalizeCosineSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		similarity float64
		expected   float64
	}{
		{"最大相似度", 1.0, 1.0},
		{"最小相似度", -1.0, 0.0},
		{"中等相似度", 0.0, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutil.NormalizeCosineSimilarity(tt.similarity)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestHashString(t *testing.T) {
	// 相同输入应产生相同输出
	hash1 := textutil.HashString("test")
	hash2 := textutil.HashString("test")
	assert.Equal(t, hash1, hash2)

	// 不同输入应产生不同输出
	hash3 := textutil.HashString("different")
	assert.NotEqual(t, hash1, hash3)

	// 哈希应为32字符的十六进制字符串
	assert.Len(t, hash1, 32)
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "短于限制",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "等于限制",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "超过限制",
			input:    "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "中文字符",
			input:    "你好世界",
			maxLen:   2,
			expected: "你好",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutil.TruncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitIntoChunks(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		chunkSize int
		overlap   int
		expected  int // 期望的块数
	}{
		{
			name:      "短文本无需分割",
			text:      "hello",
			chunkSize: 10,
			overlap:   2,
			expected:  1,
		},
		{
			name:      "正常分割",
			text:      "hello world test",
			chunkSize: 5,
			overlap:   2,
			expected:  5,
		},
		{
			name:      "无重叠分割",
			text:      "abcdefghij",
			chunkSize: 5,
			overlap:   0,
			expected:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := textutil.SplitIntoChunks(tt.text, tt.chunkSize, tt.overlap)
			assert.Len(t, chunks, tt.expected)
		})
	}
}

func TestParseJSONArray(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []string
		shouldError bool
	}{
		{
			name:        "有效 JSON 数组",
			input:       `["a", "b", "c"]`,
			expected:    []string{"a", "b", "c"},
			shouldError: false,
		},
		{
			name:        "带前缀的 JSON",
			input:       `Here are the items: ["item1", "item2"]`,
			expected:    []string{"item1", "item2"},
			shouldError: false,
		},
		{
			name:        "无效输入",
			input:       "no json here",
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := textutil.ParseJSONArray(tt.input)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSplitByLines(t *testing.T) {
	input := `1. First item
2. Second item
- Third item
Short
This is a longer line that should be included`

	result := textutil.SplitByLines(input, 5)

	// 应该过滤掉 "Short"（长度不足）
	assert.NotContains(t, result, "Short")
	// 应该包含较长的行
	assert.Contains(t, result, "First item")
}

func TestContainsInt(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	assert.True(t, textutil.ContainsInt(slice, 3))
	assert.False(t, textutil.ContainsInt(slice, 6))
	assert.False(t, textutil.ContainsInt(nil, 1))
}

func TestContainsString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(t, textutil.ContainsString(slice, "banana"))
	assert.False(t, textutil.ContainsString(slice, "grape"))
	assert.False(t, textutil.ContainsString(nil, "apple"))
}
