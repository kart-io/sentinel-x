# 网关与服务发现使用指南

## 简介
本指南介绍如何使用 Docker Compose 启动包含 Etcd 和 Traefik 的完整基础设施栈，以及如何验证服务发现功能。

## 组件说明
- **Etcd**: 分布式键值存储，用于服务注册与发现。
- **Traefik**: 云原生网关，自动监听 Etcd 中的服务变化并进行路由配置。

## 快速开始

### 1. 启动服务栈
使用 Docker Compose 启动所有基础设施组件：

```bash
cd deploy
docker-compose up -d
```

这将启动以下服务：
- MySQL (3306)
- Redis (6379)
- Etcd (2379)
- Traefik (80: Web入口, 8080: Dashboard)
- User Center (8081)

### 2. 验证 Etcd 运行状态
确保 Etcd 正常响应：

```bash
ETCDCTL_API=3 etcdctl --endpoints=127.0.0.1:2379 endpoint status
```

或者直接测试连接：
```bash
curl http://127.0.0.1:2379/health
```

### 3. 访问 Traefik 仪表盘
打开浏览器访问 [http://localhost:8080](http://localhost:8080) 查看 Traefik 仪表盘。
你应该能在 HTTP Routers 和 Services 中看到通过 Etcd 注册的服务（如果有）。

### 4. 服务注册示例
Traefik 会监听 Etcd 中的特定前缀（默认通常为 `traefik`）。要手动注册一个服务进行测试：

```bash
# 模拟注册一个服务到 Etcd
docker exec sentinel-etcd etcdctl put traefik/http/routers/my-service/rule "Host(\`example.local\`)"
docker exec sentinel-etcd etcdctl put traefik/http/services/my-service/loadbalancer/servers/0/url "http://user-center:8081"
```

### 5. 配置说明
配置文件位于 `configs/user-center.yaml`，已新增 Etcd 配置：

```yaml
etcd:
  endpoints:
    - "127.0.0.1:2379"
  username: ""
  password: ""
```

## 常见问题

### 端口冲突
如果 80 或 8080 端口被占用，请修改 `deploy/docker-compose.yaml` 中的 Traefik 端口映射。

### Etcd 连接失败
检查 Docker 容器网络，确保服务间可以通过服务名（如 `etcd`）相互访问。
