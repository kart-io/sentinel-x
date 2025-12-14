# Sentinel-X 错误码体系迁移指南

## 概述

本文档提供从旧版错误码体系迁移到新版错误码体系的完整指南。新版错误码体系参考 [sentinel-x 项目](https://github.com/kart-io/sentinel-x) 设计规范，提供更强大、更规范的错误处理能力。

## 迁移目标

- 错误码全局唯一
- 支持按模块分类
- 具备明确业务语义
- 支持多语言（中/英文）
- 兼容 HTTP 和 gRPC

## 新版特性

### 1. 结构化错误码

新版采用 **7位数字** 格式 `AABBCCC`：

```
AA  : 服务/模块代码 (00-99)
BB  : 类别代码 (00-99)
CCC : 序列号 (000-999)
```

### 2. 完整的 Errno 结构

```go
type Errno struct {
    Code      int         // 唯一错误码
    HTTP      int         // HTTP 状态码
    GRPCCode  codes.Code  // gRPC 状态码
    MessageEN string      // 英文错误消息
    MessageZH string      // 中文错误消息
    cause     error       // 底层错误
}
```

### 3. 丰富的辅助方法

```go
// 创建带原因的错误
err.WithCause(cause)

// 自定义消息
err.WithMessage("custom message")
err.WithMessagef("user %s not found", username)
err.WithMessageZH("用户不存在")
err.WithMessages("User not found", "用户不存在")

// 获取多语言消息
err.Message("zh-CN")  // 返回中文
err.Message("en")     // 返回英文

// 获取状态码
err.HTTPStatus()      // HTTP 状态码
err.GRPCStatus()      // gRPC 状态码
```

## 迁移步骤

### 步骤 1: 更新导入

```go
import (
    "github.com/kart-io/sentinel-x/pkg/errors"
    "github.com/kart-io/sentinel-x/pkg/response"
)
```

### 步骤 2: 替换错误码常量

**旧代码：**
```go
if response.IsError(err, response.CodeInvalidParam) {
    // ...
}
```

**新代码：**
```go
if errors.IsCode(err, errors.ErrInvalidParam.Code) {
    // ...
}
```

### 步骤 3: 替换错误变量

**旧代码：**
```go
return response.ErrInvalidParam.WithMessage("invalid username")
```

**新代码：**
```go
return errors.ErrInvalidParam.WithMessage("invalid username")
```

### 步骤 4: 更新 HTTP Handler

**新代码：**
```go
func handler(c transport.Context) {
    if err != nil {
        response.Fail(c, errors.ErrInvalidParam)
        return
    }
    response.OK(c, data)
}

// 带自定义消息
func handler(c transport.Context) {
    if err != nil {
        response.Fail(c, errors.ErrInvalidParam.WithMessage("username is required"))
        return
    }
    response.OK(c, data)
}

// 多语言支持
func handler(c transport.Context) {
    lang := c.Header("Accept-Language")
    if err != nil {
        response.FailWithLang(c, errors.ErrInvalidParam, lang)
        return
    }
    response.OK(c, data)
}
```

### 步骤 5: 更新 gRPC Handler

```go
func (s *Service) Method(ctx context.Context, req *Request) (*Response, error) {
    if err != nil {
        errno := errors.ErrInvalidParam.WithMessage("invalid param")
        return nil, status.Error(errno.GRPCCode, errno.MessageEN)
    }
    return &Response{}, nil
}
```

### 步骤 6: 自定义业务错误码

**方式一：使用 Builder 模式（推荐）**
```go
package myservice

import "github.com/kart-io/sentinel-x/pkg/errors"

// 定义服务编号（与团队协调分配）
const ServiceMyService = 25

func init() {
    errors.RegisterService(ServiceMyService, "my-service")
}

var (
    // 使用预设 Builder
    ErrOrderNotFound = errors.NewNotFoundError(ServiceMyService, 1).
        Message("Order not found", "订单不存在").
        MustBuild()

    ErrOrderInvalid = errors.NewRequestError(ServiceMyService, 1).
        Message("Invalid order", "订单无效").
        MustBuild()

    // 使用完整 Builder
    ErrOrderExpired = errors.NewBuilder(ServiceMyService, errors.CategoryConflict, 1).
        HTTP(http.StatusGone).
        GRPC(codes.FailedPrecondition).
        Message("Order expired", "订单已过期").
        MustBuild()
)
```

**方式二：直接注册**
```go
var ErrMyServiceSpecificError = errors.Register(&errors.Errno{
    Code:      errors.MakeCode(ServiceMyService, errors.CategoryRequest, 1),
    HTTP:      http.StatusBadRequest,
    GRPCCode:  codes.InvalidArgument,
    MessageEN: "Specific error message",
    MessageZH: "特定错误消息",
})
```

## 兼容性保证

### 1. 旧错误码自动转换

兼容层自动处理旧错误码：

```go
// 旧错误码自动转换为新错误码
newCode := errors.LegacyToNewCode(1001)  // 1001 -> 新错误码

// 从旧错误码获取 Errno
errno := errors.FromLegacyCode(1001)
```

### 2. 向后兼容的响应格式

响应格式保持不变：

```json
{
    "code": 1001,
    "message": "Invalid parameter",
    "request_id": "req-123",
    "timestamp": 1699999999999
}
```

### 3. HTTP 状态码映射不变

旧版和新版的 HTTP 状态码映射保持一致。

## 错误码对照速查

| 旧错误码 | 新错误码 | 说明 |
|---------|---------|------|
| 0 | 0 | 成功 |
| 1000 | 1000 | 请求错误 |
| 1001 | 1001 | 参数无效 |
| 2000 | 2000 | 未认证 |
| 3000 | 3000 | 禁止访问 |
| 4000 | 4000 | 资源不存在 |
| 5000 | 7000 | 服务器内部错误 |
| 5001 | 8000 | 数据库错误 |
| 5002 | 9000 | 缓存错误 |
| 5003 | 10001 | 服务不可用 |
| 5004 | 11000 | 操作超时 |

## 错误类别

### 请求错误 (01xx)
```go
errors.ErrBadRequest       // 1000 - 请求错误
errors.ErrInvalidParam     // 1001 - 参数无效
errors.ErrMissingParam     // 1002 - 缺少必需参数
errors.ErrInvalidFormat    // 1003 - 格式无效
errors.ErrValidationFailed // 1004 - 验证失败
```

### 认证错误 (02xx)
```go
errors.ErrUnauthorized       // 2000 - 未认证
errors.ErrInvalidToken       // 2001 - 令牌无效
errors.ErrTokenExpired       // 2002 - 令牌已过期
errors.ErrInvalidCredentials // 2003 - 凭证无效
```

### 权限错误 (03xx)
```go
errors.ErrForbidden      // 3000 - 禁止访问
errors.ErrNoPermission   // 3001 - 无权限
errors.ErrResourceLocked // 3002 - 资源已锁定
```

### 资源错误 (04xx)
```go
errors.ErrNotFound       // 4000 - 资源不存在
errors.ErrUserNotFound   // 4001 - 用户不存在
errors.ErrRecordNotFound // 4002 - 记录不存在
```

### 冲突错误 (05xx)
```go
errors.ErrConflict         // 5000 - 资源冲突
errors.ErrAlreadyExists    // 5001 - 资源已存在
errors.ErrDuplicateKey     // 5002 - 键值重复
errors.ErrVersionConflict  // 5003 - 版本冲突
```

### 限流错误 (06xx)
```go
errors.ErrTooManyRequests    // 6000 - 请求过于频繁
errors.ErrRateLimitExceeded  // 6001 - 超出速率限制
errors.ErrQuotaExceeded      // 6002 - 配额已用尽
```

### 内部错误 (07xx)
```go
errors.ErrInternal       // 7000 - 服务器内部错误
errors.ErrUnknown        // 7001 - 未知错误
errors.ErrPanic          // 7002 - 服务崩溃
errors.ErrNotImplemented // 7003 - 功能未实现
```

### 数据库错误 (08xx)
```go
errors.ErrDatabase      // 8000 - 数据库错误
errors.ErrDBConnection  // 8001 - 数据库连接失败
errors.ErrDBQuery       // 8002 - 数据库查询失败
errors.ErrDBTransaction // 8003 - 数据库事务失败
```

### 缓存错误 (09xx)
```go
errors.ErrCache           // 9000 - 缓存错误
errors.ErrCacheConnection // 9001 - 缓存连接失败
errors.ErrCacheMiss       // 9002 - 缓存未命中
```

### 网络错误 (10xx)
```go
errors.ErrNetwork            // 10000 - 网络错误
errors.ErrServiceUnavailable // 10001 - 服务不可用
errors.ErrConnectionRefused  // 10002 - 连接被拒绝
```

### 超时错误 (11xx)
```go
errors.ErrTimeout        // 11000 - 操作超时
errors.ErrRequestTimeout // 11001 - 请求超时
errors.ErrGatewayTimeout // 11002 - 网关超时
```

### 配置错误 (12xx)
```go
errors.ErrConfig         // 12000 - 配置错误
errors.ErrConfigNotFound // 12001 - 配置不存在
errors.ErrConfigInvalid  // 12002 - 配置无效
```

## 最佳实践

### 1. 错误包装

```go
// 推荐：保留原始错误上下文
return errors.ErrDatabase.WithCause(dbErr).WithMessage("failed to save user")

// 不推荐：丢失原始错误信息
return errors.ErrDatabase
```

### 2. 多语言支持

```go
// 从请求中获取语言
lang := c.Header("Accept-Language")

// 返回对应语言的错误消息
response.FailWithLang(c, err, lang)
```

### 3. 错误检查

```go
// 推荐：使用错误码检查
if errors.IsCode(err, errors.ErrNotFound.Code) {
    // 处理未找到错误
}

// 或使用 errors.Is
if errors.ErrNotFound.Is(err) {
    // 处理未找到错误
}
```

### 4. 业务错误码管理

```go
// 在服务模块中集中定义错误码
// myservice/errors.go

package myservice

import "github.com/kart-io/sentinel-x/pkg/errors"

const ServiceCode = 25  // 申请的服务编号

func init() {
    errors.RegisterService(ServiceCode, "my-service")
}

var (
    ErrOrderNotFound = errors.NewNotFoundError(ServiceCode, 1).
        Message("Order not found", "订单不存在").
        MustBuild()
)
```

## 文件清单

```
pkg/errors/
├── errno.go        # 核心 Errno 类型定义
├── code.go         # 错误码常量和工具函数
├── base.go         # 通用基础错误码
├── builder.go      # Builder 模式支持
├── compat.go       # 旧版兼容层
└── example_test.go # 使用示例

pkg/response/
├── response.go     # 响应结构（使用 pkg/errors）
└── writer.go       # 响应写入（支持 Errno）

example/errors/
├── order/          # 订单服务错误码示例
├── user/           # 用户服务错误码示例
├── scheduler/      # 调度服务错误码示例
└── thirdparty/     # 第三方服务错误码示例

docs/design/
├── error-code-design.md    # 错误码设计文档
├── error-code-migration.md # 迁移对照表
└── MIGRATION.md           # 本文档
```

## 常见问题

### Q1: 如何申请新的服务编号？

联系架构团队，在 `pkg/errors/code.go` 中添加新的服务常量，或在业务模块中自行定义（20-79 范围）。

### Q2: 旧代码可以不迁移吗？

可以。兼容层会自动处理旧错误码，但建议逐步迁移以享受新特性。

### Q3: 如何处理第三方服务错误？

使用 90-99 的服务编号定义第三方服务错误码。参考 `example/errors/thirdparty/` 示例。

### Q4: 错误消息可以动态修改吗？

可以。使用 `WithMessage` 或 `WithMessagef` 方法。

---

**参考资料：**
- [sentinel-x 错误码设计规范](https://github.com/kart-io/sentinel-x)
- [腾讯云 API 3.0 错误码设计](https://cloud.tencent.com/document/api/213/15694)
- [gRPC 状态码](https://grpc.io/docs/guides/status-codes/)
