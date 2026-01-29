# 快速开始指南

本指南将帮助你快速在本地启动 Sentinel-X 系统。

## 前置要求

*   [Docker](https://www.docker.com/) & Docker Compose
*   [Go](https://go.dev/) 1.25+ (用于本地运行服务)
*   [Curl](https://curl.se/) (用于测试)

## 启动步骤

### 第一步：启动基础设施

使用 Docker Compose 启动 MySQL、Redis、ETCD 和 Traefik 网关。

> **注意**：我们只启动依赖组件，不启动 `user-center` 容器，以便我们在本地运行代码进行开发调试。

```bash
cd deploy
docker-compose up -d mysql redis etcd traefik
```

等待几秒钟，确保所有容器状态为 `Up`：

```bash
docker-compose ps
```

### 第二步：运行用户中心服务

回到项目根目录，在本地运行用户中心服务。

1.  **设置环境变量**（覆盖配置文件中的 localhost 设置，或者确保 hosts 映射）：
    如果是本地运行且 Docker 端口已映射到 localhost，默认配置即可工作。

2.  **运行服务**：

```bash
# 确保在项目根目录
go run cmd/user-center/main.go
```

或者使用 Makefile（如果有）：

```bash
make run-dev
```

服务启动后，你应该能看到日志显示服务已注册到 ETCD。

### 第三步：验证服务

通过网关（Traefik，监听端口 80）访问服务，验证整条链路是否打通。

**检查健康状态**：

```bash
curl -v http://localhost:80/sentinel/user-center/health
```

**预期输出**：
HTTP 200 OK，并返回 JSON 格式的健康状态信息。

### 常用操作

*   **查看 Traefik 仪表盘**：访问 [http://localhost:8080](http://localhost:8080) 查看路由和服务状态。
*   **停止环境**：
    ```bash
    cd deploy && docker-compose down
    ```
