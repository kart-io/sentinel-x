# Security Headers Middleware

安全响应头中间件，用于保护 Web 应用免受常见的安全威胁，包括点击劫持、XSS 攻击、MIME 类型嗅探等。

## 功能特性

- **X-Frame-Options**: 防止点击劫持攻击
- **X-Content-Type-Options**: 防止 MIME 类型嗅探
- **X-XSS-Protection**: 启用浏览器 XSS 防护
- **Content-Security-Policy**: 防止各种注入攻击
- **Referrer-Policy**: 控制引用来源信息泄露
- **Strict-Transport-Security (HSTS)**: 强制使用 HTTPS 连接
- **独立函数设计**: 每个响应头设置独立，遵循单一职责原则

## 安装

```bash
go get github.com/kart-io/sentinel-x/pkg/infra/middleware
```

## 快速开始

### 使用默认配置

```go
import (
    "github.com/kart-io/sentinel-x/pkg/infra/middleware"
)

// 使用默认安全配置
router.Use(middleware.SecurityHeaders())
```

默认配置包括：

- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: default-src 'self'`
- `Referrer-Policy: strict-origin-when-cross-origin`
- HSTS: 默认禁用（需要显式启用）

### 使用自定义配置

```go
config := middleware.SecurityHeadersConfig{
    XFrameOptions:           "SAMEORIGIN",
    XContentTypeOptions:     "nosniff",
    XXSSProtection:          "1; mode=block",
    ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline'",
    ReferrerPolicy:          "no-referrer",
    StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
    EnableHSTS:              true,
}

router.Use(middleware.SecurityHeadersWithConfig(config))
```

## 配置说明

### SecurityHeadersConfig 结构

```go
type SecurityHeadersConfig struct {
    XFrameOptions           string  // X-Frame-Options 头
    XContentTypeOptions     string  // X-Content-Type-Options 头
    XXSSProtection          string  // X-XSS-Protection 头
    ContentSecurityPolicy   string  // Content-Security-Policy 头
    ReferrerPolicy          string  // Referrer-Policy 头
    StrictTransportSecurity string  // Strict-Transport-Security 头
    EnableHSTS              bool    // 是否启用 HSTS
}
```

### 各响应头说明

#### X-Frame-Options

防止点击劫持攻击。

- `DENY`: 完全禁止在框架中显示
- `SAMEORIGIN`: 只允许同源框架
- `ALLOW-FROM uri`: 允许指定的 URI

**推荐值**: `DENY` 或 `SAMEORIGIN`

#### X-Content-Type-Options

防止浏览器进行 MIME 类型嗅探。

- `nosniff`: 禁用 MIME 嗅探

**推荐值**: `nosniff`

#### X-XSS-Protection

启用浏览器内置的 XSS 过滤器。

- `0`: 禁用 XSS 过滤器
- `1`: 启用 XSS 过滤器
- `1; mode=block`: 启用并阻止渲染页面

**推荐值**: `1; mode=block`

#### Content-Security-Policy

定义内容安全策略，防止 XSS 和数据注入攻击。

**示例**:

```text
default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'
```

**推荐值**: 根据应用需求配置，越严格越好

#### Referrer-Policy

控制 Referer 头的发送策略。

**可选值**:

- `no-referrer`: 不发送 Referer
- `no-referrer-when-downgrade`: HTTPS 到 HTTP 时不发送
- `origin`: 只发送源信息
- `origin-when-cross-origin`: 跨域时只发送源
- `same-origin`: 仅同源发送
- `strict-origin`: 类似 origin，但更严格
- `strict-origin-when-cross-origin`: 默认值
- `unsafe-url`: 总是发送完整 URL

**推荐值**: `strict-origin-when-cross-origin` 或 `no-referrer`

#### Strict-Transport-Security (HSTS)

强制浏览器使用 HTTPS 连接。

**示例**:

```text
max-age=31536000; includeSubDomains; preload
```

**参数说明**:

- `max-age`: HSTS 策略有效期（秒）
- `includeSubDomains`: 包含所有子域名
- `preload`: 申请加入 HSTS 预加载列表

**重要提示**:

- 仅在 HTTPS 连接时设置
- 启用前确保网站完全支持 HTTPS
- `preload` 需要向 hstspreload.org 提交申请

## 使用场景

### 开发环境

```go
config := middleware.SecurityHeadersConfig{
    XFrameOptions:         "SAMEORIGIN",
    XContentTypeOptions:   "nosniff",
    XXSSProtection:        "1; mode=block",
    ContentSecurityPolicy: "default-src 'self' 'unsafe-inline' 'unsafe-eval'",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
    EnableHSTS:            false, // 开发环境禁用 HSTS
}
```

### 生产环境

```go
config := middleware.SecurityHeadersConfig{
    XFrameOptions:           "DENY",
    XContentTypeOptions:     "nosniff",
    XXSSProtection:          "1; mode=block",
    ContentSecurityPolicy:   "default-src 'self'; script-src 'self'; style-src 'self'",
    ReferrerPolicy:          "no-referrer",
    StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
    EnableHSTS:              true, // 生产环境启用 HSTS
}
```

### API 服务器

```go
config := middleware.SecurityHeadersConfig{
    XFrameOptions:           "DENY",
    XContentTypeOptions:     "nosniff",
    XXSSProtection:          "1; mode=block",
    ContentSecurityPolicy:   "default-src 'none'; frame-ancestors 'none'",
    ReferrerPolicy:          "no-referrer",
    StrictTransportSecurity: "max-age=31536000; includeSubDomains",
    EnableHSTS:              true,
}
```

## 独立函数

中间件提供以下独立函数，遵循单一职责原则：

### setXFrameOptions

设置 X-Frame-Options 响应头。

```go
func setXFrameOptions(c transport.Context, value string)
```

### setXContentTypeOptions

设置 X-Content-Type-Options 响应头。

```go
func setXContentTypeOptions(c transport.Context, value string)
```

### setXXSSProtection

设置 X-XSS-Protection 响应头。

```go
func setXXSSProtection(c transport.Context, value string)
```

### setContentSecurityPolicy

设置 Content-Security-Policy 响应头。

```go
func setContentSecurityPolicy(c transport.Context, value string)
```

### setReferrerPolicy

设置 Referrer-Policy 响应头。

```go
func setReferrerPolicy(c transport.Context, value string)
```

### setHSTS

设置 Strict-Transport-Security 响应头。

```go
func setHSTS(c transport.Context, value string, isHTTPS bool)
```

**注意**: 仅在 `isHTTPS` 为 `true` 时设置 HSTS 头。

### isHTTPSConnection

检查当前连接是否为 HTTPS。

```go
func isHTTPSConnection(c transport.Context) bool
```

支持以下检测方式：

- 直接 TLS 连接（`req.TLS != nil`）
- 反向代理场景（检查 `X-Forwarded-Proto` 头）

## 最佳实践

1. **始终启用基本安全头**: `X-Frame-Options`、`X-Content-Type-Options`、`X-XSS-Protection`
2. **根据应用需求配置 CSP**: 从严格策略开始，逐步放宽
3. **生产环境启用 HSTS**: 确保网站完全支持 HTTPS 后再启用
4. **测试安全配置**: 使用工具如 [Security Headers](https://securityheaders.com/) 验证配置
5. **定期更新**: 关注安全最佳实践的变化

## 安全警告

- **HSTS 谨慎启用**: 一旦启用，在 `max-age` 期间无法降级到 HTTP
- **CSP 逐步部署**: 建议先使用 `Content-Security-Policy-Report-Only` 头测试
- **反向代理配置**: 确保 `X-Forwarded-Proto` 头正确设置

## 测试

运行单元测试：

```bash
go test -v ./pkg/infra/middleware -run TestSecurityHeaders
```

运行覆盖率测试：

```bash
go test -cover ./pkg/infra/middleware
```

## 示例代码

查看 `security_headers_example_test.go` 获取更多示例。

## 参考资料

- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [MDN Web Security](https://developer.mozilla.org/en-US/docs/Web/Security)
- [HSTS Preload List](https://hstspreload.org/)
- [Content Security Policy Reference](https://content-security-policy.com/)

## 许可证

本项目遵循 MIT 许可证。
