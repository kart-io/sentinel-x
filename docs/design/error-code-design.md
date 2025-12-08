# Sentinel-X 错误码设计规范

> 参考 [onex 项目](https://github.com/onexstack/onex) 错误码设计规范

## 1. 概述

本文档定义了 Sentinel-X 项目的统一错误码规范，旨在实现：

- **全局唯一性**：每个错误码在整个系统中唯一
- **模块可追溯**：通过错误码快速定位问题所属服务和模块
- **语义明确**：错误码和错误消息具有明确的业务含义
- **多端统一**：前后端及所有调用方使用统一的错误码体系
- **国际化支持**：支持中英文错误消息

## 2. 错误码格式

### 2.1 编号规范

采用 **7位数字** 格式：`AABBCCC`

```
┌──────────────────────────────────────────────────────────────┐
│                    错误码格式: AABBCCC                        │
├──────────────────────────────────────────────────────────────┤
│  AA (00-99)  │  服务/模块代码 - 标识错误来源的服务或模块        │
│  BB (00-99)  │  类别代码 - 标识错误的具体类别                  │
│  CCC (000-999) │  序列号 - 该类别下的具体错误编号              │
└──────────────────────────────────────────────────────────────┘

示例: 0001001 = 通用服务(00) + 请求验证错误(01) + 参数无效(001)
```

### 2.2 服务/模块代码 (AA)

| 代码范围 | 分类 | 说明 |
|---------|------|------|
| `00` | 通用错误 | 所有服务共用的基础错误 |
| `01` | 网关服务 | Gateway Service |
| `02` | 用户服务 | User Service |
| `03` | 调度服务 | Scheduler Service |
| `04` | API 服务 | API Service |
| `05-09` | 保留 | 核心服务扩展预留 |
| `10-19` | 基础设施 | 数据库、缓存、消息队列等 |
| `20-79` | 业务服务 | 各业务模块 |
| `80-89` | 内部服务 | 内部工具服务 |
| `90-99` | 第三方服务 | 外部服务调用错误 |

### 2.3 类别代码 (BB)

| 代码 | 类别 | HTTP状态码 | gRPC状态码 | 说明 |
|-----|------|-----------|-----------|------|
| `00` | 成功 | 200 | OK | 操作成功 |
| `01` | 请求错误 | 400 | InvalidArgument | 请求参数错误、格式错误 |
| `02` | 认证错误 | 401 | Unauthenticated | 身份认证失败 |
| `03` | 授权错误 | 403 | PermissionDenied | 权限不足 |
| `04` | 资源错误 | 404 | NotFound | 资源不存在 |
| `05` | 冲突错误 | 409 | AlreadyExists | 资源冲突、重复 |
| `06` | 限流错误 | 429 | ResourceExhausted | 请求频率超限 |
| `07` | 内部错误 | 500 | Internal | 服务器内部错误 |
| `08` | 数据库错误 | 500 | Internal | 数据库操作错误 |
| `09` | 缓存错误 | 500 | Internal | 缓存操作错误 |
| `10` | 网络错误 | 502/503 | Unavailable | 网络通信错误 |
| `11` | 超时错误 | 504 | DeadlineExceeded | 操作超时 |
| `12` | 配置错误 | 500 | Internal | 配置错误 |

## 3. 错误分层

### 3.1 系统级错误 (System Errors)

通用的、与业务无关的系统错误，所有服务共用。

```
服务代码: 00 (通用)
错误码范围: 0000000 - 0099999
```

**示例：**
- `0001001` - 参数无效 (Invalid Parameter)
- `0002001` - 未认证 (Unauthorized)
- `0007001` - 内部服务器错误 (Internal Server Error)

### 3.2 业务级错误 (Business Errors)

与特定业务逻辑相关的错误，按服务划分。

```
服务代码: 01-89
错误码范围: 0100000 - 8999999
```

**示例：**
- `0204001` - 用户不存在 (User Not Found)
- `0305001` - 任务已存在 (Task Already Exists)

### 3.3 第三方错误 (Third-party Errors)

调用外部服务时产生的错误。

```
服务代码: 90-99
错误码范围: 9000000 - 9999999
```

**示例：**
- `9010001` - 支付服务调用失败 (Payment Service Error)
- `9110001` - 短信服务调用超时 (SMS Service Timeout)

## 4. 通用错误码定义

### 4.1 成功响应

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0 | OK | 200 | Success | 成功 |

### 4.2 请求错误 (01xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0001000 | ErrBadRequest | 400 | Bad request | 请求错误 |
| 0001001 | ErrInvalidParam | 400 | Invalid parameter | 参数无效 |
| 0001002 | ErrMissingParam | 400 | Missing required parameter | 缺少必需参数 |
| 0001003 | ErrInvalidFormat | 400 | Invalid format | 格式无效 |
| 0001004 | ErrValidationFailed | 400 | Validation failed | 验证失败 |
| 0001005 | ErrRequestTooLarge | 400 | Request entity too large | 请求体过大 |
| 0001006 | ErrUnsupportedMediaType | 415 | Unsupported media type | 不支持的媒体类型 |

### 4.3 认证错误 (02xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0002000 | ErrUnauthorized | 401 | Unauthorized | 未认证 |
| 0002001 | ErrInvalidToken | 401 | Invalid token | 令牌无效 |
| 0002002 | ErrTokenExpired | 401 | Token expired | 令牌已过期 |
| 0002003 | ErrInvalidCredentials | 401 | Invalid credentials | 凭证无效 |
| 0002004 | ErrTokenRevoked | 401 | Token revoked | 令牌已撤销 |
| 0002005 | ErrSessionExpired | 401 | Session expired | 会话已过期 |

### 4.4 授权错误 (03xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0003000 | ErrForbidden | 403 | Forbidden | 禁止访问 |
| 0003001 | ErrNoPermission | 403 | No permission | 无权限 |
| 0003002 | ErrResourceLocked | 423 | Resource locked | 资源已锁定 |
| 0003003 | ErrAccountDisabled | 403 | Account disabled | 账号已禁用 |
| 0003004 | ErrIPBlocked | 403 | IP blocked | IP 已被封禁 |

### 4.5 资源错误 (04xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0004000 | ErrNotFound | 404 | Resource not found | 资源不存在 |
| 0004001 | ErrUserNotFound | 404 | User not found | 用户不存在 |
| 0004002 | ErrRecordNotFound | 404 | Record not found | 记录不存在 |
| 0004003 | ErrFileNotFound | 404 | File not found | 文件不存在 |
| 0004004 | ErrRouteNotFound | 404 | Route not found | 路由不存在 |

### 4.6 冲突错误 (05xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0005000 | ErrConflict | 409 | Resource conflict | 资源冲突 |
| 0005001 | ErrAlreadyExists | 409 | Resource already exists | 资源已存在 |
| 0005002 | ErrDuplicateKey | 409 | Duplicate key | 键值重复 |
| 0005003 | ErrVersionConflict | 409 | Version conflict | 版本冲突 |

### 4.7 限流错误 (06xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0006000 | ErrTooManyRequests | 429 | Too many requests | 请求过于频繁 |
| 0006001 | ErrRateLimitExceeded | 429 | Rate limit exceeded | 超出速率限制 |
| 0006002 | ErrQuotaExceeded | 429 | Quota exceeded | 配额已用尽 |

### 4.8 内部错误 (07xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0007000 | ErrInternal | 500 | Internal server error | 服务器内部错误 |
| 0007001 | ErrUnknown | 500 | Unknown error | 未知错误 |
| 0007002 | ErrPanic | 500 | Service panic | 服务崩溃 |
| 0007003 | ErrNotImplemented | 501 | Not implemented | 功能未实现 |

### 4.9 数据库错误 (08xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0008000 | ErrDatabase | 500 | Database error | 数据库错误 |
| 0008001 | ErrDBConnection | 500 | Database connection failed | 数据库连接失败 |
| 0008002 | ErrDBQuery | 500 | Database query failed | 数据库查询失败 |
| 0008003 | ErrDBTransaction | 500 | Database transaction failed | 数据库事务失败 |
| 0008004 | ErrDBDeadlock | 500 | Database deadlock | 数据库死锁 |

### 4.10 缓存错误 (09xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0009000 | ErrCache | 500 | Cache error | 缓存错误 |
| 0009001 | ErrCacheConnection | 500 | Cache connection failed | 缓存连接失败 |
| 0009002 | ErrCacheMiss | 500 | Cache miss | 缓存未命中 |
| 0009003 | ErrCacheExpired | 500 | Cache expired | 缓存已过期 |

### 4.11 网络错误 (10xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0010000 | ErrNetwork | 502 | Network error | 网络错误 |
| 0010001 | ErrServiceUnavailable | 503 | Service unavailable | 服务不可用 |
| 0010002 | ErrConnectionRefused | 502 | Connection refused | 连接被拒绝 |
| 0010003 | ErrDNSResolution | 502 | DNS resolution failed | DNS 解析失败 |

### 4.12 超时错误 (11xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0011000 | ErrTimeout | 504 | Operation timeout | 操作超时 |
| 0011001 | ErrRequestTimeout | 408 | Request timeout | 请求超时 |
| 0011002 | ErrGatewayTimeout | 504 | Gateway timeout | 网关超时 |
| 0011003 | ErrContextCanceled | 499 | Context canceled | 上下文已取消 |

### 4.13 配置错误 (12xx)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0012000 | ErrConfig | 500 | Configuration error | 配置错误 |
| 0012001 | ErrConfigNotFound | 500 | Configuration not found | 配置不存在 |
| 0012002 | ErrConfigInvalid | 500 | Invalid configuration | 配置无效 |

## 5. 业务服务错误码

### 5.1 用户服务 (02)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0201001 | ErrUserInvalidUsername | 400 | Invalid username | 用户名无效 |
| 0201002 | ErrUserInvalidPassword | 400 | Invalid password | 密码无效 |
| 0201003 | ErrUserInvalidEmail | 400 | Invalid email | 邮箱无效 |
| 0202001 | ErrUserLoginFailed | 401 | Login failed | 登录失败 |
| 0202002 | ErrUserPasswordWrong | 401 | Wrong password | 密码错误 |
| 0204001 | ErrUserNotFound | 404 | User not found | 用户不存在 |
| 0205001 | ErrUserAlreadyExists | 409 | User already exists | 用户已存在 |
| 0205002 | ErrEmailAlreadyExists | 409 | Email already exists | 邮箱已被使用 |
| 0203001 | ErrUserDisabled | 403 | User disabled | 用户已禁用 |

### 5.2 调度服务 (03)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 0301001 | ErrTaskInvalidCron | 400 | Invalid cron expression | Cron 表达式无效 |
| 0301002 | ErrTaskInvalidParams | 400 | Invalid task parameters | 任务参数无效 |
| 0304001 | ErrTaskNotFound | 404 | Task not found | 任务不存在 |
| 0305001 | ErrTaskAlreadyExists | 409 | Task already exists | 任务已存在 |
| 0305002 | ErrTaskRunning | 409 | Task is running | 任务正在执行 |
| 0307001 | ErrTaskExecutionFailed | 500 | Task execution failed | 任务执行失败 |

## 6. 第三方服务错误码

### 6.1 支付服务 (90)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 9007001 | ErrPaymentFailed | 500 | Payment failed | 支付失败 |
| 9007002 | ErrPaymentRefundFailed | 500 | Refund failed | 退款失败 |
| 9011001 | ErrPaymentTimeout | 504 | Payment timeout | 支付超时 |

### 6.2 短信服务 (91)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 9107001 | ErrSMSFailed | 500 | SMS send failed | 短信发送失败 |
| 9106001 | ErrSMSLimitExceeded | 429 | SMS limit exceeded | 短信发送次数超限 |
| 9111001 | ErrSMSTimeout | 504 | SMS service timeout | 短信服务超时 |

### 6.3 邮件服务 (92)

| 错误码 | 常量名 | HTTP | 英文消息 | 中文消息 |
|-------|--------|------|---------|---------|
| 9207001 | ErrEmailFailed | 500 | Email send failed | 邮件发送失败 |
| 9211001 | ErrEmailTimeout | 504 | Email service timeout | 邮件服务超时 |

## 7. 错误响应格式

### 7.1 HTTP 响应格式

```json
{
    "code": 0001001,
    "message": "Invalid parameter: username is required",
    "request_id": "req-123456",
    "timestamp": 1699999999999
}
```

成功响应：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 1,
        "name": "example"
    },
    "request_id": "req-123456",
    "timestamp": 1699999999999
}
```

### 7.2 gRPC 错误格式

```protobuf
message ErrorInfo {
    int32 code = 1;           // 业务错误码
    string message = 2;        // 错误消息
    string reason = 3;         // 错误原因 (可选)
    map<string, string> metadata = 4; // 附加信息
}
```

### 7.3 多语言支持

通过请求头 `Accept-Language` 控制返回的错误消息语言：

- `en`, `en-US` → 英文消息
- `zh`, `zh-CN`, `zh_CN` → 中文消息

## 8. 使用规范

### 8.1 错误码申请流程

1. 确定错误所属的服务模块 (AA)
2. 确定错误类别 (BB)
3. 在该类别下申请下一个可用序列号 (CCC)
4. 在对应的错误码文件中定义并注册

### 8.2 命名规范

- 错误常量使用 `Err` 前缀
- 采用 PascalCase 命名
- 名称应清晰表达错误含义
- 业务错误需包含服务前缀

**正确示例：**
```go
ErrInvalidParam      // 通用参数错误
ErrUserNotFound      // 用户服务的用户不存在错误
ErrTaskExecutionFailed // 调度服务的任务执行失败
```

### 8.3 错误消息规范

- 英文消息：首字母大写，简洁明了
- 中文消息：直接描述错误，不使用敬语
- 不要在消息中包含敏感信息
- 可以使用占位符动态填充

### 8.4 向后兼容性

为保证向后兼容，旧的错误码（1000-5999）将通过别名映射到新的错误码体系，
在过渡期内同时支持两种错误码，过渡期结束后逐步废弃旧错误码。

## 9. 外部模块定义错误码

### 9.1 服务码申请与注册

每个业务服务需要申请一个唯一的服务码（20-79 范围），并在 `init()` 中注册：

```go
package myservice

import "github.com/kart-io/sentinel-x/pkg/errors"

// 申请的服务码
const ServiceMyService = 25

func init() {
    // 注册服务，防止服务码冲突
    errors.RegisterService(ServiceMyService, "my-service")
}
```

### 9.2 使用 Builder 模式定义错误码

推荐使用 Builder 模式定义错误码，支持链式调用：

```go
// 方式一：使用预设 Builder（推荐）
var ErrOrderNotFound = errors.NewNotFoundError(ServiceOrder, 1).
    Message("Order not found", "订单不存在").
    MustBuild()

var ErrOrderInvalidAmount = errors.NewRequestError(ServiceOrder, 1).
    Message("Invalid order amount", "订单金额无效").
    MustBuild()

// 方式二：使用完整 Builder（自定义 HTTP/gRPC 状态码）
var ErrOrderExpired = errors.NewBuilder(ServiceOrder, errors.CategoryConflict, 10).
    HTTP(http.StatusGone).           // 自定义 HTTP 状态码
    GRPC(codes.FailedPrecondition).  // 自定义 gRPC 状态码
    Message("Order has expired", "订单已过期").
    MustBuild()

// 方式三：使用快捷函数
var ErrOrderNotPaid = errors.NewConflictErr(ServiceOrder, 2,
    "Order not paid", "订单未支付")
```

### 9.3 预设 Builder 类型

| Builder 函数 | 类别 | HTTP | gRPC |
|-------------|------|------|------|
| `NewRequestError` | 请求错误 | 400 | InvalidArgument |
| `NewAuthError` | 认证错误 | 401 | Unauthenticated |
| `NewPermissionError` | 权限错误 | 403 | PermissionDenied |
| `NewNotFoundError` | 资源错误 | 404 | NotFound |
| `NewConflictError` | 冲突错误 | 409 | AlreadyExists |
| `NewRateLimitError` | 限流错误 | 429 | ResourceExhausted |
| `NewInternalError` | 内部错误 | 500 | Internal |
| `NewDatabaseError` | 数据库错误 | 500 | Internal |
| `NewCacheError` | 缓存错误 | 500 | Internal |
| `NewNetworkError` | 网络错误 | 503 | Unavailable |
| `NewTimeoutError` | 超时错误 | 504 | DeadlineExceeded |
| `NewConfigError` | 配置错误 | 500 | Internal |

### 9.4 外部错误码文件组织

建议每个服务在自己的模块中定义错误码：

```
myservice/
├── errors.go          # 错误码定义
├── service.go         # 业务逻辑
├── handler.go         # HTTP/gRPC 处理器
└── repository.go      # 数据访问
```

**errors.go 示例：**

```go
package myservice

import "github.com/kart-io/sentinel-x/pkg/errors"

const ServiceMyService = 25

func init() {
    errors.RegisterService(ServiceMyService, "my-service")
}

// Request Errors
var (
    ErrInvalidInput = errors.NewRequestErr(ServiceMyService, 1,
        "Invalid input", "输入无效")
)

// Resource Errors
var (
    ErrResourceNotFound = errors.NewNotFoundErr(ServiceMyService, 1,
        "Resource not found", "资源不存在")
)

// Conflict Errors
var (
    ErrResourceExists = errors.NewConflictErr(ServiceMyService, 1,
        "Resource already exists", "资源已存在")
)
```

### 9.5 错误码使用示例

```go
// Service 层
func (s *Service) GetResource(ctx context.Context, id string) (*Resource, error) {
    res, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, myservice.ErrResourceNotFound.WithMessagef("resource %s not found", id)
        }
        return nil, errors.ErrDatabase.WithCause(err)
    }
    return res, nil
}

// HTTP Handler
func (h *Handler) GetResource(c transport.Context) {
    id := c.Param("id")
    res, err := h.svc.GetResource(c.Request().Context(), id)
    if err != nil {
        response.FailWithErrno(c, errors.FromError(err))
        return
    }
    response.OK(c, res)
}

// gRPC Handler
func (s *GRPCServer) GetResource(ctx context.Context, req *pb.GetRequest) (*pb.Resource, error) {
    res, err := s.svc.GetResource(ctx, req.Id)
    if err != nil {
        errno := errors.FromError(err)
        return nil, status.Error(errno.GRPCCode, errno.MessageEN)
    }
    return toProto(res), nil
}
```

## 10. 附录

### 10.1 HTTP 状态码映射表

| 状态码 | 含义 | 对应类别 |
|-------|------|---------|
| 200 | OK | 成功 (00) |
| 400 | Bad Request | 请求错误 (01) |
| 401 | Unauthorized | 认证错误 (02) |
| 403 | Forbidden | 授权错误 (03) |
| 404 | Not Found | 资源错误 (04) |
| 408 | Request Timeout | 超时错误 (11) |
| 409 | Conflict | 冲突错误 (05) |
| 415 | Unsupported Media Type | 请求错误 (01) |
| 423 | Locked | 授权错误 (03) |
| 429 | Too Many Requests | 限流错误 (06) |
| 499 | Client Closed Request | 超时错误 (11) |
| 500 | Internal Server Error | 内部错误 (07-09, 12) |
| 501 | Not Implemented | 内部错误 (07) |
| 502 | Bad Gateway | 网络错误 (10) |
| 503 | Service Unavailable | 网络错误 (10) |
| 504 | Gateway Timeout | 超时错误 (11) |

### 10.2 gRPC 状态码映射表

| gRPC 状态码 | 含义 | 对应类别 |
|------------|------|---------|
| OK | 成功 | 成功 (00) |
| InvalidArgument | 参数无效 | 请求错误 (01) |
| Unauthenticated | 未认证 | 认证错误 (02) |
| PermissionDenied | 权限不足 | 授权错误 (03) |
| NotFound | 未找到 | 资源错误 (04) |
| AlreadyExists | 已存在 | 冲突错误 (05) |
| ResourceExhausted | 资源耗尽 | 限流错误 (06) |
| Internal | 内部错误 | 内部错误 (07-09, 12) |
| Unavailable | 不可用 | 网络错误 (10) |
| DeadlineExceeded | 超时 | 超时错误 (11) |
| Cancelled | 已取消 | 超时错误 (11) |
| Unimplemented | 未实现 | 内部错误 (07) |
