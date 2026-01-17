# 配置管理文档

> **更新时间**: 2026-01-17
> **适用版本**: Sentinel-X v1.0+

---

## 概述

Sentinel-X 使用 **YAML 配置文件 + 环境变量** 的混合配置方式：
- **YAML 文件**：存储非敏感的默认配置
- **环境变量**：存储敏感信息（密码、API Key）和环境特定配置

### 配置优先级

```
环境变量 > YAML 配置文件 > 默认值
```

---

## 配置文件结构

### 配置文件位置

```
configs/
├── user-center.yaml          # User Center 生产配置
├── user-center-dev.yaml      # User Center 开发配置
├── rag.yaml                  # RAG Service 生产配置
├── sentinel-api.yaml         # API Server 生产配置
├── sentinel-api-dev.yaml     # API Server 开发配置
└── auth.yaml                 # 认证配置（独立）
```

### 配置文件命名规范

- **生产配置**: `<service-name>.yaml`
- **开发配置**: `<service-name>-dev.yaml`
- **测试配置**: `<service-name>-test.yaml`

---

## 环境变量配置

### 环境变量命名规则

```
<APP_NAME>_<CONFIG_KEY>
```

- **APP_NAME**: 服务名称，全大写，使用下划线分隔
  - `USER_CENTER`
  - `SENTINEL_API`
  - `RAG` (RAG Service 没有前缀，直接使用配置键)

- **CONFIG_KEY**: 配置键路径，全大写，使用下划线分隔
  - YAML 中的点号 (`.`) 和横杠 (`-`) 都转换为下划线 (`_`)

### 示例

```yaml
# user-center.yaml
jwt:
  key: ""
  signing-method: "HS256"

mysql:
  host: "localhost"
  password: ""
```

对应的环境变量：
```bash
USER_CENTER_JWT_KEY="your-secret-key"
USER_CENTER_JWT_SIGNING_METHOD="HS256"
USER_CENTER_MYSQL_HOST="localhost"
USER_CENTER_MYSQL_PASSWORD="your-password"
```

---

## 敏感信息管理

### ⚠️ 安全原则

1. **绝对不要**在 YAML 配置文件中硬编码敏感信息
2. **必须**通过环境变量设置敏感信息
3. **必须**将 `.env` 文件添加到 `.gitignore`
4. **建议**使用密钥管理服务（Vault、AWS Secrets Manager）

### 敏感信息清单

| 配置项 | 环境变量 | 说明 |
|--------|---------|------|
| JWT 密钥 | `USER_CENTER_JWT_KEY` | 最少 64 字符 |
| MySQL 密码 | `USER_CENTER_MYSQL_PASSWORD` | 强密码 |
| Redis 密码 | `USER_CENTER_REDIS_PASSWORD` | 如果启用认证 |
| DeepSeek API Key | `DEEPSEEK_API_KEY` | RAG Chat Provider |
| OpenAI API Key | `OPENAI_API_KEY` | 如果使用 OpenAI |
| Milvus 密码 | `MILVUS_PASSWORD` | 如果启用认证 |

### 生成强密钥

```bash
# 生成 JWT 密钥（64 字节 base64 编码）
openssl rand -base64 64 | tr -d '\n'

# 生成随机密码（32 字符）
openssl rand -base64 32 | tr -d '\n'
```

---

## 配置文件说明

### User Center 配置

**文件**: `configs/user-center.yaml`

**核心配置项**:

```yaml
# 服务器配置
server:
  mode: both              # http, grpc, or both
  shutdown-timeout: 30s

# HTTP 服务器
http:
  addr: ":8081"
  adapter: gin            # gin or echo

# JWT 认证
jwt:
  disable-auth: false     # 生产环境必须为 false
  key: ""                 # 通过 USER_CENTER_JWT_KEY 设置
  signing-method: "HS256"
  expired: 2h
  max-refresh: 24h

# MySQL 数据库
mysql:
  host: "localhost"       # 通过 USER_CENTER_MYSQL_HOST 设置
  port: 3306
  username: "root"        # 通过 USER_CENTER_MYSQL_USERNAME 设置
  password: ""            # 通过 USER_CENTER_MYSQL_PASSWORD 设置
  database: "user_auth"

# Redis 缓存
redis:
  host: "localhost"
  port: 6379
  password: ""            # 通过 USER_CENTER_REDIS_PASSWORD 设置
  database: 0
```

### RAG Service 配置

**文件**: `configs/rag.yaml`

**核心配置项**:

```yaml
# Milvus 向量数据库
milvus:
  address: "localhost:19530"
  database: "default"
  username: "root"
  password: ""            # 通过 MILVUS_PASSWORD 设置

# Embedding Provider
embedding:
  provider: "ollama"      # ollama, openai
  base-url: "http://localhost:11434"
  api-key: ""             # 如果使用 OpenAI: OPENAI_API_KEY
  model: "nomic-embed-text"

# Chat Provider
chat:
  provider: "deepseek"    # ollama, openai, deepseek
  base-url: "https://api.deepseek.com"
  api-key: ""             # 通过 DEEPSEEK_API_KEY 设置
  model: "deepseek-chat"

# RAG 参数
rag:
  chunk-size: 512
  chunk-overlap: 50
  top-k: 5
  collection: "milvus_docs"
  embedding-dim: 768

  # 查询增强（可配置）
  enhancer:
    enable-query-rewrite: false
    enable-hyde: false
    enable-rerank: false
```

---

## 环境变量配置文件

### .env 文件

**位置**: 项目根目录 `.env`

**说明**:
- 从 `.env.example` 复制创建
- 已在 `.gitignore` 中，不会提交到版本控制
- 仅用于本地开发

**创建步骤**:

```bash
# 1. 复制示例文件
cp .env.example .env

# 2. 编辑 .env 文件，填入真实值
vim .env

# 3. 加载环境变量（可选，应用会自动加载）
source .env
```

**示例内容**:

```bash
# User Center
USER_CENTER_JWT_KEY="your-generated-jwt-key-here"
USER_CENTER_MYSQL_PASSWORD="your-mysql-password"
USER_CENTER_REDIS_PASSWORD="your-redis-password"

# RAG Service
DEEPSEEK_API_KEY="sk-your-deepseek-api-key"
MILVUS_PASSWORD="your-milvus-password"
```

---

## 不同环境的配置管理

### 开发环境

```bash
# 使用开发配置文件
export CONFIG_FILE=configs/user-center-dev.yaml

# 或者通过命令行参数
./bin/user-center --config=configs/user-center-dev.yaml
```

### 测试环境

```bash
# 使用测试环境变量
export USER_CENTER_MYSQL_HOST=test-mysql.example.com
export USER_CENTER_REDIS_HOST=test-redis.example.com
```

### 生产环境

**方案 1: Kubernetes Secret**

```yaml
# deploy/k8s/user-center-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: user-center-secrets
type: Opaque
data:
  jwt-key: <base64-encoded-key>
  mysql-password: <base64-encoded-password>
```

```yaml
# deploy/k8s/user-center-deployment.yaml
env:
  - name: USER_CENTER_JWT_KEY
    valueFrom:
      secretKeyRef:
        name: user-center-secrets
        key: jwt-key
  - name: USER_CENTER_MYSQL_PASSWORD
    valueFrom:
      secretKeyRef:
        name: user-center-secrets
        key: mysql-password
```

**方案 2: HashiCorp Vault**

```bash
# 从 Vault 读取密钥
export USER_CENTER_JWT_KEY=$(vault kv get -field=jwt_key secret/sentinel-x/user-center)
export USER_CENTER_MYSQL_PASSWORD=$(vault kv get -field=mysql_password secret/sentinel-x/user-center)
```

---

## 配置验证

### 启动时验证

应用启动时会验证必需的配置项：

```go
// 必需的配置项
required := []string{
    "jwt.key",
    "mysql.password",
}

// 验证失败会导致启动失败
if err := validateConfig(cfg, required); err != nil {
    log.Fatal("配置验证失败", err)
}
```

### 手动验证

```bash
# 检查环境变量是否设置
env | grep USER_CENTER

# 检查配置文件语法
yamllint configs/user-center.yaml
```

---

## 配置最佳实践

### 1. 分离敏感信息

✅ **正确做法**:
```yaml
# configs/user-center.yaml
mysql:
  password: ""  # 空字符串，通过环境变量设置
```

```bash
# .env
USER_CENTER_MYSQL_PASSWORD="actual-password"
```

❌ **错误做法**:
```yaml
# configs/user-center.yaml
mysql:
  password: "hardcoded-password"  # 不要这样做！
```

### 2. 使用配置模板

```yaml
# configs/user-center.yaml.example
mysql:
  host: "localhost"
  password: ""  # 必须通过环境变量设置: USER_CENTER_MYSQL_PASSWORD
```

### 3. 文档化配置项

在配置文件中添加注释说明：

```yaml
# JWT 认证密钥（最少 64 个字符，用于 HMAC 算法）
# ⚠️ 安全警告：绝对不要在此文件中硬编码密钥！
# 必须通过环境变量设置: USER_CENTER_JWT_KEY
# 生成密钥: openssl rand -base64 64 | tr -d '\n'
jwt:
  key: ""
```

### 4. 定期轮换密钥

```bash
# 建议每 90 天轮换一次
# 1. 生成新密钥
NEW_KEY=$(openssl rand -base64 64 | tr -d '\n')

# 2. 更新环境变量
export USER_CENTER_JWT_KEY="$NEW_KEY"

# 3. 重启服务
kubectl rollout restart deployment/user-center
```

### 5. 使用配置版本控制

```yaml
# configs/user-center.yaml
# Version: 1.2.0
# Last Updated: 2026-01-17
# Changes:
#   - 添加 JWT max-refresh 配置
#   - 更新 Redis 连接池大小
```

---

## 故障排查

### 问题 1: 配置未生效

**症状**: 修改了配置文件，但服务行为未改变

**解决方案**:
1. 检查是否重启了服务
2. 检查环境变量是否覆盖了配置文件
3. 检查配置文件路径是否正确

```bash
# 查看服务使用的配置文件
ps aux | grep user-center

# 查看环境变量
env | grep USER_CENTER
```

### 问题 2: 启动失败 - 配置缺失

**症状**: 服务启动失败，提示 "required config missing"

**解决方案**:
1. 检查必需的环境变量是否设置
2. 检查配置文件是否存在
3. 查看启动日志获取详细错误信息

```bash
# 检查必需的环境变量
echo $USER_CENTER_JWT_KEY
echo $USER_CENTER_MYSQL_PASSWORD

# 查看启动日志
./bin/user-center 2>&1 | grep -i error
```

### 问题 3: 数据库连接失败

**症状**: 服务启动失败，提示数据库连接错误

**解决方案**:
1. 检查数据库配置（host, port, username, password）
2. 检查数据库是否运行
3. 检查网络连接

```bash
# 测试数据库连接
mysql -h $USER_CENTER_MYSQL_HOST -u $USER_CENTER_MYSQL_USERNAME -p

# 检查网络连接
telnet $USER_CENTER_MYSQL_HOST 3306
```

---

## 参考资料

- [.env.example](../../.env.example) - 环境变量配置示例
- [架构设计文档](architecture.md) - 系统架构说明
- [部署指南](../usage/README.md) - 部署和运维指南
