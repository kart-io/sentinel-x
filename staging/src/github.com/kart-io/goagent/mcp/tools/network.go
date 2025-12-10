package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/kart-io/goagent/mcp/core"
	"github.com/kart-io/goagent/utils/httpclient"
	"github.com/kart-io/goagent/utils/json"
)

// HTTPRequestTool HTTP 请求工具
type HTTPRequestTool struct {
	name         string
	description  string
	category     string
	schema       *core.ToolSchema
	requiresAuth bool
	isDangerous  bool
	client       *httpclient.Client
}

// NewHTTPRequestTool 创建 HTTP 请求工具
func NewHTTPRequestTool() *HTTPRequestTool {
	schema := &core.ToolSchema{
		Type: "object",
		Properties: map[string]core.PropertySchema{
			"url": {
				Type:        "string",
				Description: "请求URL",
				Format:      "uri",
			},
			"method": {
				Type:        "string",
				Description: "HTTP 方法",
				Default:     "GET",
				Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH"},
			},
			"headers": {
				Type:        "object",
				Description: "请求头",
			},
			"body": {
				Type:        "string",
				Description: "请求体（JSON字符串）",
			},
			"timeout": {
				Type:        "integer",
				Description: "超时时间（秒）",
				Default:     30,
			},
		},
		Required: []string{"url"},
	}

	return &HTTPRequestTool{
		name:         "http_request",
		description:  "发送 HTTP 请求",
		category:     "network",
		schema:       schema,
		requiresAuth: false,
		isDangerous:  false,
		client: httpclient.NewClient(&httpclient.Config{
			Timeout: 30 * time.Second,
		}),
	}
}

// Name 返回工具名称
func (t *HTTPRequestTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *HTTPRequestTool) Description() string {
	return t.description
}

// Category 返回工具类别
func (t *HTTPRequestTool) Category() string {
	return t.category
}

// Schema 返回工具的 JSON Schema
func (t *HTTPRequestTool) Schema() *core.ToolSchema {
	return t.schema
}

// RequiresAuth 返回是否需要认证
func (t *HTTPRequestTool) RequiresAuth() bool {
	return t.requiresAuth
}

// IsDangerous 返回是否是危险操作
func (t *HTTPRequestTool) IsDangerous() bool {
	return t.isDangerous
}

// Execute 执行工具
func (t *HTTPRequestTool) Execute(ctx context.Context, input map[string]interface{}) (*core.ToolResult, error) {
	startTime := time.Now()

	url, _ := input["url"].(string)
	method, _ := input["method"].(string)
	if method == "" {
		method = "GET"
	}

	// 创建请求
	req := t.client.R().SetContext(ctx)

	// 设置请求头
	if headers, ok := input["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.SetHeader(key, strValue)
			}
		}
	}

	// 设置请求体
	if body, ok := input["body"].(string); ok && body != "" {
		req.SetBody(body)
	}

	// 设置超时
	if timeout, ok := input["timeout"].(float64); ok {
		client := httpclient.NewClient(&httpclient.Config{
			Timeout: time.Duration(timeout) * time.Second,
		})
		req = client.R().SetContext(ctx)

		// 重新设置headers和body
		if headers, ok := input["headers"].(map[string]interface{}); ok {
			for key, value := range headers {
				if strValue, ok := value.(string); ok {
					req.SetHeader(key, strValue)
				}
			}
		}
		if body, ok := input["body"].(string); ok && body != "" {
			req.SetBody(body)
		}
	}

	// 发送请求
	var resp *resty.Response
	var err error

	switch strings.ToUpper(method) {
	case "GET":
		resp, err = req.Get(url)
	case "POST":
		resp, err = req.Post(url)
	case "PUT":
		resp, err = req.Put(url)
	case "DELETE":
		resp, err = req.Delete(url)
	case "PATCH":
		resp, err = req.Patch(url)
	default:
		return &core.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("unsupported HTTP method: %s", method),
			ErrorCode: "METHOD_ERROR",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return &core.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("request failed: %v", err),
			ErrorCode: "HTTP_ERROR",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	// 读取响应体
	respBody := resp.Body()

	// 尝试解析为 JSON
	var jsonBody interface{}
	_ = json.Unmarshal(respBody, &jsonBody)

	result := &core.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"status_code": resp.StatusCode(),
			"status":      resp.Status(),
			"headers":     resp.Header(),
			"body":        string(respBody),
			"json":        jsonBody,
			"size":        len(respBody),
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// Validate 验证输入
func (t *HTTPRequestTool) Validate(input map[string]interface{}) error {
	url, ok := input["url"].(string)
	if !ok || url == "" {
		return &core.ErrInvalidInput{Field: "url", Message: "must be a non-empty string"}
	}

	return nil
}

// JSONParseTool JSON 解析工具
type JSONParseTool struct {
	name         string
	description  string
	category     string
	schema       *core.ToolSchema
	requiresAuth bool
	isDangerous  bool
}

// NewJSONParseTool 创建 JSON 解析工具
func NewJSONParseTool() *JSONParseTool {
	schema := &core.ToolSchema{
		Type: "object",
		Properties: map[string]core.PropertySchema{
			"json": {
				Type:        "string",
				Description: "JSON 字符串",
			},
			"path": {
				Type:        "string",
				Description: "JSON 路径（可选，如 $.user.name）",
			},
		},
		Required: []string{"json"},
	}

	return &JSONParseTool{
		name:         "json_parse",
		description:  "解析 JSON 数据",
		category:     "data",
		schema:       schema,
		requiresAuth: false,
		isDangerous:  false,
	}
}

// Name 返回工具名称
func (t *JSONParseTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *JSONParseTool) Description() string {
	return t.description
}

// Category 返回工具类别
func (t *JSONParseTool) Category() string {
	return t.category
}

// Schema 返回工具的 JSON Schema
func (t *JSONParseTool) Schema() *core.ToolSchema {
	return t.schema
}

// RequiresAuth 返回是否需要认证
func (t *JSONParseTool) RequiresAuth() bool {
	return t.requiresAuth
}

// IsDangerous 返回是否是危险操作
func (t *JSONParseTool) IsDangerous() bool {
	return t.isDangerous
}

// Execute 执行工具
func (t *JSONParseTool) Execute(ctx context.Context, input map[string]interface{}) (*core.ToolResult, error) {
	startTime := time.Now()

	jsonStr, _ := input["json"].(string)

	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return &core.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to parse JSON: %v", err),
			ErrorCode: "PARSE_ERROR",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	result := &core.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"parsed": data,
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// Validate 验证输入
func (t *JSONParseTool) Validate(input map[string]interface{}) error {
	jsonStr, ok := input["json"].(string)
	if !ok || jsonStr == "" {
		return &core.ErrInvalidInput{Field: "json", Message: "must be a non-empty string"}
	}

	return nil
}

// ShellExecuteTool Shell 命令执行工具
type ShellExecuteTool struct {
	name         string
	description  string
	category     string
	schema       *core.ToolSchema
	requiresAuth bool
	isDangerous  bool
}

// NewShellExecuteTool 创建 Shell 命令执行工具
func NewShellExecuteTool() *ShellExecuteTool {
	schema := &core.ToolSchema{
		Type: "object",
		Properties: map[string]core.PropertySchema{
			"command": {
				Type:        "string",
				Description: "要执行的命令",
			},
			"args": {
				Type:        "array",
				Description: "命令参数",
				Items: &core.PropertySchema{
					Type: "string",
				},
			},
			"timeout": {
				Type:        "integer",
				Description: "超时时间（秒）",
				Default:     30,
			},
		},
		Required: []string{"command"},
	}

	return &ShellExecuteTool{
		name:         "shell_execute",
		description:  "执行 Shell 命令",
		category:     "system",
		schema:       schema,
		requiresAuth: true,
		isDangerous:  true, // Shell 执行是危险操作
	}
}

// Name 返回工具名称
func (t *ShellExecuteTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *ShellExecuteTool) Description() string {
	return t.description
}

// Category 返回工具类别
func (t *ShellExecuteTool) Category() string {
	return t.category
}

// Schema 返回工具的 JSON Schema
func (t *ShellExecuteTool) Schema() *core.ToolSchema {
	return t.schema
}

// RequiresAuth 返回是否需要认证
func (t *ShellExecuteTool) RequiresAuth() bool {
	return t.requiresAuth
}

// IsDangerous 返回是否是危险操作
func (t *ShellExecuteTool) IsDangerous() bool {
	return t.isDangerous
}

// Execute 执行工具
func (t *ShellExecuteTool) Execute(ctx context.Context, input map[string]interface{}) (*core.ToolResult, error) {
	startTime := time.Now()

	command, _ := input["command"].(string)

	// 注意：实际生产环境中应该使用 os/exec 包并进行严格的安全检查
	// 这里仅作示例

	result := &core.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"command": command,
			"output":  "Command execution is disabled for security reasons in this example",
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// Validate 验证输入
func (t *ShellExecuteTool) Validate(input map[string]interface{}) error {
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return &core.ErrInvalidInput{Field: "command", Message: "must be a non-empty string"}
	}

	return nil
}
