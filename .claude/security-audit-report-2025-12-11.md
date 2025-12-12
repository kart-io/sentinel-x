# Sentinel-X 安全审查报告

**审查日期**: 2025-12-11
**审查人**: Security Auditor (Claude)
**项目**: Sentinel-X 微服务平台
**审查范围**: 认证授权系统、JWT 实现、输入验证、数据库安全、敏感数据处理

---

## 执行摘要

本次安全审查针对 Sentinel-X 项目的核心安全组件进行了全面分析，重点关注了 JWT 认证实现、用户认证流程、授权机制、SQL 注入防护和敏感数据处理。总体而言，项目在安全设计上遵循了多项最佳实践，但仍存在若干中高风险漏洞需要立即修复。

**关键发现统计**：
- 严重漏洞（Critical）: 1 个
- 高危漏洞（High）: 2 个
- 中危漏洞（Medium）: 4 个
- 低危漏洞（Low）: 3 个
- 安全最佳实践建议: 5 个

**整体安全评级**: B- (需要改进)

---

## 1. 严重漏洞（Critical Severity）

### 🚨 CRITICAL-001: 生产环境使用硬编码 JWT 密钥

**文件位置**: `/configs/user-center.yaml:155`

**漏洞描述**:
配置文件中硬编码了 JWT 签名密钥，且该密钥仅 64 字符，处于安全边界最低要求。虽然配置中有注释要求通过环境变量设置，但提供了默认值：

```yaml
jwt:
  disable-auth: true  # 开发环境禁用认证
  key: "your-super-secret-key-change-in-production-minimum-64-characters-required-here"
```

**风险评估**:
- **CVSS 评分**: 9.1 (严重)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 低
- **影响**: 完全破坏认证系统，攻击者可伪造任意用户 Token

**攻击场景**:
1. 攻击者从公开的代码仓库获取默认密钥
2. 使用该密钥伪造 admin 用户的 JWT token
3. 绕过所有认证检查，获取管理员权限
4. 执行特权操作，包括创建后门账户、窃取数据

**修复建议**:

**立即行动（必须）**:
```yaml
# user-center.yaml
jwt:
  disable-auth: false
  # 移除硬编码密钥，强制使用环境变量
  # key: ""  # 不提供默认值
  signing-method: "HS512"  # 升级到更强的算法
  expired: "2h"
  max-refresh: "24h"
```

**环境变量设置**:
```bash
# 生成强随机密钥（128 字符 = 1024 bits）
export USER_CENTER_JWT_KEY=$(openssl rand -base64 96)

# 或使用推荐的 256 字符密钥
export USER_CENTER_JWT_KEY=$(openssl rand -base64 192)
```

**代码增强（推荐）**:
在 `pkg/options/auth/jwt/options.go` 中添加启动时检查：

```go
func (o *Options) Validate() error {
    if o.DisableAuth {
        return nil
    }

    // 禁止使用测试/默认密钥
    dangerousKeys := []string{
        "your-super-secret-key",
        "change-in-production",
        "test-key",
        "secret",
    }

    keyLower := strings.ToLower(o.Key)
    for _, dangerous := range dangerousKeys {
        if strings.Contains(keyLower, dangerous) {
            return fmt.Errorf("SECURITY: detected test/default JWT key, refusing to start. "+
                "Set USER_CENTER_JWT_KEY environment variable with a strong random key")
        }
    }

    // 现有验证逻辑...
}
```

**验证方法**:
```bash
# 启动时应该失败（如果使用默认密钥）
make run-user-center

# 正确设置后应该成功启动
export USER_CENTER_JWT_KEY=$(openssl rand -base64 192)
make run-user-center
```

**参考标准**:
- OWASP Top 10 2021: A07:2021 - Identification and Authentication Failures
- CWE-798: Use of Hard-coded Credentials
- NIST SP 800-57: Recommendation for Key Management (要求 HMAC 密钥至少 112 bits 熵)

---

## 2. 高危漏洞（High Severity）

### 🔴 HIGH-001: Token 刷新机制存在会话固定风险

**文件位置**: `/pkg/security/auth/jwt/jwt.go:309-370`

**漏洞描述**:
虽然代码在第 367 行的注释中提到"不传递 TokenID 以生成新 ID"，但实际实现中存在竞态条件风险。如果在 `revokeOldToken()` 失败时（第 342-354 行），旧 token 未被撤销，但新 token 已经生成，攻击者可以同时使用新旧两个 token。

```go
func (j *JWT) Refresh(ctx context.Context, tokenString string) (auth.Token, error) {
    // 1. 验证 token
    claims, err := j.verifyForRefresh(ctx, tokenString)
    if err != nil {
        return nil, err
    }

    // 2. 检查刷新窗口
    if err := j.checkRefreshWindow(claims); err != nil {
        return nil, err
    }

    // 3. 撤销旧 token（非阻塞 - 失败只记录警告）
    j.revokeOldToken(ctx, tokenString)  // ⚠️ 失败不会阻止刷新

    // 4. 生成新 token
    return j.generateRefreshToken(ctx, claims)  // ✓ 但已经生成新 token
}
```

**风险评估**:
- **CVSS 评分**: 7.5 (高危)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 中等
- **影响**: 会话劫持、权限维持

**攻击场景**:
1. 攻击者窃取用户的 JWT token（例如通过 XSS）
2. 在合法用户刷新 token 时，撤销操作因 Redis 故障失败
3. 攻击者持有的旧 token 仍然有效
4. 攻击者和合法用户同时拥有有效 token，导致并发会话

**修复建议**:

**核心原则**: 先生成新 token 并存储，再撤销旧 token，确保原子性。

```go
// 改进的 Refresh 实现
func (j *JWT) Refresh(ctx context.Context, tokenString string) (auth.Token, error) {
    // 1. 验证 token
    claims, err := j.verifyForRefresh(ctx, tokenString)
    if err != nil {
        return nil, err
    }

    // 2. 检查刷新窗口
    if err := j.checkRefreshWindow(claims); err != nil {
        return nil, err
    }

    // 3. 原子性检查：确保 store 可用
    if j.store != nil {
        // 预检查：尝试访问 store
        _, err := j.store.IsRevoked(ctx, tokenString)
        if err != nil {
            return nil, errors.ErrInternal.WithCause(err).
                WithMessage("token store unavailable, cannot safely refresh")
        }
    }

    // 4. 生成新 token（注意：新 token 有新的 ID）
    newToken, err := j.generateRefreshToken(ctx, claims)
    if err != nil {
        return nil, err
    }

    // 5. 撤销旧 token（现在必须成功）
    if j.store != nil {
        if err := j.Revoke(ctx, tokenString); err != nil {
            // 撤销失败是严重错误，记录并返回错误
            logger.Errorw("SECURITY: failed to revoke old token during refresh",
                "error", err,
                "subject", claims.Subject,
                "tokenPrefix", tokenPrefix(tokenString))

            // 理想情况：应该撤销新生成的 token
            // 但由于新 token 还未返回给客户端，直接返回错误即可
            return nil, errors.ErrInternal.
                WithMessage("failed to complete token refresh securely")
        }
    }

    // 6. 安全审计日志
    logger.Infow("token refresh successful",
        "subject", claims.Subject,
        "old_token_prefix", tokenPrefix(tokenString),
        "new_token_id", newToken.GetAccessToken()[:16])

    return newToken, nil
}
```

**验证方法**:
```go
// 测试用例：模拟 store 撤销失败
func TestRefresh_StoreFailure(t *testing.T) {
    mockStore := &FailingStore{} // 模拟撤销总是失败
    jwtAuth, _ := jwt.New(
        jwt.WithKey(testKey),
        jwt.WithStore(mockStore),
    )

    oldToken, _ := jwtAuth.Sign(ctx, "user123")

    // 应该返回错误，而非生成新 token
    newToken, err := jwtAuth.Refresh(ctx, oldToken.GetAccessToken())

    assert.Error(t, err)
    assert.Nil(t, newToken)
    assert.Contains(t, err.Error(), "cannot safely refresh")
}
```

**参考标准**:
- OWASP ASVS 4.0: V3.2.3 - Session tokens invalidated on logout
- CWE-384: Session Fixation

---

### 🔴 HIGH-002: Token 撤销 TTL 使用 MaxRefresh 而非 ExpiresAt 可能导致存储耗尽

**文件位置**: `/pkg/security/auth/jwt/jwt.go:485-496`

**漏洞描述**:
撤销 token 时使用 `MaxRefresh` 时间作为 TTL，而非实际的 `ExpiresAt`。如果 MaxRefresh 设置为较长时间（例如 30 天），而 token 的有效期只有 2 小时，会导致已过期的 token 在 Redis 中保留 28 天。

```go
func (j *JWT) Revoke(ctx context.Context, tokenString string) error {
    // ...省略验证代码...

    // 计算 TTL 直到 MaxRefresh 时间（而非 ExpiresAt）
    issuedAt := claims.IssuedAt.Time
    maxRefreshTime := issuedAt.Add(j.opts.MaxRefresh)  // ⚠️ 可能是 30 天
    ttl := time.Until(maxRefreshTime)

    // 如果 token 实际上 2 小时后就过期了，这里仍然存储 30 天
    return j.store.Revoke(ctx, tokenString, ttl)
}
```

**风险评估**:
- **CVSS 评分**: 6.5 (中高危)
- **攻击向量**: 本地/网络
- **攻击复杂度**: 低
- **影响**: 拒绝服务（存储耗尽）、性能降级

**攻击场景**:
1. 攻击者发起大量登录和注销操作
2. 每次注销都在 Redis 中存储一个 30 天 TTL 的记录
3. 短时间内积累数百万条撤销记录
4. Redis 内存耗尽，导致服务不可用
5. 或者 Redis 性能严重下降，影响所有依赖 Redis 的服务

**修复建议**:

**方案 1: 使用 ExpiresAt 和 MaxRefresh 中的较小值**

```go
func (j *JWT) Revoke(ctx context.Context, tokenString string) error {
    // ...省略前面的验证代码...

    // 计算 token 的实际过期时间和 MaxRefresh 时间
    expiresAt := claims.ExpiresAt.Time
    issuedAt := claims.IssuedAt.Time
    maxRefreshTime := issuedAt.Add(j.opts.MaxRefresh)

    // 使用两者中较早的时间作为 TTL
    var ttl time.Duration
    if expiresAt.Before(maxRefreshTime) {
        // Token 已过期或即将过期，使用 ExpiresAt
        ttl = time.Until(expiresAt)
        logger.Debugw("using token expiration for revocation TTL",
            "token_id", claims.ID,
            "ttl_seconds", int(ttl.Seconds()))
    } else {
        // Token 仍可刷新，使用 MaxRefresh
        ttl = time.Until(maxRefreshTime)
        logger.Debugw("using max refresh time for revocation TTL",
            "token_id", claims.ID,
            "ttl_seconds", int(ttl.Seconds()))
    }

    // 如果已经过期，不需要存储
    if ttl <= 0 {
        logger.Debugw("token already expired, skipping revocation storage",
            "token_id", claims.ID)
        return nil
    }

    return j.store.Revoke(ctx, tokenString, ttl)
}
```

**方案 2: 添加配置选项控制行为**

```go
// Options 中添加字段
type Options struct {
    // ...现有字段...

    // RevokeUseExpiresAt 控制撤销 TTL 的计算方式
    // true: 使用 token 的 ExpiresAt（推荐，节省存储）
    // false: 使用 MaxRefresh（当前行为，更安全但占用更多存储）
    RevokeUseExpiresAt bool `json:"revoke-use-expires-at" mapstructure:"revoke-use-expires-at"`
}
```

**监控和告警**:
```go
// 在 RedisStore.Revoke 中添加监控
func (s *RedisStore) Revoke(ctx context.Context, token string, expiration time.Duration) error {
    key := s.prefix + token

    // 记录长期撤销条目（超过 7 天）
    if expiration > 7*24*time.Hour {
        metrics.IncrementCounter("jwt.revoke.long_ttl_count")
        logger.Warnw("storing token revocation with long TTL",
            "ttl_hours", expiration.Hours(),
            "key_prefix", s.prefix)
    }

    return s.client.Client().Set(ctx, key, "revoked", expiration).Err()
}
```

**容量规划**:
假设每个 token 占用 1 KB（包括 key + value + Redis 开销）：
- 当前实现：1000 用户/天 × 30 天 = 30,000 条记录 ≈ 30 MB
- 优化后：1000 用户/天 × 2 小时 = 约 83 条记录 ≈ 83 KB

**参考标准**:
- OWASP ASVS 4.0: V3.3.3 - Token revocation efficient
- CWE-400: Uncontrolled Resource Consumption

---

## 3. 中危漏洞（Medium Severity）

### 🟠 MEDIUM-001: 用户密码强度未验证

**文件位置**:
- `/internal/user-center/biz/auth.go:86`
- `/internal/user-center/biz/user.go:25`
- `/internal/model/auth.go:19`

**漏洞描述**:
注册和创建用户时，仅验证密码字段为 `required`，未检查密码强度，允许用户使用弱密码如 "123456", "password" 等。

```go
// RegisterRequest - 仅验证必填，无强度要求
type RegisterRequest struct {
    Username string `json:"username" form:"username" validate:"required"`
    Password string `json:"password" form:"password" validate:"required"`  // ⚠️
    Email    string `json:"email" form:"email" validate:"required,email"`
}
```

**风险评估**:
- **CVSS 评分**: 5.3 (中危)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 低
- **影响**: 账户接管、暴力破解成功率高

**攻击场景**:
1. 用户注册时使用弱密码 "123456"
2. 攻击者通过暴力破解或字典攻击获取密码
3. 使用窃取的凭据登录系统
4. 访问用户的敏感数据或执行恶意操作

**修复建议**:

**步骤 1: 添加自定义密码验证器**

```go
// pkg/utils/validator/custom_rules.go
func (v *Validator) registerCustomRules() {
    // 注册密码强度验证器
    _ = v.RegisterValidationWithTranslation(
        "password_strong",
        validatePasswordStrength,
        map[string]string{
            LangEN: "password must be at least 8 characters with uppercase, lowercase, digit and special character",
            LangZH: "密码必须至少 8 个字符，包含大小写字母、数字和特殊字符",
        },
    )
}

// validatePasswordStrength 验证密码强度
func validatePasswordStrength(fl validator.FieldLevel) bool {
    password := fl.Field().String()

    // 最小长度 8 字符
    if len(password) < 8 {
        return false
    }

    // 最大长度 128 字符（防止 DoS）
    if len(password) > 128 {
        return false
    }

    var (
        hasUpper   bool
        hasLower   bool
        hasDigit   bool
        hasSpecial bool
    )

    for _, char := range password {
        switch {
        case unicode.IsUpper(char):
            hasUpper = true
        case unicode.IsLower(char):
            hasLower = true
        case unicode.IsDigit(char):
            hasDigit = true
        case unicode.IsPunct(char) || unicode.IsSymbol(char):
            hasSpecial = true
        }
    }

    // 要求至少满足 3 种字符类型
    typesCount := 0
    if hasUpper { typesCount++ }
    if hasLower { typesCount++ }
    if hasDigit { typesCount++ }
    if hasSpecial { typesCount++ }

    return typesCount >= 3
}
```

**步骤 2: 更新模型验证规则**

```go
// internal/model/auth.go
type RegisterRequest struct {
    Username string `json:"username" form:"username" validate:"required,min=3,max=32,alphanum"`
    Password string `json:"password" form:"password" validate:"required,password_strong"`
    Email    string `json:"email" form:"email" validate:"required,email"`
}

// 可选：添加密码确认字段
type RegisterRequest struct {
    Username        string `json:"username" form:"username" validate:"required,min=3,max=32,alphanum"`
    Password        string `json:"password" form:"password" validate:"required,password_strong"`
    PasswordConfirm string `json:"password_confirm" form:"password_confirm" validate:"required,eqfield=Password"`
    Email           string `json:"email" form:"email" validate:"required,email"`
}
```

**步骤 3: 添加常见密码黑名单检查**

```go
// pkg/utils/validator/password_blacklist.go
var commonPasswords = map[string]bool{
    "123456":    true,
    "password":  true,
    "12345678":  true,
    "qwerty":    true,
    "123456789": true,
    "12345":     true,
    "1234":      true,
    "111111":    true,
    "1234567":   true,
    "dragon":    true,
    "123123":    true,
    "baseball":  true,
    "abc123":    true,
    "football":  true,
    "monkey":    true,
    "letmein":   true,
    "696969":    true,
    "shadow":    true,
    "master":    true,
    "666666":    true,
    // ...添加更多常见密码
}

func validatePasswordNotCommon(fl validator.FieldLevel) bool {
    password := strings.ToLower(fl.Field().String())
    return !commonPasswords[password]
}
```

**步骤 4: 添加配置选项**

```yaml
# configs/user-center.yaml
security:
  password:
    min-length: 8
    max-length: 128
    require-uppercase: true
    require-lowercase: true
    require-digit: true
    require-special: true
    min-types-required: 3  # 至少满足 3 种字符类型
    check-common-passwords: true
```

**验证方法**:
```bash
# 测试弱密码被拒绝
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "123456",
    "email": "test@example.com"
  }'
# 应该返回: {"error": "密码必须至少 8 个字符，包含大小写字母、数字和特殊字符"}

# 测试强密码被接受
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "MyP@ssw0rd2025!",
    "email": "test@example.com"
  }'
# 应该返回: {"message": "user registered"}
```

**参考标准**:
- OWASP ASVS 4.0: V2.1.1 - Password policy
- NIST SP 800-63B: Password strength requirements
- CWE-521: Weak Password Requirements

---

### 🟠 MEDIUM-002: 缺少账户锁定机制防止暴力破解

**文件位置**: `/internal/user-center/biz/auth.go:31-60`

**漏洞描述**:
登录逻辑中没有实现失败尝试限制或账户锁定机制，攻击者可以无限次尝试密码，易受暴力破解攻击。

```go
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
    // 获取用户信息
    user, err := s.store.Users().Get(ctx, req.Username)
    if err != nil {
        return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
    }

    // 验证密码 - ⚠️ 无失败次数限制
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
    }

    // ...省略 token 生成...
}
```

**风险评估**:
- **CVSS 评分**: 5.9 (中危)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 低
- **影响**: 账户接管

**攻击场景**:
1. 攻击者使用自动化工具对目标账户进行暴力破解
2. 每秒尝试 100 个密码（常见密码字典）
3. 在几小时内遍历数千个常见密码
4. 成功获取弱密码用户的账户权限

**修复建议**:

**方案 1: Redis-based 登录失败跟踪**

```go
// internal/user-center/biz/auth_lockout.go
package biz

import (
    "context"
    "fmt"
    "time"

    "github.com/kart-io/sentinel-x/pkg/component/redis"
)

const (
    // 失败次数阈值
    MaxLoginAttempts = 5
    // 锁定时间
    LockoutDuration = 15 * time.Minute
    // 失败记录过期时间
    AttemptWindow = 1 * time.Hour
)

type LoginLockout struct {
    redis *redis.Client
}

func NewLoginLockout(redis *redis.Client) *LoginLockout {
    return &LoginLockout{redis: redis}
}

// CheckAndRecordFailure 检查并记录失败尝试
func (l *LoginLockout) CheckAndRecordFailure(ctx context.Context, username string) error {
    key := fmt.Sprintf("login:attempts:%s", username)
    lockKey := fmt.Sprintf("login:locked:%s", username)

    // 检查账户是否已锁定
    locked, err := l.redis.Client().Exists(ctx, lockKey).Result()
    if err != nil {
        return fmt.Errorf("failed to check lockout status: %w", err)
    }

    if locked > 0 {
        // 获取剩余锁定时间
        ttl, _ := l.redis.Client().TTL(ctx, lockKey).Result()
        return errors.ErrAccountLocked.WithMessage(
            fmt.Sprintf("账户已被锁定，请在 %d 分钟后重试", int(ttl.Minutes())))
    }

    // 增加失败次数
    attempts, err := l.redis.Client().Incr(ctx, key).Result()
    if err != nil {
        return fmt.Errorf("failed to record attempt: %w", err)
    }

    // 设置失败记录过期时间
    if attempts == 1 {
        l.redis.Client().Expire(ctx, key, AttemptWindow)
    }

    // 检查是否达到锁定阈值
    if attempts >= MaxLoginAttempts {
        // 锁定账户
        l.redis.Client().Set(ctx, lockKey, "locked", LockoutDuration)

        // 记录安全事件
        logger.Warnw("account locked due to failed login attempts",
            "username", username,
            "attempts", attempts,
            "lockout_duration_minutes", LockoutDuration.Minutes())

        return errors.ErrAccountLocked.WithMessage(
            fmt.Sprintf("登录失败次数过多，账户已被锁定 %d 分钟", int(LockoutDuration.Minutes())))
    }

    // 返回剩余尝试次数
    remaining := MaxLoginAttempts - int(attempts)
    return errors.ErrUnauthorized.WithMessage(
        fmt.Sprintf("无效的用户名或密码，剩余尝试次数：%d", remaining))
}

// ClearFailureRecord 清除失败记录（登录成功时调用）
func (l *LoginLockout) ClearFailureRecord(ctx context.Context, username string) {
    key := fmt.Sprintf("login:attempts:%s", username)
    l.redis.Client().Del(ctx, key)
}
```

**更新 Login 方法**:

```go
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
    // 1. 检查账户锁定状态
    if err := s.lockout.CheckLockoutStatus(ctx, req.Username); err != nil {
        return nil, err
    }

    // 2. 获取用户信息
    user, err := s.store.Users().Get(ctx, req.Username)
    if err != nil {
        // 记录失败尝试（即使用户不存在，也记录 IP 的尝试次数）
        _ = s.lockout.CheckAndRecordFailure(ctx, req.Username)
        return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
    }

    // 3. 验证密码
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        // 记录失败尝试并可能锁定账户
        lockErr := s.lockout.CheckAndRecordFailure(ctx, req.Username)
        if lockErr != nil {
            return nil, lockErr
        }
        return nil, errors.ErrUnauthorized.WithMessage("无效的用户名或密码")
    }

    // 4. 检查用户状态
    if user.Status == 0 {
        return nil, errors.ErrAccountDisabled.WithMessage("账号已被禁用")
    }

    // 5. 清除失败记录
    s.lockout.ClearFailureRecord(ctx, req.Username)

    // 6. 生成访问令牌
    token, err := s.jwtAuth.Sign(ctx, req.Username, auth.WithExtra(map[string]interface{}{
        "id": user.ID,
    }))
    if err != nil {
        return nil, errors.ErrInternal.WithCause(err)
    }

    // 7. 记录成功登录
    logger.Infow("user login successful",
        "username", req.Username,
        "user_id", user.ID)

    return &model.LoginResponse{
        Token:     token.GetAccessToken(),
        ExpiresIn: token.GetExpiresAt(),
        UserID:    user.ID,
    }, nil
}
```

**方案 2: 添加 CAPTCHA 验证**

```go
// 在失败 3 次后要求 CAPTCHA
type LoginRequest struct {
    Username string `json:"username" validate:"required"`
    Password string `json:"password" validate:"required"`
    Captcha  string `json:"captcha"`  // CAPTCHA 响应
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
    attempts, _ := s.getFailureAttempts(ctx, req.Username)

    // 失败 3 次后要求 CAPTCHA
    if attempts >= 3 && req.Captcha == "" {
        return nil, errors.ErrBadRequest.WithMessage("请完成验证码验证")
    }

    if attempts >= 3 {
        // 验证 CAPTCHA
        if !s.verifyCaptcha(ctx, req.Captcha) {
            return nil, errors.ErrBadRequest.WithMessage("验证码错误")
        }
    }

    // ...继续原有登录逻辑...
}
```

**监控和告警**:
```yaml
# 告警规则
alerts:
  - name: BruteForceAttack
    condition: sum(rate(login_failures[5m])) > 100
    severity: high
    message: "检测到暴力破解攻击，5 分钟内超过 100 次失败登录"

  - name: AccountLockoutSpike
    condition: sum(rate(account_lockouts[1h])) > 10
    severity: medium
    message: "账户锁定数量异常，1 小时内超过 10 个账户被锁定"
```

**参考标准**:
- OWASP ASVS 4.0: V2.2.1 - Anti-automation controls
- CWE-307: Improper Restriction of Excessive Authentication Attempts

---

### 🟠 MEDIUM-003: Token 撤销在分布式环境中的一致性问题

**文件位置**: `/pkg/security/auth/jwt/store.go` (MemoryStore)

**漏洞描述**:
项目提供了 `MemoryStore` 实现，但在分布式部署中（多个实例），各实例的内存存储不同步，导致 token 撤销不一致。

```go
// MemoryStore 仅适用于单实例部署
type MemoryStore struct {
    mu     sync.RWMutex
    tokens map[string]time.Time  // ⚠️ 本地内存，无法跨实例同步
    cleanupInterval time.Duration
    stopCleanup     chan struct{}
}
```

**风险评估**:
- **CVSS 评分**: 5.4 (中危)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 中等
- **影响**: 认证绕过（部分实例）

**攻击场景**:
1. 用户在实例 A 登录，获得 JWT token
2. 用户在实例 A 注销，token 被添加到实例 A 的 MemoryStore
3. 攻击者使用被撤销的 token 访问实例 B
4. 实例 B 的 MemoryStore 中没有该 token，验证通过
5. 攻击者成功使用已注销的 token 访问系统

**修复建议**:

**方案 1: 强制使用 RedisStore（推荐）**

```go
// cmd/user-center/main.go
func initJWT(cfg *config.Config, ds *datasource.Manager) (*jwt.JWT, error) {
    // 获取 Redis 客户端
    redisClient, err := ds.GetRedis("primary")
    if err != nil {
        return nil, fmt.Errorf("failed to get redis client for JWT: %w", err)
    }

    // 创建 RedisStore（分布式安全）
    store := jwt.NewRedisStore(redisClient, "jwt:revoked:")

    // 创建 JWT 认证器
    jwtAuth, err := jwt.New(
        jwt.WithOptions(cfg.JWT),
        jwt.WithStore(store),  // 强制使用 RedisStore
    )
    if err != nil {
        return nil, err
    }

    logger.Infow("JWT initialized with distributed token revocation",
        "store_type", "redis",
        "signing_method", cfg.JWT.SigningMethod)

    return jwtAuth, nil
}
```

**方案 2: 添加启动检查**

```go
// pkg/security/auth/jwt/jwt.go
func New(opts ...Option) (*JWT, error) {
    j := &JWT{
        opts: NewOptions(),
    }

    for _, opt := range opts {
        opt(j)
    }

    // 验证选项
    if err := j.opts.Complete(); err != nil {
        return nil, fmt.Errorf("complete options: %w", err)
    }

    if err := j.opts.Validate(); err != nil {
        return nil, fmt.Errorf("validate options: %w", err)
    }

    // ⚠️ 安全检查：分布式环境必须配置 Store
    if j.store == nil {
        if isDistributedEnv() {
            return nil, fmt.Errorf("SECURITY: distributed deployment detected but no token store configured. "+
                "Token revocation will not work correctly. Use jwt.WithStore(redisStore)")
        }

        // 单实例环境发出警告
        logger.Warnw("JWT initialized without token store",
            "warning", "Token revocation will not work in distributed deployments",
            "recommendation", "Configure RedisStore for production use")
    }

    // ...其余初始化代码...
}

// 检测是否为分布式环境
func isDistributedEnv() bool {
    // 检查环境变量
    if os.Getenv("DEPLOYMENT_MODE") == "distributed" {
        return true
    }

    // 检查是否配置了多个实例
    if replicas := os.Getenv("REPLICAS"); replicas != "" && replicas != "1" {
        return true
    }

    return false
}
```

**方案 3: 文档和部署指南**

```markdown
# 生产部署检查清单

## JWT 配置要求

### ✅ 必须配置（Required）
- [ ] JWT 密钥通过环境变量设置，长度 >= 128 字符
- [ ] 使用 RedisStore 实现分布式 token 撤销
- [ ] Redis 配置高可用（主从/集群模式）
- [ ] 启用 Redis 持久化（RDB + AOF）

### ⚠️ 禁止事项（Forbidden）
- [ ] 不要使用 MemoryStore（仅限单机测试）
- [ ] 不要使用硬编码密钥
- [ ] 不要禁用认证（disable-auth: false）

### 配置示例

```yaml
# user-center.yaml（生产环境）
jwt:
  disable-auth: false
  # key 必须通过环境变量设置
  signing-method: "HS512"
  expired: "2h"
  max-refresh: "24h"

redis:
  primary:
    addr: "redis-cluster:6379"
    password: "${REDIS_PASSWORD}"
    db: 0
    pool-size: 50
```

**验证方法**:
```bash
# 部署检查脚本
#!/bin/bash

# 检查 JWT Store 类型
if grep -q "NewMemoryStore" cmd/user-center/main.go; then
    echo "❌ FAIL: Using MemoryStore in production deployment"
    exit 1
fi

# 检查 Redis 配置
if ! kubectl get configmap user-center-config -o yaml | grep -q "redis:"; then
    echo "❌ FAIL: Redis not configured for JWT store"
    exit 1
fi

echo "✅ PASS: JWT configuration valid for distributed deployment"
```

**参考标准**:
- OWASP ASVS 4.0: V3.3.2 - Distributed session management
- CWE-613: Insufficient Session Expiration

---

### 🟠 MEDIUM-004: 认证中间件的 Token 标准化可能破坏签名验证

**文件位置**: `/pkg/infra/middleware/auth/auth.go:192-199`

**漏洞描述**:
Token 提取逻辑中对 token 进行了多种标准化处理，包括删除空格、转换 base64 格式。这些操作可能破坏 JWT 签名，导致合法 token 被拒绝，或在某些情况下绕过验证。

```go
// extractToken 提取并标准化 token
func extractToken(ctx transport.Context, lookup tokenLookup, scheme string) string {
    var token string

    // ...提取 token...

    // 标准化 token - ⚠️ 可能破坏 JWT 签名
    token = strings.TrimSpace(token)
    token = strings.ReplaceAll(token, " ", "")      // 删除内部空格
    token = strings.ReplaceAll(token, "+", "-")     // 标准 base64 转 URL-safe
    token = strings.ReplaceAll(token, "/", "_")     // 标准 base64 转 URL-safe
    token = strings.TrimRight(token, "=")           // 删除填充

    return token
}
```

**问题分析**:

1. **JWT 格式**: JWT 由三部分组成 `header.payload.signature`，每部分都是 base64url 编码
2. **签名依赖**: 签名是对 `header.payload` 计算的 HMAC/RSA 签名
3. **编码规范**: JWT 使用 base64url 编码（已经是 URL-safe，不需要转换）

**风险评估**:
- **CVSS 评分**: 4.3 (中低危)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 高
- **影响**: 认证绕过（理论上）、合法请求被拒绝

**潜在问题**:

1. **删除空格可能有效**（清理意外输入）
2. **Base64 转换是多余的**：JWT 本身就用 base64url
3. **删除填充符可能导致解码失败**：某些 JWT 库要求保留填充符

**修复建议**:

**方案 1: 简化 Token 提取（推荐）**

```go
// extractToken 提取并最小化标准化 token
func extractToken(ctx transport.Context, lookup tokenLookup, scheme string) string {
    var token string

    switch lookup.source {
    case "header":
        token = ctx.Header(lookup.name)
        if scheme != "" && strings.HasPrefix(token, scheme+" ") {
            token = strings.TrimPrefix(token, scheme+" ")
        }
    case "query":
        token = ctx.Query(lookup.name)
    case "cookie":
        if cookie, err := ctx.HTTPRequest().Cookie(lookup.name); err == nil {
            token = cookie.Value
        }
    }

    // 仅做最小化清理：去除首尾空白
    token = strings.TrimSpace(token)

    // 不进行 base64 格式转换，JWT 库会处理
    // 不删除填充符，某些实现可能依赖它

    return token
}
```

**方案 2: 添加配置选项控制标准化行为**

```go
type AuthOptions struct {
    // ...现有字段...

    // TokenNormalization 控制 token 标准化行为
    TokenNormalization TokenNormalizationMode
}

type TokenNormalizationMode int

const (
    // NormalizeNone 不做任何标准化（推荐）
    NormalizeNone TokenNormalizationMode = iota
    // NormalizeTrimSpace 仅去除首尾空白
    NormalizeTrimSpace
    // NormalizeFull 完整标准化（当前行为，可能有问题）
    NormalizeFull
)
```

**方案 3: 添加测试验证**

```go
func TestExtractToken_PreservesJWTFormat(t *testing.T) {
    // 真实 JWT token（HS256 签名）
    validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
                  "eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." +
                  "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

    // 模拟请求
    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("Authorization", "Bearer "+validToken)

    ctx := &mockContext{req: req}
    lookup := tokenLookup{source: "header", name: "Authorization"}

    // 提取 token
    extracted := extractToken(ctx, lookup, "Bearer")

    // 验证：提取的 token 应该完全相同
    assert.Equal(t, validToken, extracted, "Token should not be modified")

    // 验证：token 应该能被 JWT 库解析
    parser := jwt.NewParser()
    _, _, err := parser.ParseUnverified(extracted, jwt.MapClaims{})
    assert.NoError(t, err, "Extracted token should be valid JWT")
}

func TestExtractToken_HandlesWhitespace(t *testing.T) {
    validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc"

    tests := []struct {
        name     string
        header   string
        expected string
    }{
        {
            name:     "正常格式",
            header:   "Bearer " + validToken,
            expected: validToken,
        },
        {
            name:     "额外空格",
            header:   "Bearer  " + validToken + " ",
            expected: validToken,
        },
        {
            name:     "前后空白",
            header:   " Bearer " + validToken + "\n",
            expected: validToken,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/test", nil)
            req.Header.Set("Authorization", tt.header)

            ctx := &mockContext{req: req}
            lookup := tokenLookup{source: "header", name: "Authorization"}

            extracted := extractToken(ctx, lookup, "Bearer")
            assert.Equal(t, tt.expected, extracted)
        })
    }
}
```

**建议行动**:
1. 立即移除 base64 格式转换逻辑（第 194-196 行）
2. 移除填充符删除逻辑（第 197 行）
3. 保留空格清理逻辑（第 193 行）
4. 添加测试验证修改后不会破坏 JWT 解析

**参考标准**:
- RFC 7519: JSON Web Token (JWT)
- RFC 4648: Base64 encoding
- OWASP ASVS 4.0: V3.5.2 - Token verification

---

## 4. 低危漏洞（Low Severity）

### 🟡 LOW-001: 数据库查询未使用参数化，存在潜在 SQL 注入风险

**文件位置**: `/internal/user-center/store/user.go`

**漏洞描述**:
虽然项目使用了 GORM ORM 框架，所有查询都通过参数化处理，但在 `List` 方法中使用了原始 SQL 字符串（窗口函数）：

```go
func (u *users) List(ctx context.Context, offset, limit int) (int64, []*model.User, error) {
    // ...
    err := u.db.WithContext(ctx).
        Select(`
            id,
            username,
            email,
            avatar,
            mobile,
            status,
            created_at,
            updated_at,
            created_by,
            updated_by,
            COUNT(*) OVER() as total_count
        `).  // ⚠️ 硬编码字段列表，虽然安全但不利于维护
        Model(&model.User{}).
        Offset(offset).  // ✓ GORM 自动参数化
        Limit(limit).    // ✓ GORM 自动参数化
        Find(&results).Error
    // ...
}
```

**当前状态评估**: **安全 ✓**
- `offset` 和 `limit` 由 GORM 自动参数化
- 字段列表是硬编码的，不接受用户输入
- 没有动态 SQL 拼接

**风险评估**:
- **CVSS 评分**: 2.4 (低危)
- **攻击向量**: 无直接攻击路径（当前实现安全）
- **影响**: 潜在维护风险

**改进建议**（维护性优化，非安全必需）:

```go
func (u *users) List(ctx context.Context, offset, limit int) (int64, []*model.User, error) {
    // 方案 1: 使用 GORM 的 Select 字段列表
    selectFields := []string{
        "id", "username", "email", "avatar", "mobile",
        "status", "created_at", "updated_at", "created_by", "updated_by",
    }

    var results []struct {
        model.User
        TotalCount int64 `gorm:"column:total_count"`
    }

    err := u.db.WithContext(ctx).
        Select(strings.Join(selectFields, ", ") + ", COUNT(*) OVER() as total_count").
        Model(&model.User{}).
        Offset(offset).
        Limit(limit).
        Find(&results).Error

    // ...
}
```

**参考标准**:
- OWASP Top 10 2021: A03:2021 - Injection
- CWE-89: SQL Injection

---

### 🟡 LOW-002: 敏感日志信息泄露风险

**文件位置**:
- `/pkg/infra/middleware/auth/auth.go:233-259`
- `/pkg/security/auth/jwt/jwt.go:349-353`

**漏洞描述**:
认证失败日志中记录了 token 前缀，虽然不是完整 token，但仍可能泄露部分信息。

```go
func logAuthFailure(ctx transport.Context, token string, err error) {
    // 记录 token 前缀 - ⚠️ 可能泄露部分信息
    tokenPrefix := ""
    if len(token) > 20 {
        tokenPrefix = token[:20] + "..."  // 前 20 字符
    } else if len(token) > 0 {
        tokenPrefix = token[:len(token)/2] + "..."
    }

    logger.Warnw("authentication failed",
        "error", err.Error(),
        "remote_addr", req.RemoteAddr,
        "token_prefix", tokenPrefix,  // ⚠️ 记录 token 前缀
        // ...
    )
}
```

**风险评估**:
- **CVSS 评分**: 3.1 (低危)
- **攻击向量**: 本地/日志访问
- **攻击复杂度**: 高
- **影响**: 信息泄露（有限）

**攻击场景**:
1. 攻击者获取日志文件访问权限
2. 从日志中收集大量 token 前缀
3. 尝试通过前缀模式分析 token 生成算法
4. 在极端情况下可能辅助暴力破解

**改进建议**:

```go
func logAuthFailure(ctx transport.Context, token string, err error) {
    // 方案 1: 只记录 token 哈希
    tokenHash := ""
    if len(token) > 0 {
        h := sha256.Sum256([]byte(token))
        tokenHash = hex.EncodeToString(h[:8])  // 前 8 字节
    }

    logger.Warnw("authentication failed",
        "error", err.Error(),
        "remote_addr", req.RemoteAddr,
        "token_hash", tokenHash,  // ✓ 只记录哈希，无法反推
        "path", req.URL.Path,
        "method", req.Method,
        "user_agent", req.UserAgent(),
    )
}

// 方案 2: 使用请求 ID 关联
func logAuthFailure(ctx transport.Context, token string, err error) {
    // 从上下文获取请求 ID（需要配置请求 ID 中间件）
    requestID := ctx.Get("request_id")

    logger.Warnw("authentication failed",
        "error", err.Error(),
        "request_id", requestID,  // ✓ 使用请求 ID，不记录 token
        "remote_addr", req.RemoteAddr,
        "path", req.URL.Path,
        "method", req.Method,
    )
}
```

**配置日志策略**:

```yaml
# 生产环境日志配置
logging:
  level: "info"  # 不要使用 debug
  output: "json"
  redact-fields:  # 自动脱敏字段
    - "token"
    - "password"
    - "authorization"
    - "cookie"
  rotation:
    enabled: true
    max-size: "100MB"
    max-age: "7d"
    compress: true
  access-control:
    - path: "/var/log/sentinel-x/"
      mode: "0600"  # 仅应用程序可读写
      owner: "sentinel-x"
```

**参考标准**:
- OWASP Logging Cheat Sheet
- CWE-532: Insertion of Sensitive Information into Log File

---

### 🟡 LOW-003: User-Agent 和 IP 地址可被伪造，影响审计日志可靠性

**文件位置**:
- `/pkg/infra/middleware/auth/auth.go:257`
- `/pkg/security/auth/middleware/http.go:92-95`

**漏洞描述**:
安全审计日志中记录的 `User-Agent` 和 `RemoteAddr` 可以被客户端伪造，降低了日志的可靠性和取证价值。

```go
func logAuthFailure(ctx transport.Context, token string, err error) {
    logger.Warnw("authentication failed",
        "error", err.Error(),
        "remote_addr", req.RemoteAddr,      // ⚠️ 可能是代理地址
        "token_prefix", tokenPrefix,
        "path", req.URL.Path,
        "method", req.Method,
        "user_agent", req.UserAgent(),      // ⚠️ 可以伪造
    )
}
```

**风险评估**:
- **CVSS 评分**: 2.7 (低危)
- **攻击向量**: 网络（远程）
- **攻击复杂度**: 低
- **影响**: 审计绕过（日志污染）

**攻击场景**:
1. 攻击者发起攻击时伪造 User-Agent 和 X-Forwarded-For
2. 安全日志中记录了虚假的来源信息
3. 事后取证时无法准确定位攻击来源
4. 影响威胁情报和入侵检测系统的准确性

**改进建议**:

**方案 1: 记录多层地址信息**

```go
func logAuthFailure(ctx transport.Context, token string, err error) {
    req := ctx.HTTPRequest()
    if req == nil {
        return
    }

    // 提取多层地址信息
    clientIP := getClientIP(req)
    directIP := getDirectIP(req)
    forwardedFor := req.Header.Get("X-Forwarded-For")
    realIP := req.Header.Get("X-Real-IP")

    logger.Warnw("authentication failed",
        "error", err.Error(),
        // 网络层信息
        "client_ip", clientIP,              // ✓ 经过验证的客户端 IP
        "direct_ip", directIP,              // ✓ 直接连接的 IP（无法伪造）
        "x_forwarded_for", forwardedFor,    // ⚠️ 可能被伪造（记录但不信任）
        "x_real_ip", realIP,                // ⚠️ 可能被伪造
        // 请求信息
        "path", req.URL.Path,
        "method", req.Method,
        "user_agent", req.UserAgent(),      // ⚠️ 可伪造但仍有分析价值
        "user_agent_hash", hashUserAgent(req.UserAgent()),  // ✓ 用于聚合分析
        // 指纹信息
        "tls_version", getTLSVersion(req),  // ✓ 难以伪造
        "cipher_suite", getCipherSuite(req), // ✓ 难以伪造
    )
}

// getClientIP 提取可信的客户端 IP
func getClientIP(req *http.Request) string {
    // 如果在负载均衡器后面，信任特定头部
    if trustedProxy := os.Getenv("TRUSTED_PROXY"); trustedProxy != "" {
        if forwardedFor := req.Header.Get("X-Forwarded-For"); forwardedFor != "" {
            // 取第一个 IP（原始客户端）
            ips := strings.Split(forwardedFor, ",")
            if len(ips) > 0 {
                return strings.TrimSpace(ips[0])
            }
        }
    }

    // 否则使用直接连接的 IP
    return getDirectIP(req)
}

// getDirectIP 获取直接连接的 IP（无法伪造）
func getDirectIP(req *http.Request) string {
    if req.RemoteAddr != "" {
        // 移除端口号
        host, _, err := net.SplitHostPort(req.RemoteAddr)
        if err == nil {
            return host
        }
        return req.RemoteAddr
    }
    return "unknown"
}
```

**方案 2: TLS 指纹识别**

```go
// getTLSVersion 获取 TLS 版本（难以伪造）
func getTLSVersion(req *http.Request) string {
    if req.TLS == nil {
        return "no-tls"
    }

    switch req.TLS.Version {
    case tls.VersionTLS10:
        return "TLS 1.0"
    case tls.VersionTLS11:
        return "TLS 1.1"
    case tls.VersionTLS12:
        return "TLS 1.2"
    case tls.VersionTLS13:
        return "TLS 1.3"
    default:
        return "unknown"
    }
}

// getCipherSuite 获取加密套件
func getCipherSuite(req *http.Request) string {
    if req.TLS == nil {
        return "no-tls"
    }
    return tls.CipherSuiteName(req.TLS.CipherSuite)
}
```

**方案 3: 网络层验证**

```yaml
# 配置可信代理列表
network:
  trusted-proxies:
    - "10.0.0.0/8"      # 内部网络
    - "172.16.0.0/12"   # 内部网络
    - "192.168.0.0/16"  # 内部网络

  client-ip-header: "X-Forwarded-For"
  validate-forwarded-headers: true
```

**验证方法**:
```bash
# 测试 1: 正常请求
curl -H "X-Forwarded-For: 1.2.3.4" https://api.example.com/auth/login
# 日志应该记录: direct_ip=<实际IP>, x_forwarded_for=1.2.3.4

# 测试 2: 伪造 User-Agent
curl -A "AttackerBot/1.0" https://api.example.com/auth/login
# 日志应该记录: user_agent=AttackerBot/1.0, user_agent_hash=<hash>
```

**参考标准**:
- OWASP Logging Cheat Sheet: V7.2 - Logging content
- CWE-639: Authorization Bypass Through User-Controlled Key

---

## 5. 安全最佳实践建议

### 💡 BEST-PRACTICE-001: 实施安全响应头

**建议**:
添加 HTTP 安全响应头中间件，防御常见的 Web 攻击。

```go
// pkg/infra/middleware/security/headers.go
package security

import (
    "github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// SecurityHeadersConfig 安全头配置
type SecurityHeadersConfig struct {
    EnableHSTS            bool
    HSTSMaxAge            int
    EnableCSP             bool
    CSPDirective          string
    EnableXFrameOptions   bool
    XFrameOptions         string
    EnableXContentType    bool
    EnableReferrerPolicy  bool
    ReferrerPolicy        string
}

// DefaultSecurityHeadersConfig 默认配置
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
    return &SecurityHeadersConfig{
        EnableHSTS:           true,
        HSTSMaxAge:           31536000, // 1 年
        EnableCSP:            true,
        CSPDirective:         "default-src 'self'",
        EnableXFrameOptions:  true,
        XFrameOptions:        "DENY",
        EnableXContentType:   true,
        EnableReferrerPolicy: true,
        ReferrerPolicy:       "strict-origin-when-cross-origin",
    }
}

// SecurityHeaders 添加安全响应头
func SecurityHeaders(config *SecurityHeadersConfig) transport.MiddlewareFunc {
    if config == nil {
        config = DefaultSecurityHeadersConfig()
    }

    return func(next transport.HandlerFunc) transport.HandlerFunc {
        return func(ctx transport.Context) {
            // Strict-Transport-Security (HSTS)
            if config.EnableHSTS {
                ctx.SetHeader("Strict-Transport-Security",
                    fmt.Sprintf("max-age=%d; includeSubDomains; preload", config.HSTSMaxAge))
            }

            // Content-Security-Policy (CSP)
            if config.EnableCSP {
                ctx.SetHeader("Content-Security-Policy", config.CSPDirective)
            }

            // X-Frame-Options (防御点击劫持)
            if config.EnableXFrameOptions {
                ctx.SetHeader("X-Frame-Options", config.XFrameOptions)
            }

            // X-Content-Type-Options (防止 MIME 类型嗅探)
            if config.EnableXContentType {
                ctx.SetHeader("X-Content-Type-Options", "nosniff")
            }

            // Referrer-Policy
            if config.EnableReferrerPolicy {
                ctx.SetHeader("Referrer-Policy", config.ReferrerPolicy)
            }

            // X-XSS-Protection (遗留浏览器)
            ctx.SetHeader("X-XSS-Protection", "1; mode=block")

            // Permissions-Policy (限制浏览器功能)
            ctx.SetHeader("Permissions-Policy",
                "geolocation=(), microphone=(), camera=()")

            next(ctx)
        }
    }
}
```

**使用方法**:
```go
// cmd/user-center/main.go
func setupMiddlewares(router *gin.Engine) {
    // 添加安全头中间件
    router.Use(security.SecurityHeaders(nil)) // 使用默认配置

    // 或自定义配置
    router.Use(security.SecurityHeaders(&security.SecurityHeadersConfig{
        EnableHSTS:    true,
        HSTSMaxAge:    63072000, // 2 年
        EnableCSP:     true,
        CSPDirective:  "default-src 'self'; script-src 'self' 'unsafe-inline'",
        // ...
    }))
}
```

---

### 💡 BEST-PRACTICE-002: 添加 Rate Limiting 防护

**建议**:
实施细粒度的速率限制，防止暴力破解和 DoS 攻击。

```go
// pkg/infra/middleware/security/ratelimit.go
package security

import (
    "fmt"
    "time"

    "github.com/kart-io/sentinel-x/pkg/component/redis"
    "github.com/kart-io/sentinel-x/pkg/infra/server/transport"
    "github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
    // 全局速率限制
    GlobalRate  int           // 每秒请求数
    GlobalBurst int           // 突发容量

    // 按 IP 限制
    PerIPRate   int
    PerIPWindow time.Duration

    // 按用户限制
    PerUserRate   int
    PerUserWindow time.Duration

    // 特定端点限制
    EndpointRules map[string]EndpointRule
}

type EndpointRule struct {
    Rate   int
    Window time.Duration
}

// RateLimiterFactory 创建速率限制器
type RateLimiterFactory struct {
    redis  *redis.Client
    config *RateLimitConfig
}

func NewRateLimiterFactory(redis *redis.Client, config *RateLimitConfig) *RateLimiterFactory {
    return &RateLimiterFactory{
        redis:  redis,
        config: config,
    }
}

// RateLimitByIP 按 IP 限制
func (f *RateLimiterFactory) RateLimitByIP() transport.MiddlewareFunc {
    return func(next transport.HandlerFunc) transport.HandlerFunc {
        return func(ctx transport.Context) {
            ip := ctx.ClientIP()
            key := fmt.Sprintf("ratelimit:ip:%s", ip)

            allowed, err := f.checkRateLimit(ctx.Request(), key, f.config.PerIPRate, f.config.PerIPWindow)
            if err != nil {
                logger.Errorw("rate limit check failed", "error", err)
                // 失败时允许通过（fail-open）
                next(ctx)
                return
            }

            if !allowed {
                ctx.JSON(429, errors.ErrTooManyRequests.WithMessage("请求过于频繁，请稍后再试"))
                return
            }

            next(ctx)
        }
    }
}

// RateLimitByEndpoint 按端点限制
func (f *RateLimiterFactory) RateLimitByEndpoint() transport.MiddlewareFunc {
    return func(next transport.HandlerFunc) transport.HandlerFunc {
        return func(ctx transport.Context) {
            path := ctx.HTTPRequest().URL.Path

            // 检查是否有特定规则
            if rule, exists := f.config.EndpointRules[path]; exists {
                ip := ctx.ClientIP()
                key := fmt.Sprintf("ratelimit:endpoint:%s:%s", path, ip)

                allowed, err := f.checkRateLimit(ctx.Request(), key, rule.Rate, rule.Window)
                if err != nil {
                    logger.Errorw("endpoint rate limit check failed", "error", err, "path", path)
                    next(ctx)
                    return
                }

                if !allowed {
                    ctx.JSON(429, errors.ErrTooManyRequests.WithMessage(
                        fmt.Sprintf("该接口访问过于频繁，请在 %v 后重试", rule.Window)))
                    return
                }
            }

            next(ctx)
        }
    }
}

// checkRateLimit 使用 Redis 实现滑动窗口速率限制
func (f *RateLimiterFactory) checkRateLimit(ctx context.Context, key string, rate int, window time.Duration) (bool, error) {
    now := time.Now()
    windowStart := now.Add(-window).UnixNano()

    pipe := f.redis.Client().Pipeline()

    // 1. 移除窗口外的记录
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

    // 2. 计算当前窗口内的请求数
    countCmd := pipe.ZCard(ctx, key)

    // 3. 添加当前请求
    pipe.ZAdd(ctx, key, redis.Z{
        Score:  float64(now.UnixNano()),
        Member: now.UnixNano(),
    })

    // 4. 设置过期时间
    pipe.Expire(ctx, key, window)

    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }

    count := countCmd.Val()
    return count < int64(rate), nil
}
```

**配置示例**:
```yaml
# configs/user-center.yaml
security:
  rate-limit:
    enabled: true

    # 全局限制
    global-rate: 1000  # 每秒 1000 请求
    global-burst: 100

    # 按 IP 限制
    per-ip-rate: 100
    per-ip-window: "1m"

    # 按用户限制
    per-user-rate: 200
    per-user-window: "1m"

    # 特定端点限制
    endpoint-rules:
      "/auth/login":
        rate: 5
        window: "1m"  # 每分钟最多 5 次登录尝试
      "/auth/register":
        rate: 3
        window: "1h"  # 每小时最多 3 次注册
      "/auth/password-reset":
        rate: 3
        window: "1h"
```

---

### 💡 BEST-PRACTICE-003: 实施 API 输入长度限制

**建议**:
添加请求体大小限制，防止拒绝服务攻击。

```go
// pkg/infra/middleware/security/bodysize.go
package security

import (
    "github.com/kart-io/sentinel-x/pkg/infra/server/transport"
    "github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// BodySizeLimit 限制请求体大小
func BodySizeLimit(maxBytes int64) transport.MiddlewareFunc {
    return func(next transport.HandlerFunc) transport.HandlerFunc {
        return func(ctx transport.Context) {
            req := ctx.HTTPRequest()

            // 检查 Content-Length
            if req.ContentLength > maxBytes {
                ctx.JSON(413, errors.ErrRequestEntityTooLarge.WithMessage(
                    fmt.Sprintf("请求体过大，最大允许 %d 字节", maxBytes)))
                return
            }

            // 限制实际读取大小（防止 Content-Length 伪造）
            req.Body = http.MaxBytesReader(ctx.Writer(), req.Body, maxBytes)

            next(ctx)
        }
    }
}
```

**使用方法**:
```go
// cmd/user-center/main.go
router.Use(security.BodySizeLimit(1 * 1024 * 1024)) // 1 MB
```

---

### 💡 BEST-PRACTICE-004: 添加安全审计事件

**建议**:
记录关键安全事件到专用审计日志。

```go
// pkg/infra/audit/audit.go
package audit

import (
    "context"
    "time"

    "github.com/kart-io/logger"
)

// EventType 审计事件类型
type EventType string

const (
    EventLogin              EventType = "LOGIN"
    EventLoginFailed        EventType = "LOGIN_FAILED"
    EventLogout             EventType = "LOGOUT"
    EventRegister           EventType = "REGISTER"
    EventPasswordChange     EventType = "PASSWORD_CHANGE"
    EventPasswordReset      EventType = "PASSWORD_RESET"
    EventAccountLocked      EventType = "ACCOUNT_LOCKED"
    EventAccountUnlocked    EventType = "ACCOUNT_UNLOCKED"
    EventTokenRefresh       EventType = "TOKEN_REFRESH"
    EventTokenRevoke        EventType = "TOKEN_REVOKE"
    EventPermissionDenied   EventType = "PERMISSION_DENIED"
    EventDataAccess         EventType = "DATA_ACCESS"
    EventDataModification   EventType = "DATA_MODIFICATION"
    EventSecurityViolation  EventType = "SECURITY_VIOLATION"
)

// AuditEvent 审计事件
type AuditEvent struct {
    Timestamp   time.Time              `json:"timestamp"`
    EventType   EventType              `json:"event_type"`
    Actor       string                 `json:"actor"`        // 操作者
    ActorIP     string                 `json:"actor_ip"`
    Resource    string                 `json:"resource"`     // 操作资源
    Action      string                 `json:"action"`       // 操作动作
    Result      string                 `json:"result"`       // 成功/失败
    Details     map[string]interface{} `json:"details"`
    Severity    string                 `json:"severity"`     // INFO/WARN/ERROR/CRITICAL
}

// Logger 审计日志记录器
type Logger struct {
    // 可以使用专用的审计日志后端（数据库、文件、SIEM）
}

// Log 记录审计事件
func (l *Logger) Log(ctx context.Context, event *AuditEvent) {
    // 补充时间戳
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }

    // 记录到专用审计日志
    logger.Infow("AUDIT",
        "timestamp", event.Timestamp.Unix(),
        "event_type", event.EventType,
        "actor", event.Actor,
        "actor_ip", event.ActorIP,
        "resource", event.Resource,
        "action", event.Action,
        "result", event.Result,
        "severity", event.Severity,
        "details", event.Details,
    )

    // TODO: 也可以发送到 SIEM 系统、安全运营中心
}

// LogLogin 记录登录事件
func (l *Logger) LogLogin(ctx context.Context, username, ip string, success bool) {
    result := "SUCCESS"
    severity := "INFO"
    eventType := EventLogin

    if !success {
        result = "FAILED"
        severity = "WARN"
        eventType = EventLoginFailed
    }

    l.Log(ctx, &AuditEvent{
        EventType: eventType,
        Actor:     username,
        ActorIP:   ip,
        Resource:  "auth",
        Action:    "login",
        Result:    result,
        Severity:  severity,
    })
}
```

**集成到业务代码**:
```go
// internal/user-center/biz/auth.go
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
    // 提取 IP
    ip := getClientIP(ctx)

    // ...原有登录逻辑...

    // 成功登录
    s.auditLogger.LogLogin(ctx, req.Username, ip, true)

    return response, nil
}
```

---

### 💡 BEST-PRACTICE-005: 实施密钥轮换策略

**建议**:
定期轮换 JWT 签名密钥，降低密钥泄露风险。

```go
// pkg/security/auth/jwt/keyrotation.go
package jwt

import (
    "context"
    "sync"
    "time"

    "github.com/kart-io/logger"
)

// KeyManager 管理密钥轮换
type KeyManager struct {
    mu          sync.RWMutex
    currentKey  string
    previousKey string
    nextRotation time.Time
    rotationInterval time.Duration
}

// NewKeyManager 创建密钥管理器
func NewKeyManager(initialKey string, rotationInterval time.Duration) *KeyManager {
    km := &KeyManager{
        currentKey:       initialKey,
        rotationInterval: rotationInterval,
        nextRotation:     time.Now().Add(rotationInterval),
    }

    // 启动后台轮换任务
    go km.rotationWorker()

    return km
}

// GetCurrentKey 获取当前密钥
func (km *KeyManager) GetCurrentKey() string {
    km.mu.RLock()
    defer km.mu.RUnlock()
    return km.currentKey
}

// GetVerificationKeys 获取验证密钥列表（包含当前和之前的密钥）
func (km *KeyManager) GetVerificationKeys() []string {
    km.mu.RLock()
    defer km.mu.RUnlock()

    keys := []string{km.currentKey}
    if km.previousKey != "" {
        keys = append(keys, km.previousKey)
    }
    return keys
}

// RotateKey 执行密钥轮换
func (km *KeyManager) RotateKey(newKey string) {
    km.mu.Lock()
    defer km.mu.Unlock()

    logger.Infow("rotating JWT signing key",
        "previous_key_hash", hashKey(km.currentKey),
        "new_key_hash", hashKey(newKey))

    km.previousKey = km.currentKey
    km.currentKey = newKey
    km.nextRotation = time.Now().Add(km.rotationInterval)
}

// rotationWorker 后台轮换任务
func (km *KeyManager) rotationWorker() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        if time.Now().After(km.nextRotation) {
            // 生成新密钥
            newKey, err := generateSecureKey(128) // 128 字符
            if err != nil {
                logger.Errorw("failed to generate new key", "error", err)
                continue
            }

            km.RotateKey(newKey)
        }
    }
}
```

---

## 6. 总结与建议

### 立即行动项（Critical & High）

| 优先级 | 漏洞ID | 描述 | 预计修复时间 |
|--------|--------|------|------------|
| P0 | CRITICAL-001 | 移除硬编码 JWT 密钥，强制环境变量 | 1 小时 |
| P1 | HIGH-001 | 修复 Token 刷新竞态条件 | 4 小时 |
| P1 | HIGH-002 | 优化 Token 撤销 TTL 逻辑 | 2 小时 |

### 短期改进项（Medium, 1-2 周）

1. 添加密码强度验证器（MEDIUM-001）
2. 实施账户锁定机制（MEDIUM-002）
3. 强制使用 RedisStore（MEDIUM-003）
4. 简化 Token 提取逻辑（MEDIUM-004）

### 长期优化项（Low & Best Practices, 1-2 个月）

1. 实施安全响应头
2. 添加 Rate Limiting 防护
3. 完善安全审计日志
4. 实施密钥轮换策略
5. 优化日志脱敏策略

### 合规性检查

✅ **已满足**:
- OWASP Top 10 2021: A02 (Cryptographic Failures) - 使用 bcrypt 哈希密码
- OWASP Top 10 2021: A03 (Injection) - 使用 GORM 参数化查询
- OWASP Top 10 2021: A07 (Identification and Authentication Failures) - 部分满足

⚠️ **需改进**:
- OWASP Top 10 2021: A07 - 密码策略、账户锁定
- OWASP ASVS 4.0: V2.1 (Password Security) - 密码强度
- OWASP ASVS 4.0: V2.2 (General Authenticator Security) - 暴力破解防护

### 风险评级矩阵

```
影响 \\ 可能性  | 低 | 中 | 高 |
----------------|----|----|----|
严重 (Critical) |    |    | ■  | ← CRITICAL-001
高 (High)       |    | ■  | ■  | ← HIGH-001, HIGH-002
中 (Medium)     | ■  | ■  | ■  | ← MEDIUM-001~004
低 (Low)        | ■  | ■  |    | ← LOW-001~003
```

### 推荐的修复顺序

**第 1 天（紧急）**:
1. 修复 CRITICAL-001（硬编码密钥）
2. 修复 HIGH-001（Token 刷新竞态）

**第 1 周**:
3. 修复 HIGH-002（Token 撤销 TTL）
4. 实施 MEDIUM-001（密码强度）
5. 实施 MEDIUM-002（账户锁定）

**第 2 周**:
6. 修复 MEDIUM-003（强制 RedisStore）
7. 修复 MEDIUM-004（Token 提取）
8. 修复 LOW-001~003

**第 3-4 周**:
9. 实施安全最佳实践（BEST-PRACTICE-001~005）
10. 完善监控和告警
11. 进行安全回归测试

---

## 附录

### A. 参考标准

- OWASP Top 10 2021
- OWASP ASVS 4.0
- NIST SP 800-63B: Digital Identity Guidelines
- CWE/SANS Top 25 Most Dangerous Software Weaknesses
- RFC 7519: JSON Web Token (JWT)
- RFC 6749: OAuth 2.0 Authorization Framework

### B. 测试工具推荐

- **静态代码分析**: gosec, semgrep
- **依赖扫描**: govulncheck, Snyk
- **动态测试**: OWASP ZAP, Burp Suite
- **JWT 测试**: jwt_tool, jwt.io
- **渗透测试**: Metasploit, SQLMap

### C. 联系方式

如有安全相关问题或发现新的安全漏洞，请联系：
- **安全团队邮箱**: security@example.com
- **漏洞报告**: https://example.com/security/report
- **紧急热线**: +86-xxx-xxxx-xxxx (仅用于严重安全事件)

---

**报告结束**

*本报告为机密文档，仅供内部使用。未经授权不得外传。*
