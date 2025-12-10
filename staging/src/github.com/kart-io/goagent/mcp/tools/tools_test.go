package tools

import (
	"testing"

	"github.com/kart-io/goagent/mcp/core"
)

// 测试所有工具都正确实现了 Tool 接口
func TestToolsImplementInterface(t *testing.T) {
	tests := []struct {
		name string
		tool core.Tool
	}{
		{"ReadFileTool", NewReadFileTool()},
		{"WriteFileTool", NewWriteFileTool()},
		{"ListDirectoryTool", NewListDirectoryTool()},
		{"SearchFilesTool", NewSearchFilesTool()},
		{"HTTPRequestTool", NewHTTPRequestTool()},
		{"JSONParseTool", NewJSONParseTool()},
		{"ShellExecuteTool", NewShellExecuteTool()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试所有接口方法都能正常调用
			if tt.tool.Name() == "" {
				t.Error("Name() 返回空字符串")
			}
			if tt.tool.Description() == "" {
				t.Error("Description() 返回空字符串")
			}
			if tt.tool.Category() == "" {
				t.Error("Category() 返回空字符串")
			}
			if tt.tool.Schema() == nil {
				t.Error("Schema() 返回 nil")
			}

			// 验证方法返回值的类型
			_ = tt.tool.RequiresAuth()
			_ = tt.tool.IsDangerous()

			// 验证工具的基本信息
			t.Logf("工具: %s, 类别: %s, 需要认证: %v, 危险操作: %v",
				tt.tool.Name(),
				tt.tool.Category(),
				tt.tool.RequiresAuth(),
				tt.tool.IsDangerous())
		})
	}
}

// 测试工具的基本属性
func TestToolProperties(t *testing.T) {
	tests := []struct {
		name         string
		tool         core.Tool
		expectedName string
		expectedCat  string
		reqAuth      bool
		isDangerous  bool
	}{
		{"ReadFileTool", NewReadFileTool(), "read_file", "filesystem", false, false},
		{"WriteFileTool", NewWriteFileTool(), "write_file", "filesystem", true, true},
		{"ListDirectoryTool", NewListDirectoryTool(), "list_directory", "filesystem", false, false},
		{"SearchFilesTool", NewSearchFilesTool(), "search_files", "filesystem", false, false},
		{"HTTPRequestTool", NewHTTPRequestTool(), "http_request", "network", false, false},
		{"JSONParseTool", NewJSONParseTool(), "json_parse", "data", false, false},
		{"ShellExecuteTool", NewShellExecuteTool(), "shell_execute", "system", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tool.Name() != tt.expectedName {
				t.Errorf("Name() = %v, 期望 %v", tt.tool.Name(), tt.expectedName)
			}
			if tt.tool.Category() != tt.expectedCat {
				t.Errorf("Category() = %v, 期望 %v", tt.tool.Category(), tt.expectedCat)
			}
			if tt.tool.RequiresAuth() != tt.reqAuth {
				t.Errorf("RequiresAuth() = %v, 期望 %v", tt.tool.RequiresAuth(), tt.reqAuth)
			}
			if tt.tool.IsDangerous() != tt.isDangerous {
				t.Errorf("IsDangerous() = %v, 期望 %v", tt.tool.IsDangerous(), tt.isDangerous)
			}
		})
	}
}
