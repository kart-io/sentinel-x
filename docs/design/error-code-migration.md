# 错误码迁移对照表

## 概述

本文档提供旧版错误码到新版错误码的完整映射关系，用于指导系统升级和代码迁移。

## 迁移说明

### 新版错误码格式

新版错误码采用 **7位数字** 格式 `AABBCCC`：
- `AA`: 服务/模块代码 (00-99)
- `BB`: 类别代码 (00-99)
- `CCC`: 序列号 (000-999)

### 旧版错误码范围

旧版错误码采用 **4位数字** 格式：
- `0`: 成功
- `1000-1999`: 客户端错误
- `2000-2999`: 认证错误
- `3000-3999`: 授权错误
- `4000-4999`: 资源不存在错误
- `5000-5999`: 服务器错误

## 错误码对照表

### 成功响应

| 旧错误码 | 旧常量名 | 新错误码 | 新常量名 | 说明 |
|---------|---------|---------|---------|------|
| 0 | CodeSuccess | 0 | OK | 成功 |

### 客户端错误 (1000-1999 → 0001xxx)

| 旧错误码 | 旧常量名 | 新错误码 | 新常量名 | HTTP | 说明 |
|---------|---------|---------|---------|------|------|
| 1000 | CodeBadRequest | 1000 | ErrBadRequest | 400 | 请求错误 |
| 1001 | CodeInvalidParam | 1001 | ErrInvalidParam | 400 | 参数无效 |
| 1002 | CodeMissingParam | 1002 | ErrMissingParam | 400 | 缺少必需参数 |
| 1003 | CodeInvalidFormat | 1003 | ErrInvalidFormat | 400 | 格式无效 |
| 1004 | CodeValidationFailed | 1004 | ErrValidationFailed | 400 | 验证失败 |
| 1005 | CodeTooManyRequests | 6000 | ErrTooManyRequests | 429 | 请求过于频繁 |

### 认证错误 (2000-2999 → 0002xxx)

| 旧错误码 | 旧常量名 | 新错误码 | 新常量名 | HTTP | 说明 |
|---------|---------|---------|---------|------|------|
| 2000 | CodeUnauthorized | 2000 | ErrUnauthorized | 401 | 未认证 |
| 2001 | CodeInvalidToken | 2001 | ErrInvalidToken | 401 | 令牌无效 |
| 2002 | CodeTokenExpired | 2002 | ErrTokenExpired | 401 | 令牌已过期 |
| 2003 | CodeInvalidCredential | 2003 | ErrInvalidCredentials | 401 | 凭证无效 |

### 授权错误 (3000-3999 → 0003xxx)

| 旧错误码 | 旧常量名 | 新错误码 | 新常量名 | HTTP | 说明 |
|---------|---------|---------|---------|------|------|
| 3000 | CodeForbidden | 3000 | ErrForbidden | 403 | 禁止访问 |
| 3001 | CodeNoPermission | 3001 | ErrNoPermission | 403 | 无权限 |
| 3002 | CodeResourceLocked | 3002 | ErrResourceLocked | 423 | 资源已锁定 |

### 资源不存在错误 (4000-4999 → 0004xxx)

| 旧错误码 | 旧常量名 | 新错误码 | 新常量名 | HTTP | 说明 |
|---------|---------|---------|---------|------|------|
| 4000 | CodeNotFound | 4000 | ErrNotFound | 404 | 资源不存在 |
| 4001 | CodeUserNotFound | 4001 | ErrUserNotFound | 404 | 用户不存在 |
| 4002 | CodeRecordNotFound | 4002 | ErrRecordNotFound | 404 | 记录不存在 |

### 服务器错误 (5000-5999 → 0007xxx/0008xxx/0009xxx)

| 旧错误码 | 旧常量名 | 新错误码 | 新常量名 | HTTP | 说明 |
|---------|---------|---------|---------|------|------|
| 5000 | CodeInternalError | 7000 | ErrInternal | 500 | 服务器内部错误 |
| 5001 | CodeDatabaseError | 8000 | ErrDatabase | 500 | 数据库错误 |
| 5002 | CodeCacheError | 9000 | ErrCache | 500 | 缓存错误 |
| 5003 | CodeExternalService | 10001 | ErrServiceUnavailable | 503 | 服务不可用 |
| 5004 | CodeTimeout | 11000 | ErrTimeout | 504 | 操作超时 |

## 新增错误码

以下是新系统新增的错误码类型：

### 请求错误扩展 (0001xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 1005 | ErrRequestTooLarge | 413 | 请求体过大 |
| 1006 | ErrUnsupportedMediaType | 415 | 不支持的媒体类型 |

### 认证错误扩展 (0002xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 2004 | ErrTokenRevoked | 401 | 令牌已撤销 |
| 2005 | ErrSessionExpired | 401 | 会话已过期 |

### 授权错误扩展 (0003xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 3003 | ErrAccountDisabled | 403 | 账号已禁用 |
| 3004 | ErrIPBlocked | 403 | IP 已被封禁 |

### 资源错误扩展 (0004xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 4003 | ErrFileNotFound | 404 | 文件不存在 |
| 4004 | ErrRouteNotFound | 404 | 路由不存在 |

### 冲突错误 (0005xxx) - 新增类别

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 5000 | ErrConflict | 409 | 资源冲突 |
| 5001 | ErrAlreadyExists | 409 | 资源已存在 |
| 5002 | ErrDuplicateKey | 409 | 键值重复 |
| 5003 | ErrVersionConflict | 409 | 版本冲突 |

### 限流错误 (0006xxx) - 新增类别

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 6000 | ErrTooManyRequests | 429 | 请求过于频繁 |
| 6001 | ErrRateLimitExceeded | 429 | 超出速率限制 |
| 6002 | ErrQuotaExceeded | 429 | 配额已用尽 |

### 内部错误扩展 (0007xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 7001 | ErrUnknown | 500 | 未知错误 |
| 7002 | ErrPanic | 500 | 服务崩溃 |
| 7003 | ErrNotImplemented | 501 | 功能未实现 |

### 数据库错误扩展 (0008xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 8001 | ErrDBConnection | 500 | 数据库连接失败 |
| 8002 | ErrDBQuery | 500 | 数据库查询失败 |
| 8003 | ErrDBTransaction | 500 | 数据库事务失败 |
| 8004 | ErrDBDeadlock | 500 | 数据库死锁 |

### 缓存错误扩展 (0009xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 9001 | ErrCacheConnection | 500 | 缓存连接失败 |
| 9002 | ErrCacheMiss | 500 | 缓存未命中 |
| 9003 | ErrCacheExpired | 500 | 缓存已过期 |

### 网络错误 (0010xxx) - 新增类别

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 10000 | ErrNetwork | 502 | 网络错误 |
| 10001 | ErrServiceUnavailable | 503 | 服务不可用 |
| 10002 | ErrConnectionRefused | 502 | 连接被拒绝 |
| 10003 | ErrDNSResolution | 502 | DNS 解析失败 |

### 超时错误扩展 (0011xxx)

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 11001 | ErrRequestTimeout | 408 | 请求超时 |
| 11002 | ErrGatewayTimeout | 504 | 网关超时 |
| 11003 | ErrContextCanceled | 499 | 上下文已取消 |

### 配置错误 (0012xxx) - 新增类别

| 错误码 | 常量名 | HTTP | 说明 |
|-------|--------|------|------|
| 12000 | ErrConfig | 500 | 配置错误 |
| 12001 | ErrConfigNotFound | 500 | 配置不存在 |
| 12002 | ErrConfigInvalid | 500 | 配置无效 |

## 代码迁移指南

### 1. 导入包变更

**旧代码：**
```go
import "github.com/kart-io/sentinel-x/pkg/response"

// 使用旧错误码
return response.ErrInvalidParam.WithMessage("invalid username")
```

**新代码：**
```go
import "github.com/kart-io/sentinel-x/pkg/errors"

// 使用新错误码
return errors.ErrInvalidParam.WithMessage("invalid username")
```

### 2. 错误码常量迁移

**旧代码：**
```go
if response.IsError(err, response.CodeInvalidParam) {
    // 处理参数错误
}
```

**新代码：**
```go
if errors.IsCode(err, errors.ErrInvalidParam.Code) {
    // 处理参数错误
}
```

### 3. 兼容层使用

在过渡期间，可以使用兼容层函数：

```go
import "github.com/kart-io/sentinel-x/pkg/errors"

// 将旧错误码转换为新错误码
newCode := errors.LegacyToNewCode(1001)

// 将新错误码转换为旧错误码
legacyCode := errors.NewToLegacyCode(1001)

// 检查是否为旧错误码
if errors.IsLegacyCode(code) {
    // 处理旧错误码
}

// 从旧错误码获取 Errno
errno := errors.FromLegacyCode(1001)
```

### 4. HTTP Handler 迁移

**旧代码：**
```go
func handler(c transport.Context) {
    if err != nil {
        response.Fail(c, response.ErrInvalidParam.WithMessage("invalid"))
        return
    }
    response.OK(c, data)
}
```

**新代码（推荐）：**
```go
func handler(c transport.Context) {
    if err != nil {
        response.FailWithErrno(c, errors.ErrInvalidParam.WithMessage("invalid"))
        return
    }
    response.OK(c, data)
}
```

## 迁移时间表

| 阶段 | 版本 | 时间 | 说明 |
|-----|------|------|------|
| 引入 | v1.x | 当前 | 引入新错误码系统，两种系统并存 |
| 过渡 | v2.0 | +6个月 | 旧错误码标记为 deprecated，日志警告 |
| 移除 | v3.0 | +12个月 | 移除旧错误码，仅保留兼容层 |

## 注意事项

1. **向后兼容**：新版本保持对旧错误码的识别能力
2. **渐进迁移**：可以逐步迁移，不需要一次性完成
3. **测试覆盖**：迁移前确保有足够的测试覆盖
4. **文档更新**：API 文档需同步更新错误码说明
5. **客户端通知**：如果是对外 API，需提前通知客户端错误码变更
