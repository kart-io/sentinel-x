# 环境变量配置指南

## 概述

Sentinel-X 支持通过环境变量覆盖 YAML 配置文件中的设置。这是生产环境中管理敏感信息（如密码、密钥）的推荐方式。

## 环境变量命名规则

环境变量遵循以下命名规则：

```text
<应用名称>_<配置键路径>
```

- 应用名称：配置文件名的大写形式（如 `SENTINEL_API`、`USER_CENTER`）
- 配置键路径：YAML 配置文件中的键路径，使用下划线 `_` 分隔
- 所有字母必须大写
- 配置键中的点号 `.` 和横杠 `-` 都会被转换为下划线 `_`

### 示例

YAML 配置文件 `sentinel-api.yaml` 中的配置：

```yaml
jwt:
  key: "secret-key"
mysql:
  password: "db-password"
  max-idle-connections: 10
```

对应的环境变量：

```bash
SENTINEL_API_JWT_KEY="your-secret-key"
SENTINEL_API_MYSQL_PASSWORD="your-db-password"
SENTINEL_API_MYSQL_MAX_IDLE_CONNECTIONS=10
```

## 敏感信息清单

以下配置项包含敏感信息，必须通过环境变量设置，不应硬编码在配置文件中：

### Sentinel API Server (sentinel-api)

- `SENTINEL_API_JWT_KEY` - JWT 签名密钥（最少 64 个字符）
- `SENTINEL_API_MYSQL_USERNAME` - MySQL 用户名
- `SENTINEL_API_MYSQL_PASSWORD` - MySQL 密码
- `SENTINEL_API_MYSQL_DATABASE` - MySQL 数据库名
- `SENTINEL_API_REDIS_PASSWORD` - Redis 密码

### User Center (user-center)

- `USER_CENTER_JWT_KEY` - JWT 签名密钥（最少 64 个字符）
- `USER_CENTER_MYSQL_USERNAME` - MySQL 用户名
- `USER_CENTER_MYSQL_PASSWORD` - MySQL 密码
- `USER_CENTER_MYSQL_DATABASE` - MySQL 数据库名
- `USER_CENTER_REDIS_PASSWORD` - Redis 密码

### Sentinel Example (sentinel-example)

- `SENTINEL_EXAMPLE_JWT_KEY` - JWT 签名密钥（最少 32 个字符）

## 使用方法

### 方法一：使用 .env 文件（开发环境）

1. 复制示例文件：

   ```bash
   cp .env.example .env
   ```

2. 编辑 `.env` 文件，填入实际值：

   ```bash
   # .env
   SENTINEL_API_JWT_KEY="your-64-character-secret-key-here..."
   SENTINEL_API_MYSQL_USERNAME="root"
   SENTINEL_API_MYSQL_PASSWORD="your-secure-password"
   SENTINEL_API_MYSQL_DATABASE="user_auth"
   ```

3. 使用环境变量加载工具（如 `direnv`、`dotenv`）自动加载：

   ```bash
   # 安装 direnv
   brew install direnv

   # 允许 .envrc
   echo 'dotenv' > .envrc
   direnv allow
   ```

### 方法二：直接导出环境变量

```bash
export SENTINEL_API_JWT_KEY="your-64-character-secret-key-here..."
export SENTINEL_API_MYSQL_USERNAME="root"
export SENTINEL_API_MYSQL_PASSWORD="your-secure-password"
export SENTINEL_API_MYSQL_DATABASE="user_auth"

# 运行应用
./bin/sentinel-api
```

### 方法三：Docker 容器环境变量

使用 Docker Compose：

```yaml
# docker-compose.yml
services:
  sentinel-api:
    image: sentinel-x/api:latest
    environment:
      - SENTINEL_API_JWT_KEY=${SENTINEL_API_JWT_KEY}
      - SENTINEL_API_MYSQL_USERNAME=${SENTINEL_API_MYSQL_USERNAME}
      - SENTINEL_API_MYSQL_PASSWORD=${SENTINEL_API_MYSQL_PASSWORD}
      - SENTINEL_API_MYSQL_DATABASE=${SENTINEL_API_MYSQL_DATABASE}
    env_file:
      - .env
```

或使用 Docker 命令行：

```bash
docker run -d \
  -e SENTINEL_API_JWT_KEY="your-key" \
  -e SENTINEL_API_MYSQL_USERNAME="root" \
  -e SENTINEL_API_MYSQL_PASSWORD="password" \
  -e SENTINEL_API_MYSQL_DATABASE="user_auth" \
  sentinel-x/api:latest
```

### 方法四：Kubernetes Secrets

1. 创建 Secret：

   ```bash
   kubectl create secret generic sentinel-api-secrets \
     --from-literal=jwt-key='your-64-character-secret-key' \
     --from-literal=mysql-password='your-secure-password'
   ```

2. 在 Deployment 中引用：

   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: sentinel-api
   spec:
     template:
       spec:
         containers:
         - name: sentinel-api
           image: sentinel-x/api:latest
           env:
           - name: SENTINEL_API_JWT_KEY
             valueFrom:
               secretKeyRef:
                 name: sentinel-api-secrets
                 key: jwt-key
           - name: SENTINEL_API_MYSQL_PASSWORD
             valueFrom:
               secretKeyRef:
                 name: sentinel-api-secrets
                 key: mysql-password
   ```

## 密钥生成

### JWT 密钥生成

生成 64 字符（512 位）的强随机密钥：

```bash
# 使用 OpenSSL
openssl rand -base64 64 | tr -d '\n'

# 使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(64))"

# 使用 Go
go run -c 'import("crypto/rand"; "encoding/base64"; "fmt"; "io"); b := make([]byte, 64); io.ReadFull(rand.Reader, b); fmt.Println(base64.StdEncoding.EncodeToString(b))'
```

### MySQL 密码生成

生成强随机密码：

```bash
# 使用 OpenSSL
openssl rand -base64 32 | tr -d '\n'

# 使用 pwgen
pwgen -s 32 1
```

## 安全建议

1. **不要将敏感信息硬编码在配置文件中**
   - 配置文件中的密码和密钥应设置为空字符串
   - 使用环境变量或密钥管理服务

2. **不要将 .env 文件提交到版本控制**
   - `.env` 已在 `.gitignore` 中
   - 只提交 `.env.example` 示例文件

3. **使用强随机密钥**
   - JWT 密钥至少 64 个字符
   - 使用加密安全的随机生成器
   - 不要使用字典单词或简单模式

4. **定期轮换密钥**
   - 建议每 90 天轮换一次
   - 实施密钥轮换策略和流程

5. **使用密钥管理服务**
   - 生产环境推荐使用专业的密钥管理服务：
     - HashiCorp Vault
     - AWS Secrets Manager
     - Azure Key Vault
     - Google Cloud Secret Manager

6. **限制环境变量访问权限**
   - 只授权必要的人员和进程访问
   - 使用 RBAC（基于角色的访问控制）

7. **审计和监控**
   - 记录密钥访问日志
   - 监控异常访问模式
   - 定期审查权限设置

## 配置优先级

配置加载的优先级顺序（从高到低）：

1. 命令行标志（Flags）
2. 环境变量（Environment Variables）
3. 配置文件（YAML Files）
4. 默认值（Default Values）

环境变量会覆盖配置文件中的设置，但会被命令行标志覆盖。

## 验证配置

启动应用时，可以通过日志检查配置是否正确加载：

```bash
# 设置日志级别为 debug
export SENTINEL_API_LOG_LEVEL=debug

# 运行应用（密码会被自动脱敏显示为 [REDACTED]）
./bin/sentinel-api
```

日志示例：

```text
INFO  [config] Configuration loaded successfully
DEBUG [config] MySQL{host=localhost, port=3306, user=root, password=[REDACTED], database=user_auth}
DEBUG [config] Redis{host=localhost, port=6379, password=[REDACTED], database=0}
DEBUG [config] JWT{key=[REDACTED], method=HS256, issuer=sentinel-x}
```

## 故障排查

### 问题：环境变量未生效

**解决方案：**

1. 确认环境变量名称正确（大小写敏感）
2. 检查应用名称前缀是否匹配
3. 验证环境变量已正确导出：

   ```bash
   echo $SENTINEL_API_JWT_KEY
   ```

### 问题：JWT 密钥长度不足

**错误信息：**

```text
Error: JWT key must be at least 64 characters for HMAC algorithms
```

**解决方案：**

使用足够长的密钥：

```bash
export SENTINEL_API_JWT_KEY="$(openssl rand -base64 64 | tr -d '\n')"
```

### 问题：数据库连接失败

**解决方案：**

1. 验证数据库凭证是否正确
2. 检查数据库服务是否运行
3. 确认网络连接和防火墙规则
4. 查看详细错误日志

## 参考资料

- [Viper 配置库文档](https://github.com/spf13/viper)
- [Twelve-Factor App - 配置](https://12factor.net/config)
- [OWASP 密钥管理速查表](https://cheatsheetseries.owasp.org/cheatsheets/Key_Management_Cheat_Sheet.html)
