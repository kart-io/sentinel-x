// Package textutil 提供 RAG 相关的文本处理工具函数。
package textutil

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CosineSimilarity 计算两个向量的余弦相似度。
// 返回值范围为 [-1, 1]，1 表示完全相同，-1 表示完全相反。
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// NormalizeCosineSimilarity 将余弦相似度归一化到 [0, 1] 范围。
func NormalizeCosineSimilarity(similarity float64) float64 {
	return (similarity + 1) / 2
}

// HashString 计算字符串的 MD5 哈希值。
func HashString(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// TruncateString 截断字符串到指定的最大 Unicode 字符数。
func TruncateString(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}

// SplitIntoChunks 将文本分割成重叠的块。
// chunkSize 是每个块的大小（Unicode 字符数），overlap 是块之间的重叠大小。
func SplitIntoChunks(text string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize - 1
	}

	runes := []rune(text)
	if len(runes) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	step := chunkSize - overlap

	for i := 0; i < len(runes); i += step {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)
		if end == len(runes) {
			break
		}
	}

	return chunks
}

// ParseJSONArray 从文本中提取并解析 JSON 数组。
// 如果解析失败，返回 nil 和错误。
func ParseJSONArray(s string) ([]string, error) {
	// 提取 JSON 数组部分
	re := regexp.MustCompile(`\[[\s\S]*\]`)
	match := re.FindString(s)
	if match == "" {
		return nil, fmt.Errorf("未找到 JSON 数组")
	}

	var result []string
	if err := json.Unmarshal([]byte(match), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SplitByLines 按行分割文本，移除列表标记和空行。
// 仅返回长度大于 minLen 的行。
func SplitByLines(s string, minLen int) []string {
	if minLen <= 0 {
		minLen = 5
	}

	var result []string
	lines := strings.Split(s, "\n")
	listMarkerRegex := regexp.MustCompile(`^[\d\.\-\*\)]+\s*`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 移除列表标记
		line = listMarkerRegex.ReplaceAllString(line, "")
		line = strings.Trim(line, `"'`)
		if line != "" && len(line) > minLen {
			result = append(result, line)
		}
	}
	return result
}

// ExtractMarkdownSections 从 Markdown 内容中提取章节。
// 返回章节标题和对应内容的映射。
func ExtractMarkdownSections(content string) map[string]string {
	sections := make(map[string]string)
	headerRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

	parts := headerRegex.Split(content, -1)
	headers := headerRegex.FindAllStringSubmatch(content, -1)

	currentSection := "Introduction"
	for i, part := range parts {
		if i > 0 && i-1 < len(headers) {
			currentSection = headers[i-1][2]
		}
		part = strings.TrimSpace(part)
		if len(part) > 0 {
			sections[currentSection] = part
		}
	}

	return sections
}

// ContainsInt 检查整数切片是否包含指定元素。
func ContainsInt(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// ContainsString 检查字符串切片是否包含指定元素。
func ContainsString(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
