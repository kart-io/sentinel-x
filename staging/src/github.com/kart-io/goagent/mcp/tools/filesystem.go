package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kart-io/goagent/mcp/core"
)

// ListDirectoryTool 列出目录工具
type ListDirectoryTool struct {
	name         string
	description  string
	category     string
	schema       *core.ToolSchema
	requiresAuth bool
	isDangerous  bool
}

// NewListDirectoryTool 创建列出目录工具
func NewListDirectoryTool() *ListDirectoryTool {
	schema := &core.ToolSchema{
		Type: "object",
		Properties: map[string]core.PropertySchema{
			"path": {
				Type:        "string",
				Description: "目录路径",
			},
			"recursive": {
				Type:        "boolean",
				Description: "是否递归列出子目录",
				Default:     false,
			},
			"include_hidden": {
				Type:        "boolean",
				Description: "是否包含隐藏文件",
				Default:     false,
			},
		},
		Required: []string{"path"},
	}

	return &ListDirectoryTool{
		name:         "list_directory",
		description:  "列出目录内容",
		category:     "filesystem",
		schema:       schema,
		requiresAuth: false,
		isDangerous:  false,
	}
}

// Name 返回工具名称
func (t *ListDirectoryTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *ListDirectoryTool) Description() string {
	return t.description
}

// Category 返回工具类别
func (t *ListDirectoryTool) Category() string {
	return t.category
}

// Schema 返回工具的 JSON Schema
func (t *ListDirectoryTool) Schema() *core.ToolSchema {
	return t.schema
}

// RequiresAuth 返回是否需要认证
func (t *ListDirectoryTool) RequiresAuth() bool {
	return t.requiresAuth
}

// IsDangerous 返回是否是危险操作
func (t *ListDirectoryTool) IsDangerous() bool {
	return t.isDangerous
}

// Execute 执行工具
func (t *ListDirectoryTool) Execute(ctx context.Context, input map[string]interface{}) (*core.ToolResult, error) {
	startTime := time.Now()

	path, _ := input["path"].(string)
	recursive, _ := input["recursive"].(bool)
	includeHidden, _ := input["include_hidden"].(bool)

	var files []map[string]interface{}
	var err error

	if recursive {
		files, err = t.listRecursive(path, includeHidden)
	} else {
		files, err = t.listFlat(path, includeHidden)
	}

	if err != nil {
		return &core.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to list directory: %v", err),
			ErrorCode: "DIR_LIST_ERROR",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	result := &core.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"path":  path,
			"files": files,
			"count": len(files),
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// listFlat 平面列出目录
func (t *ListDirectoryTool) listFlat(path string, includeHidden bool) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		name := entry.Name()

		// 跳过隐藏文件
		if !includeHidden && len(name) > 0 && name[0] == '.' {
			continue
		}

		info, _ := entry.Info()
		fileInfo := map[string]interface{}{
			"name":     name,
			"path":     filepath.Join(path, name),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"modified": info.ModTime(),
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// listRecursive 递归列出目录
func (t *ListDirectoryTool) listRecursive(path string, includeHidden bool) ([]map[string]interface{}, error) {
	files := make([]map[string]interface{}, 0)

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := filepath.Base(filePath)

		// 跳过隐藏文件和目录
		if !includeHidden && len(name) > 0 && name[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(path, filePath)
		fileInfo := map[string]interface{}{
			"name":     name,
			"path":     filePath,
			"rel_path": relPath,
			"is_dir":   info.IsDir(),
			"size":     info.Size(),
			"modified": info.ModTime(),
		}

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

// Validate 验证输入
func (t *ListDirectoryTool) Validate(input map[string]interface{}) error {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return &core.ErrInvalidInput{Field: "path", Message: "must be a non-empty string"}
	}

	return nil
}

// SearchFilesTool 搜索文件工具
type SearchFilesTool struct {
	name         string
	description  string
	category     string
	schema       *core.ToolSchema
	requiresAuth bool
	isDangerous  bool
}

// NewSearchFilesTool 创建搜索文件工具
func NewSearchFilesTool() *SearchFilesTool {
	schema := &core.ToolSchema{
		Type: "object",
		Properties: map[string]core.PropertySchema{
			"path": {
				Type:        "string",
				Description: "搜索路径",
			},
			"pattern": {
				Type:        "string",
				Description: "文件名模式（支持通配符 * 和 ?）",
			},
			"content": {
				Type:        "string",
				Description: "搜索文件内容（可选）",
			},
			"max_results": {
				Type:        "integer",
				Description: "最大结果数",
				Default:     100,
			},
		},
		Required: []string{"path", "pattern"},
	}

	return &SearchFilesTool{
		name:         "search_files",
		description:  "搜索文件",
		category:     "filesystem",
		schema:       schema,
		requiresAuth: false,
		isDangerous:  false,
	}
}

// Name 返回工具名称
func (t *SearchFilesTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *SearchFilesTool) Description() string {
	return t.description
}

// Category 返回工具类别
func (t *SearchFilesTool) Category() string {
	return t.category
}

// Schema 返回工具的 JSON Schema
func (t *SearchFilesTool) Schema() *core.ToolSchema {
	return t.schema
}

// RequiresAuth 返回是否需要认证
func (t *SearchFilesTool) RequiresAuth() bool {
	return t.requiresAuth
}

// IsDangerous 返回是否是危险操作
func (t *SearchFilesTool) IsDangerous() bool {
	return t.isDangerous
}

// Execute 执行工具
func (t *SearchFilesTool) Execute(ctx context.Context, input map[string]interface{}) (*core.ToolResult, error) {
	startTime := time.Now()

	searchPath, _ := input["path"].(string)
	pattern, _ := input["pattern"].(string)
	contentSearch, _ := input["content"].(string)
	maxResults := 100
	if mr, ok := input["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	matches := make([]map[string]interface{}, 0)
	count := 0

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		//nolint:nilerr // Intentionally skip errors to continue traversing
		if err != nil {
			return nil // 跳过错误
		}

		if count >= maxResults {
			return io.EOF // 达到最大结果数
		}

		// 匹配文件名
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if !matched {
			return nil
		}

		// 如果需要搜索内容
		if contentSearch != "" && !info.IsDir() {
			content, err := os.ReadFile(path)
			//nolint:nilerr // Skip files that cannot be read
			if err != nil {
				return nil
			}
			if !contains(string(content), contentSearch) {
				return nil
			}
		}

		match := map[string]interface{}{
			"path":     path,
			"name":     filepath.Base(path),
			"is_dir":   info.IsDir(),
			"size":     info.Size(),
			"modified": info.ModTime(),
		}

		matches = append(matches, match)
		count++

		return nil
	})

	if err != nil && !errors.Is(err, io.EOF) {
		return &core.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("search failed: %v", err),
			ErrorCode: "SEARCH_ERROR",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	result := &core.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"matches": matches,
			"count":   len(matches),
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// Validate 验证输入
func (t *SearchFilesTool) Validate(input map[string]interface{}) error {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return &core.ErrInvalidInput{Field: "path", Message: "must be a non-empty string"}
	}

	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return &core.ErrInvalidInput{Field: "pattern", Message: "must be a non-empty string"}
	}

	return nil
}

// contains 检查字符串是否包含子串
// 使用标准库实现，性能 O(n+m)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
