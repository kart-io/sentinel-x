# Middleware Unit Tests

本目录包含 Sentinel-X 中间件模块的单元测试。

## 测试覆盖

### 1. CORS 中间件测试 (cors_test.go)

测试 `pkg/middleware/cors.go` 的 CORS 中间件功能。

#### 测试用例

- **TestCORSConfigValidate**: 验证 CORS 配置校验逻辑
  - 有效配置（指定源）
  - 空源列表应失败
  - 通配符与凭证同时使用应失败
  - 无凭证的通配符配置有效
  - 多个源配置有效

- **TestCORSWithConfig_PreflightRequest**: 测试预检请求（OPTIONS）处理
  - 验证 CORS 头正确设置
  - 验证 Allow-Methods, Allow-Headers, Max-Age 头
  - 验证凭证头设置

- **TestCORSWithConfig_NormalRequest**: 测试正常请求的 CORS 处理
  - 验证 CORS 头设置
  - 验证 Expose-Headers 设置
  - 验证请求继续传递给下一个处理器

- **TestCORSWithConfig_DisallowedOrigin**: 测试不允许的源
  - 验证不允许的源不设置 CORS 头
  - 验证请求仍然被处理

- **TestCORSWithConfig_WildcardOrigin**: 测试通配符源
  - 验证通配符允许任意源
  - 验证响应头正确设置

- **TestCORSWithConfig_NoOriginHeader**: 测试无 Origin 头的请求
  - 验证无 Origin 头时不设置 CORS 头
  - 验证请求正常处理

- **TestCORS_DefaultConfig**: 测试默认配置
  - 验证空配置导致 panic

- **TestCORSWithConfig_Panic**: 测试无效配置
  - 验证无效配置导致 panic

**覆盖率**: 97.4% - 100%

---

### 2. Recovery 中间件测试 (recovery_test.go)

测试 `pkg/middleware/recovery.go` 的 panic 恢复功能。

#### 测试用例

- **TestRecovery_NoPanic**: 测试正常请求（无 panic）
  - 验证处理器正常调用
  - 验证无错误响应

- **TestRecovery_CatchesPanic**: 测试 panic 捕获
  - 验证 panic 被捕获
  - 验证返回 JSON 错误响应
  - 验证 HTTP 500 状态码

- **TestRecoveryWithConfig_StackTrace**: 测试堆栈跟踪功能
  - 启用堆栈跟踪
  - 禁用堆栈跟踪
  - 验证响应状态码

- **TestRecoveryWithConfig_OnPanicCallback**: 测试 panic 回调
  - 验证回调函数被调用
  - 验证 panic 错误传递给回调
  - 验证堆栈信息传递给回调

- **TestRecoveryWithConfig_PanicWithDifferentTypes**: 测试不同类型的 panic
  - 字符串 panic
  - 错误 panic
  - 整数 panic
  - nil panic
  - 验证所有类型都被正确处理

- **TestRecovery_DefaultConfig**: 测试默认配置
  - 验证默认配置正常工作

**覆盖率**: 100%

---

### 3. RequestID 中间件测试 (request_id_test.go)

测试 `pkg/middleware/request_id.go` 的请求 ID 生成和管理。

#### 测试用例

- **TestRequestID_GeneratesID**: 测试 ID 生成
  - 验证自动生成请求 ID
  - 验证 ID 格式（32 字符十六进制）
  - 验证响应头设置

- **TestRequestID_PreservesExistingID**: 测试保留现有 ID
  - 验证从请求头提取现有 ID
  - 验证不覆盖现有 ID

- **TestRequestIDWithConfig_CustomHeader**: 测试自定义头名称
  - 验证自定义头正确使用
  - 验证默认头未设置

- **TestRequestIDWithConfig_CustomGenerator**: 测试自定义生成器
  - 验证自定义生成器被调用
  - 验证生成的 ID 正确使用

- **TestRequestID_StoresInContext**: 测试上下文存储
  - 验证 ID 存储在上下文中
  - 验证 GetRequestID 正确提取 ID
  - 验证上下文 ID 与响应头匹配

- **TestGetRequestID_NotFound**: 测试未找到 ID
  - 验证空上下文返回空字符串

- **TestGetRequestID_WrongType**: 测试错误类型
  - 验证错误类型值返回空字符串

- **TestRequestIDWithConfig_Defaults**: 测试默认值
  - 验证空配置使用默认值

- **TestGenerateRequestID_Uniqueness**: 测试 ID 唯一性
  - 生成 100 个 ID
  - 验证所有 ID 唯一
  - 验证 ID 格式正确

- **TestRequestID_MultipleRequests**: 测试多请求
  - 验证每个请求获得唯一 ID

- **TestRequestIDWithConfig_EmptyHeader**: 测试空头配置
  - 验证空头使用默认值

**覆盖率**: 100%

---

### 4. Timeout 中间件测试 (timeout_test.go)

测试 `pkg/middleware/timeout.go` 的超时处理功能。

#### 测试用例

- **TestTimeout_NormalRequest**: 测试正常请求
  - 快速请求在超时前完成
  - 验证无超时响应

- **TestTimeout_SlowRequest**: 测试慢请求
  - 请求超过超时时间
  - 验证超时错误响应
  - 验证 HTTP 408 状态码

- **TestTimeoutWithConfig_SkipPaths**: 测试跳过路径
  - 跳过的路径不应用超时
  - 正常路径应用超时
  - 多个跳过路径

- **TestTimeoutWithConfig_DefaultTimeout**: 测试默认超时
  - 验证空配置使用默认超时

- **TestTimeout_ContextDeadline**: 测试上下文截止时间
  - 验证上下文设置了截止时间
  - 验证截止时间正确

- **TestTimeout_CanceledContext**: 测试取消的上下文
  - 超时后上下文应被取消
  - 验证上下文错误为 DeadlineExceeded

- **TestTimeout_GoroutineDoesNotLeak**: 测试协程泄漏
  - 多次请求验证无协程泄漏

- **TestTimeout_PanicInHandler**: 测试处理器 panic
  - 处理器 panic 不应导致中间件 panic
  - 验证协程正确清理

- **TestTimeout_MultipleTimeouts**: 测试多个并发超时
  - 并发慢请求
  - 验证所有请求正确超时

- **TestTimeoutWithConfig_ZeroTimeout**: 测试零超时
  - 验证零超时使用默认值

- **TestTimeout_VeryShortTimeout**: 测试极短超时
  - 验证短超时正确触发

**覆盖率**: 100%

---

## 测试工具

### Mock Context (mock_test.go)

实现了 `transport.Context` 接口的测试模拟对象，用于所有中间件测试。

#### 功能

- 实现完整的 `transport.Context` 接口
- 支持请求/响应头管理
- 支持 JSON 响应捕获
- 线程安全（使用互斥锁）

---

## 运行测试

### 运行所有测试

```bash
go test ./pkg/middleware/...
```

### 运行特定测试

```bash
# CORS 测试
go test ./pkg/middleware/... -run TestCORS

# Recovery 测试
go test ./pkg/middleware/... -run TestRecovery

# RequestID 测试
go test ./pkg/middleware/... -run TestRequestID

# Timeout 测试
go test ./pkg/middleware/... -run TestTimeout
```

### 查看覆盖率

```bash
go test -coverprofile=coverage.out ./pkg/middleware/...
go tool cover -html=coverage.out
```

### 详细输出

```bash
go test -v ./pkg/middleware/...
```

---

## 测试统计

- **总测试用例**: 50+
- **平均覆盖率**: 97-100%
- **测试文件**: 5 个
- **代码行数**: 约 800 行

---

## 注意事项

1. **超时测试**: 部分测试包含 `time.Sleep`，执行时间较长
2. **并发测试**: 部分测试验证并发安全性
3. **Mock 对象**: 所有测试使用统一的 mock context
4. **表驱动测试**: 使用 Go 标准的表驱动测试模式

---

## 维护指南

### 添加新测试

1. 在相应的 `*_test.go` 文件中添加测试函数
2. 使用 `newMockContext()` 创建测试上下文
3. 遵循表驱动测试模式
4. 确保测试覆盖边界情况

### 更新 Mock Context

如果 `transport.Context` 接口发生变化：

1. 更新 `mock_test.go` 中的 `mockContext` 实现
2. 确保实现所有新方法
3. 运行所有测试验证兼容性

### 测试命名规范

- 测试函数: `Test<功能>_<场景>`
- 子测试: 描述性名称，使用下划线分隔
- 表驱动测试: 使用 `tests` 切片，每项包含 `name` 字段
