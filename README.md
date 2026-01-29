# Sentinel-X

Sentinel-X 是一个分布式智能运维系统。

## 文档导航

本项目文档位于 `docs/` 目录下：

*   **[快速开始 (Quick Start)](docs/quick-start.md)**: 从零开始运行系统的步骤指南。
*   **[网关架构 (Gateway Architecture)](docs/gateway-architecture.md)**: 了解网关、服务注册与发现机制。
*   **[系统设计 (System Design)](docs/design/architecture.md)**: 了解系统架构与设计理念。
    *   [架构决策记录 (ADR)](docs/design/adr/README.md): 重要架构决策的记录
*   **[配置管理 (Configuration)](docs/configuration/README.md)**: 配置文件说明和环境变量管理。
*   **[性能文档 (Performance)](docs/performance/README.md)**: 性能目标、测试方法和优化历史。
*   **[开发指南 (Development Guide)](docs/development/guide.md)**: 如何参与贡献、代码结构说明及测试方法。
*   **[使用指南 (Usage Guide)](docs/usage/README.md)**: 构建、部署与使用说明。
*   **[API 文档 (API Reference)](docs/api/README.md)**: 接口定义与说明。

## 快速开始

### 环境要求

*   Go 1.25.3 或更高版本

### 构建

编译项目组件：

```bash
go build ./cmd/...
```

## 开发模式

本项目采用类似于 Kubernetes 的 **Monorepo** 结构。核心库 (`goagent`, `logger`) 源码直接包含在 `staging/src/github.com/kart-io/goagent` 和 `staging/src/github.com/kart-io/logger` 目录下。

详细开发流程请参阅 [开发指南](docs/development/guide.md)。

## 许可证

查看 [LICENSE](LICENSE) 文件。
