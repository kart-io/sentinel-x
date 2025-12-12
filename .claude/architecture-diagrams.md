# Sentinel-X 架构图文档

> 生成时间：2025-12-11
> 本文档包含项目的各类架构图（Mermaid 格式）

---

## 一、系统整体架构

```mermaid
flowchart TB
    subgraph Client["客户端"]
        Web["Web 浏览器"]
        Mobile["移动应用"]
        API_Client["API 客户端"]
    end

    subgraph Gateway["网关层"]
        LB["负载均衡"]
    end

    subgraph Services["服务层"]
        API["API Server<br/>:8080/:9090"]
        UC["User Center<br/>:8081/:9091"]
        Scheduler["Scheduler<br/>(待实现)"]
    end

    subgraph Middleware["中间件栈"]
        Recovery["Recovery"]
        RequestID["RequestID"]
        Logger["Logger"]
        Auth["Auth (JWT)"]
        Authz["Authz (Casbin)"]
        Metrics["Metrics"]
        Tracing["Tracing"]
    end

    subgraph Storage["存储层"]
        MySQL[(MySQL)]
        Redis[(Redis)]
        MongoDB[(MongoDB)]
        ETCD[(ETCD)]
    end

    subgraph Observability["可观测性"]
        Jaeger["Jaeger"]
        Prometheus["Prometheus"]
        Logs["日志系统"]
    end

    Client --> LB
    LB --> API
    LB --> UC
    API --> Middleware
    UC --> Middleware
    Middleware --> Storage
    Middleware --> Observability
```

---

## 二、请求处理流程

```mermaid
sequenceDiagram
    participant C as 客户端
    participant M as 中间件栈
    participant H as Handler
    participant B as Biz
    participant S as Store
    participant DB as 数据库

    C->>M: HTTP/gRPC 请求

    Note over M: Recovery
    Note over M: RequestID 生成
    Note over M: Logger 记录
    Note over M: Auth JWT 验证
    Note over M: Authz 权限检查

    M->>H: 转发请求
    H->>H: 参数验证
    H->>B: 调用业务逻辑
    B->>S: 数据操作
    S->>DB: SQL 查询
    DB-->>S: 返回结果
    S-->>B: 返回数据
    B-->>H: 返回结果
    H-->>M: 响应封装

    Note over M: Metrics 记录
    Note over M: Tracing 结束

    M-->>C: HTTP/gRPC 响应
```

---

## 三、模块依赖关系

```mermaid
flowchart TD
    subgraph CMD["cmd/"]
        api["api/main.go"]
        uc["user-center/main.go"]
        sched["scheduler/main.go"]
    end

    subgraph Internal["internal/"]
        api_app["api/app.go"]
        bootstrap["bootstrap/"]
        model["model/"]
        uc_app["user-center/"]
    end

    subgraph PKG["pkg/"]
        component["component/"]
        infra["infra/"]
        security["security/"]
        utils["utils/"]
    end

    subgraph Component["component/"]
        mysql["mysql/"]
        redis["redis/"]
        postgres["postgres/"]
        mongodb["mongodb/"]
        etcd["etcd/"]
    end

    subgraph Infra["infra/"]
        adapter["adapter/"]
        app["app/"]
        datasource["datasource/"]
        logger["logger/"]
        middleware["middleware/"]
        server["server/"]
        tracing["tracing/"]
    end

    subgraph Security["security/"]
        auth["auth/jwt/"]
        authz["authz/casbin/"]
    end

    subgraph Utils["utils/"]
        errors["errors/"]
        id["id/"]
        json["json/"]
        response["response/"]
        validator["validator/"]
    end

    api --> api_app
    uc --> uc_app

    api_app --> bootstrap
    uc_app --> bootstrap
    uc_app --> model

    bootstrap --> infra
    bootstrap --> security
    bootstrap --> component

    uc_app --> utils

    infra --> Component
    security --> Component
```

---

## 四、用户中心分层架构

```mermaid
flowchart TB
    subgraph Router["Router 路由层"]
        R1["POST /api/v1/auth/login"]
        R2["POST /api/v1/auth/register"]
        R3["GET /api/v1/users"]
        R4["GET /api/v1/users/:id"]
    end

    subgraph Handler["Handler 处理层"]
        H1["AuthHandler"]
        H2["UserHandler"]
    end

    subgraph Biz["Biz 业务层"]
        B1["AuthBiz"]
        B2["UserBiz"]
    end

    subgraph Store["Store 存储层"]
        S1["UserStore Interface"]
        S2["MySQLUserStore"]
    end

    subgraph DB["数据库"]
        MySQL[(MySQL)]
    end

    R1 --> H1
    R2 --> H1
    R3 --> H2
    R4 --> H2

    H1 --> B1
    H2 --> B2

    B1 --> S1
    B2 --> S1

    S1 --> S2
    S2 --> MySQL
```

---

## 五、中间件执行顺序

```mermaid
flowchart LR
    subgraph Request["请求方向 →"]
        A["Recovery"] --> B["RequestID"]
        B --> C["Logger"]
        C --> D["CORS"]
        D --> E["Timeout"]
        E --> F["Auth"]
        F --> G["Authz"]
        G --> H["RateLimit"]
        H --> I["Handler"]
    end

    subgraph Response["← 响应方向"]
        I --> J["Metrics"]
        J --> K["Tracing"]
        K --> L["Client"]
    end
```

---

## 六、启动引导流程

```mermaid
flowchart TD
    Start["main.go"] --> NewApp["NewApp()"]
    NewApp --> Run["app.Run()"]

    Run --> Bootstrap["Bootstrap 启动"]

    Bootstrap --> Init1["1. LoggingInitializer"]
    Init1 --> Init2["2. DatasourceInitializer"]
    Init2 --> Init3["3. AuthInitializer"]
    Init3 --> Init4["4. MiddlewareInitializer"]
    Init4 --> Init5["5. ServerInitializer"]

    Init5 --> Start_HTTP["启动 HTTP Server"]
    Init5 --> Start_gRPC["启动 gRPC Server"]

    Start_HTTP --> Listen["监听请求"]
    Start_gRPC --> Listen

    Listen --> Shutdown{"收到关闭信号?"}
    Shutdown -->|是| Graceful["优雅关闭"]
    Shutdown -->|否| Listen

    Graceful --> Stop_HTTP["停止 HTTP"]
    Graceful --> Stop_gRPC["停止 gRPC"]
    Stop_HTTP --> Close_DS["关闭数据源"]
    Stop_gRPC --> Close_DS
    Close_DS --> End["退出"]
```

---

## 七、数据源管理架构

```mermaid
flowchart TB
    subgraph Manager["DataSourceManager"]
        Register["Register()"]
        Get["Get()"]
        Close["Close()"]
    end

    subgraph Clients["数据源客户端"]
        MySQL["MySQLClient"]
        Redis["RedisClient"]
        PostgreSQL["PostgreSQLClient"]
        MongoDB["MongoDBClient"]
    end

    subgraph Health["健康检查"]
        HC["HealthChecker"]
    end

    subgraph Pool["连接池"]
        CP["ConnectionPool"]
    end

    Register --> MySQL
    Register --> Redis
    Register --> PostgreSQL
    Register --> MongoDB

    MySQL --> Pool
    Redis --> Pool
    PostgreSQL --> Pool
    MongoDB --> Pool

    Get --> MySQL
    Get --> Redis
    Get --> PostgreSQL
    Get --> MongoDB

    MySQL --> HC
    Redis --> HC
    PostgreSQL --> HC
    MongoDB --> HC
```

---

## 八、认证授权流程

```mermaid
sequenceDiagram
    participant C as 客户端
    participant Auth as Auth 中间件
    participant JWT as JWT Service
    participant Authz as Authz 中间件
    participant Casbin as Casbin Enforcer
    participant H as Handler

    C->>Auth: 请求 + Token
    Auth->>JWT: 验证 Token

    alt Token 有效
        JWT-->>Auth: 用户信息
        Auth->>Authz: 传递用户信息
        Authz->>Casbin: 检查权限(user, resource, action)

        alt 有权限
            Casbin-->>Authz: 允许
            Authz->>H: 继续处理
            H-->>C: 正常响应
        else 无权限
            Casbin-->>Authz: 拒绝
            Authz-->>C: 403 Forbidden
        end
    else Token 无效
        JWT-->>Auth: 验证失败
        Auth-->>C: 401 Unauthorized
    end
```

---

## 九、错误处理流程

```mermaid
flowchart TD
    subgraph Source["错误来源"]
        S1["参数验证错误"]
        S2["业务逻辑错误"]
        S3["数据库错误"]
        S4["外部服务错误"]
    end

    subgraph ErrorPkg["pkg/utils/errors"]
        Builder["ErrorBuilder"]
        Registry["ErrorRegistry"]
        Errno["Errno 定义"]
    end

    subgraph Response["响应处理"]
        Writer["ResponseWriter"]
        Format["统一格式"]
    end

    S1 --> Builder
    S2 --> Builder
    S3 --> Builder
    S4 --> Builder

    Builder --> Registry
    Registry --> Errno

    Errno --> Writer
    Writer --> Format

    Format --> JSON["JSON 响应<br/>{code, message, data}"]
```

---

## 十、包结构概览

```mermaid
mindmap
  root((sentinel-x))
    cmd
      api
      user-center
      scheduler
    internal
      api
      bootstrap
      model
      user-center
        handler
        biz
        store
        router
    pkg
      component
        mysql
        redis
        postgres
        mongodb
        etcd
      infra
        adapter
        app
        datasource
        logger
        middleware
        server
        tracing
      security
        auth/jwt
        authz/casbin
      utils
        errors
        id
        json
        response
        validator
    configs
    docs
    staging
      logger
```

---

## 十一、部署架构

```mermaid
flowchart TB
    subgraph Internet["互联网"]
        Users["用户"]
    end

    subgraph K8s["Kubernetes 集群"]
        subgraph Ingress["Ingress 层"]
            Nginx["Nginx Ingress"]
        end

        subgraph Services["服务层"]
            API1["API Server Pod 1"]
            API2["API Server Pod 2"]
            UC1["User Center Pod 1"]
            UC2["User Center Pod 2"]
        end

        subgraph Middleware["中间件层"]
            Redis_Cluster["Redis Cluster"]
            MySQL_Primary["MySQL Primary"]
            MySQL_Replica["MySQL Replica"]
        end

        subgraph Observability["可观测性"]
            Jaeger_Agent["Jaeger Agent"]
            Prometheus_Server["Prometheus"]
            Grafana["Grafana"]
        end
    end

    Users --> Nginx
    Nginx --> API1
    Nginx --> API2
    Nginx --> UC1
    Nginx --> UC2

    API1 --> Redis_Cluster
    API2 --> Redis_Cluster
    UC1 --> Redis_Cluster
    UC2 --> Redis_Cluster

    API1 --> MySQL_Primary
    API2 --> MySQL_Replica
    UC1 --> MySQL_Primary
    UC2 --> MySQL_Replica

    API1 --> Jaeger_Agent
    API2 --> Jaeger_Agent
    UC1 --> Jaeger_Agent
    UC2 --> Jaeger_Agent

    Prometheus_Server --> API1
    Prometheus_Server --> API2
    Prometheus_Server --> UC1
    Prometheus_Server --> UC2
```

---

## 使用说明

以上架构图均使用 Mermaid 语法编写，可以：

1. 在 GitHub/GitLab 中直接渲染
2. 使用 VS Code 的 Mermaid 插件预览
3. 通过 [Mermaid Live Editor](https://mermaid.live/) 在线编辑
4. 集成到 Markdown 文档中使用
