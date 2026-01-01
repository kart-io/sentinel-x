# 测试覆盖率指南

本文档说明如何运行测试、生成覆盖率报告和提升测试覆盖率。

## 快速开始

### 运行所有测试

```bash
make test
```

### 生成覆盖率报告

```bash
make test-cover
```

### 生成 HTML 可视化报告

```bash
make test-cover-html
```

这将生成以下文件：
- `coverage.out` - 覆盖率数据文件
- `coverage.html` - HTML 可视化报告
- `test-output.log` - 测试输出日志

打开 HTML 报告：
```bash
open coverage.html
```

## 测试命令详解

### 1. 基础测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/rag/biz/...

# 显示详细输出
go test -v ./...

# 运行匹配模式的测试
go test -run TestQueryCache ./internal/rag/biz/
```

### 2. 覆盖率测试

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看覆盖率摘要
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html

# 查看总体覆盖率
go tool cover -func=coverage.out | grep total
```

### 3. 并发和竞态检测

```bash
# 启用竞态检测
go test -race ./...

# 随机化测试顺序
go test -shuffle=on ./...

# 设置超时
go test -timeout=30m ./...
```

### 4. 基准测试

```bash
# 运行基准测试
make bench

# 或直接使用 go test
go test -bench=. -benchmem ./...
```

## 覆盖率目标

### 总体目标

- **项目总体覆盖率**: ≥60%
- **核心业务逻辑**: ≥80%
- **工具函数**: ≥90%

### 按模块覆盖率目标

| 模块 | 目标覆盖率 | 当前状态 |
|------|-----------|---------|
| `internal/rag/biz` | ≥80% | ⚠️ 需改进 |
| `internal/user-center/biz` | ≥80% | ⚠️ 需改进 |
| `pkg/security` | ≥85% | ✅ 已达标 |
| `pkg/utils` | ≥90% | ✅ 已达标 |
| `pkg/llm` | ≥95% | ✅ 已达标 |
| `pkg/cache` | ≥85% | ✅ 已达标 |

## 编写测试指南

### 测试文件命名

- 测试文件名: `<file>_test.go`
- 与被测试文件位于同一包
- 示例: `cache.go` → `cache_test.go`

### 测试函数命名

```go
func TestFunctionName(t *testing.T) {
    // 基础测试
}

func TestFunctionName_Scenario(t *testing.T) {
    // 特定场景测试
}

func BenchmarkFunctionName(b *testing.B) {
    // 基准测试
}
```

### 测试结构

使用 Table-Driven Tests 模式：

```go
func TestQueryCache_Get(t *testing.T) {
    tests := []struct {
        name    string
        question string
        want    *model.QueryResult
        wantErr bool
    }{
        {
            name:     "缓存命中",
            question: "什么是 RAG？",
            want:     &model.QueryResult{Answer: "..."},
            wantErr:  false,
        },
        {
            name:     "缓存未命中",
            question: "不存在的问题",
            want:     nil,
            wantErr:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### 使用 testify 断言

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    // require: 失败时立即停止
    require.NotNil(t, obj)

    // assert: 失败时继续执行
    assert.Equal(t, expected, actual)
    assert.NoError(t, err)
    assert.Contains(t, str, substr)
}
```

### Mock 和 Stub

```go
// 使用接口实现 mock
type mockStore struct {
    GetFunc func(ctx context.Context, key string) (*model.QueryResult, error)
}

func (m *mockStore) Get(ctx context.Context, key string) (*model.QueryResult, error) {
    return m.GetFunc(ctx, key)
}

// 在测试中使用
func TestWithMock(t *testing.T) {
    mock := &mockStore{
        GetFunc: func(ctx context.Context, key string) (*model.QueryResult, error) {
            return &model.QueryResult{Answer: "mocked"}, nil
        },
    }

    // 使用 mock 进行测试
}
```

## 测试覆盖率提升策略

### 1. 识别未覆盖代码

```bash
# 运行覆盖率脚本
./scripts/test-coverage.sh

# 查看低覆盖率文件（<50%）
# 输出会列出需要重点关注的文件
```

### 2. 优先级排序

1. **高优先级**: 核心业务逻辑（biz 层）
2. **中优先级**: 数据访问层（store 层）、处理器（handler 层）
3. **低优先级**: 自动生成的代码（protobuf、k8s client）

### 3. 测试类型

- **单元测试**: 测试单个函数或方法
- **集成测试**: 测试多个组件协作
- **边界测试**: 测试边界条件和异常情况
- **性能测试**: 基准测试和压力测试

### 4. 常见未覆盖场景

- 错误处理路径
- 边界条件（nil, 0, 空字符串）
- 并发场景
- 配置分支（if/else）
- 清理逻辑（defer, Close）

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: Test Coverage
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Run tests with coverage
        run: make test-cover

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests
          fail_ci_if_error: true
```

## 测试最佳实践

### ✅ 推荐做法

1. **独立性**: 每个测试应该独立运行，不依赖其他测试
2. **可重复**: 测试结果应该是确定性的，多次运行结果一致
3. **快速**: 单元测试应该在毫秒级完成
4. **清晰**: 测试名称应该清楚地描述测试场景
5. **完整**: 测试正常路径和异常路径

### ❌ 避免

1. **过度 Mock**: 不要过度使用 mock，优先测试真实逻辑
2. **脆弱测试**: 避免测试依赖实现细节
3. **重复测试**: 避免测试重复覆盖相同代码路径
4. **慢速测试**: 避免在单元测试中进行网络调用或磁盘 I/O
5. **魔法数字**: 使用有意义的常量而非魔法数字

## 常见问题

### Q: 如何跳过慢速测试？

```go
func TestSlowOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过耗时测试")
    }
    // 慢速测试逻辑
}

// 运行时跳过
go test -short ./...
```

### Q: 如何测试需要外部依赖的代码？

使用环境变量或构建标签来跳过：

```go
func TestWithRedis(t *testing.T) {
    if os.Getenv("REDIS_ADDR") == "" {
        t.Skip("Redis 不可用，跳过测试")
    }
    // 测试逻辑
}
```

### Q: 如何测试并发代码？

```go
func TestConcurrent(t *testing.T) {
    const goroutines = 100
    var wg sync.WaitGroup
    wg.Add(goroutines)

    for i := 0; i < goroutines; i++ {
        go func() {
            defer wg.Done()
            // 并发操作
        }()
    }

    wg.Wait()
    // 验证结果
}
```

## 参考资料

- [Go 测试官方文档](https://golang.org/pkg/testing/)
- [testify 断言库](https://github.com/stretchr/testify)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Go 测试最佳实践](https://golang.org/doc/effective_go#testing)
