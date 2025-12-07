# Sentinel-X

Sentinel-X 是一个分布式智能运维系统。

## 文档导航

本项目文档位于 `docs/` 目录下：

*   **[系统设计 (System Design)](docs/design/architecture.md)**: 了解系统架构与设计理念。
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

本项目采用类似于 Kubernetes 的 **Monorepo** 结构。核心代理库 (`goagent`) 源码直接包含在 `staging/src/github.com/kart-io/goagent` 目录下。

详细开发流程请参阅 [开发指南](docs/development/guide.md)。

## 许可证

查看 [LICENSE](LICENSE) 文件。
