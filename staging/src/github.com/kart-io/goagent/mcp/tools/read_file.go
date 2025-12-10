package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	agentErrors "github.com/kart-io/goagent/errors"
	"github.com/kart-io/goagent/mcp/core"
)

// ReadFileTool 读取文件工具
type ReadFileTool struct {
	name         string
	description  string
	category     string
	schema       *core.ToolSchema
	requiresAuth bool
	isDangerous  bool
}

// NewReadFileTool 创建读取文件工具
func NewReadFileTool() *ReadFileTool {
	schema := &core.ToolSchema{
		Type: "object",
		Properties: map[string]core.PropertySchema{
			"path": {
				Type:        "string",
				Description: "文件路径",
			},
			"encoding": {
				Type:        "string",
				Description: "文件编码（utf-8/ascii）",
				Default:     "utf-8",
				Enum:        []interface{}{"utf-8", "ascii"},
			},
		},
		Required: []string{"path"},
	}

	return &ReadFileTool{
		name:         "read_file",
		description:  "读取文件内容",
		category:     "filesystem",
		schema:       schema,
		requiresAuth: false,
		isDangerous:  false,
	}
}

// Name 返回工具名称
func (t *ReadFileTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *ReadFileTool) Description() string {
	return t.description
}

// Category 返回工具类别
func (t *ReadFileTool) Category() string {
	return t.category
}

// Schema 返回工具的 JSON Schema
func (t *ReadFileTool) Schema() *core.ToolSchema {
	return t.schema
}

// RequiresAuth 返回是否需要认证
func (t *ReadFileTool) RequiresAuth() bool {
	return t.requiresAuth
}

// IsDangerous 返回是否是危险操作
func (t *ReadFileTool) IsDangerous() bool {
	return t.isDangerous
}

// Execute 执行工具
func (t *ReadFileTool) Execute(ctx context.Context, input map[string]interface{}) (*core.ToolResult, error) {
	startTime := time.Now()

	path, ok := input["path"].(string)
	if !ok {
		return &core.ToolResult{
				Success:   false,
				Error:     "path must be a string",
				ErrorCode: "INVALID_INPUT",
				Duration:  time.Since(startTime),
				Timestamp: time.Now(),
			}, agentErrors.New(agentErrors.CodeInvalidInput, "invalid path type").
				WithComponent("read_file_tool").
				WithOperation("execute").
				WithContext("field", "path")
	}

	// 读取文件
	content, err := os.ReadFile(path)
	if err != nil {
		return &core.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to read file: %v", err),
			ErrorCode: "FILE_READ_ERROR",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	// 获取文件信息
	fileInfo, _ := os.Stat(path)

	result := &core.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"content": string(content),
			"size":    len(content),
			"path":    path,
		},
		Metadata: map[string]interface{}{
			"file_size":     fileInfo.Size(),
			"modified_time": fileInfo.ModTime(),
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// Validate 验证输入
func (t *ReadFileTool) Validate(input map[string]interface{}) error {
	path, ok := input["path"].(string)
	if !ok {
		return &core.ErrInvalidInput{Field: "path", Message: "must be a string"}
	}

	if path == "" {
		return &core.ErrInvalidInput{Field: "path", Message: "cannot be empty"}
	}

	return nil
}
