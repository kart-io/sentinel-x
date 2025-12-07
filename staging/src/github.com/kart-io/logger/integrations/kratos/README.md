# Kratos Integration

Kratos 微服务框架的统一日志器适配，提供完整的结构化日志记录、HTTP 请求追踪、中间件监控和标准库兼容。

## 📋 特性

- ✅ **完整 Kratos 接口**: 实现 Kratos 官方 `log.Logger` 接口
- ✅ **结构化日志**: 支持键值对格式的结构化日志记录
- ✅ **HTTP 请求追踪**: 自动记录 HTTP 请求/响应详情
- ✅ **中间件监控**: 追踪中间件执行时间和状态
- ✅ **日志过滤**: 支持自定义过滤规则
- ✅ **标准库兼容**: 提供标准库 `log` 接口兼容
- ✅ **Helper 支持**: 类似 Kratos 官方的便捷日志方法
- ✅ **零依赖**: 无需引入 Kratos 库即可使用

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "github.com/kart-io/logger/factory"
    "github.com/kart-io/logger/integrations/kratos"
    "github.com/kart-io/logger/option"
    
    // Kratos 相关导入（实际项目中）
    // "github.com/go-kratos/kratos/v2/log"
    // "github.com/go-kratos/kratos/v2/middleware/logging"
)

func main() {
    // 创建统一日志器
    opt := option.DefaultLogOption()
    logger, err := factory.NewLogger(opt)
    if err != nil {
        panic(err)
    }

    // 创建 Kratos 适配器
    kratosLogger := kratos.NewKratosAdapter(logger)

    // 使用 Kratos 日志接口
    kratosLogger.Log(kratos.LevelInfo, "msg", "服务启动", "service", "user-api")
    
    // 创建带有通用字段的子日志器
    serviceLogger := kratosLogger.With("service", "user-api", "version", "1.0.0")
    serviceLogger.Log(kratos.LevelInfo, "msg", "服务初始化完成")
}
```

### Helper 便捷方法

```go
// 创建 Kratos Helper
helper := kratos.NewKratosHelper(logger)

// 使用便捷方法
helper.Info("用户服务启动")
helper.Debug("调试信息")
helper.Warn("连接池即将满载")
helper.Error("数据库连接失败")

// 带参数的日志
helper.Info("用户登录", "user_id", "12345", "ip", "192.168.1.1")
```

## 🔧 日志级别

### Level 定义

```go
const (
    LevelDebug kratos.Level = iota - 1  // 调试信息
    LevelInfo                           // 一般信息
    LevelWarn                           // 警告信息
    LevelError                          // 错误信息
    LevelFatal                          // 致命错误
)
```

### 级别使用

```go
kratosLogger := kratos.NewKratosAdapter(logger)

// 不同级别的日志
kratosLogger.Log(kratos.LevelDebug, "msg", "调试信息", "data", debugData)
kratosLogger.Log(kratos.LevelInfo, "msg", "操作完成", "operation", "create_user")
kratosLogger.Log(kratos.LevelWarn, "msg", "性能警告", "slow_operation", "query") 
kratosLogger.Log(kratos.LevelError, "msg", "操作失败", "error", err.Error())
kratosLogger.Log(kratos.LevelFatal, "msg", "系统崩溃", "panic", panicInfo)
```

## 📊 日志输出示例

### 基础日志

```json
{
  "level": "info",
  "timestamp": "2025-08-30T13:45:30.123456789Z",
  "message": "用户服务启动",
  "component": "kratos",
  "level": "info",
  "service": "user-api",
  "version": "1.0.0"
}
```

### HTTP 请求日志

```json
{
  "level": "info",
  "timestamp": "2025-08-30T13:45:30.234567890Z",
  "message": "HTTP POST /api/users",
  "component": "kratos",
  "operation": "http_request",
  "method": "POST", 
  "path": "/api/users",
  "status_code": 201,
  "duration_ms": 45.67,
  "user_id": "12345"
}
```

### 中间件执行日志

```json
{
  "level": "debug",
  "timestamp": "2025-08-30T13:45:30.345678901Z",
  "message": "Middleware executed",
  "component": "kratos",
  "operation": "middleware",
  "middleware_name": "auth",
  "duration_ms": 1.23
}
```

### 错误日志

```json
{
  "level": "error",
  "timestamp": "2025-08-30T13:45:30.456789012Z",
  "message": "HTTP request failed",
  "component": "kratos", 
  "operation": "http_error",
  "method": "GET",
  "path": "/api/users/999",
  "status_code": 404,
  "error": "user not found"
}
```

## 🎯 高级功能

### 1. 子日志器创建

```go
kratosLogger := kratos.NewKratosAdapter(logger)

// 创建带有持久字段的子日志器
userServiceLogger := kratosLogger.With(
    "service", "user-service",
    "version", "1.2.3",
    "instance_id", "srv-001",
)

// 子日志器会自动包含这些字段
userServiceLogger.Log(kratos.LevelInfo, "msg", "用户创建", "user_id", "123")
// 输出会包含 service, version, instance_id 字段
```

### 2. 日志过滤

```go
// 定义过滤函数
filter := func(level kratos.Level, keyvals ...interface{}) bool {
    // 过滤掉调试级别的日志
    if level == kratos.LevelDebug {
        return false
    }
    
    // 过滤特定操作
    for i := 0; i < len(keyvals); i += 2 {
        if keyvals[i] == "operation" && keyvals[i+1] == "health_check" {
            return false // 过滤掉健康检查日志
        }
    }
    
    return true
}

// 创建带过滤器的日志器
filteredLogger := kratos.NewKratosFilter(logger, filter)
filteredLogger.Log(kratos.LevelDebug, "msg", "调试信息") // 被过滤，不会输出
filteredLogger.Log(kratos.LevelInfo, "msg", "正常信息")  // 正常输出
```

### 3. 标准库兼容

```go
// 创建标准库兼容的日志器
stdLogger := kratos.NewKratosStdLogger(logger)

// 使用标准库接口
stdLogger.Print("这是标准库日志")
stdLogger.Printf("格式化日志: %s", "内容")
stdLogger.Println("带换行的日志")

// 可以用作其他库的日志器
// http.Server{
//     ErrorLog: log.New(stdLogger, "HTTP: ", 0),
// }
```

## 🌐 Web 服务集成

### HTTP 请求日志记录

```go
// 在 HTTP 处理器中记录请求
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
    start := time.Now()
    
    // 处理请求...
    user, err := s.userRepo.Create(ctx, req.User)
    duration := time.Since(start).Nanoseconds()
    
    if err != nil {
        // 记录错误
        s.kratosLogger.LogError(err, "POST", "/api/users", 500)
        return nil, err
    }
    
    // 记录成功请求
    s.kratosLogger.LogRequest("POST", "/api/users", 201, duration, req.User.ID)
    
    return &CreateUserResponse{User: user}, nil
}
```

### 中间件集成

```go
// 认证中间件示例
func AuthMiddleware(kratosLogger kratos.HTTPAdapter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // 执行认证逻辑...
            token := r.Header.Get("Authorization")
            if token == "" {
                kratosLogger.LogError(
                    errors.New("missing authorization header"),
                    r.Method, r.URL.Path, 401,
                )
                http.Error(w, "Unauthorized", 401)
                return
            }
            
            // 记录中间件执行
            duration := time.Since(start).Nanoseconds()
            kratosLogger.LogMiddleware("auth", duration)
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## 🧪 实战示例

### 微服务应用

```go
// main.go - 服务启动
func main() {
    // 创建日志器
    opt := &option.LogOption{
        Engine: "zap",
        Level:  "INFO",
        Format: "json",
        OTLPEndpoint: "http://jaeger:14268/api/traces",
    }
    logger, err := factory.NewLogger(opt)
    if err != nil {
        panic(err)
    }

    // 创建 Kratos 适配器
    kratosLogger := kratos.NewKratosAdapter(logger)
    
    // 创建服务专用日志器
    serviceLogger := kratosLogger.With(
        "service", "user-service",
        "version", "1.0.0",
        "env", "production",
    )
    
    // 初始化服务
    userService := NewUserService(serviceLogger)
    
    // 启动服务
    serviceLogger.Log(kratos.LevelInfo, "msg", "服务启动完成", "port", 8080)
}

// service.go - 业务服务
type UserService struct {
    logger kratos.Logger
    repo   UserRepository
}

func NewUserService(logger kratos.Logger) *UserService {
    return &UserService{
        logger: logger,
        repo:   NewUserRepository(),
    }
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    // 记录开始
    s.logger.Log(kratos.LevelDebug, "msg", "查询用户", "user_id", userID)
    
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        // 记录错误
        s.logger.Log(kratos.LevelError, 
            "msg", "用户查询失败",
            "user_id", userID,
            "error", err.Error(),
        )
        return nil, err
    }
    
    if user == nil {
        // 记录未找到
        s.logger.Log(kratos.LevelWarn, "msg", "用户不存在", "user_id", userID)
        return nil, ErrUserNotFound
    }
    
    // 记录成功
    s.logger.Log(kratos.LevelInfo, 
        "msg", "用户查询成功",
        "user_id", userID,
        "username", user.Username,
    )
    
    return user, nil
}
```

### gRPC 服务集成

```go
// grpc_server.go
type GRPCServer struct {
    logger kratos.Logger
}

func NewGRPCServer(logger kratos.Logger) *GRPCServer {
    return &GRPCServer{
        logger: logger.With("component", "grpc_server"),
    }
}

func (s *GRPCServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    start := time.Now()
    
    // 记录请求开始
    s.logger.Log(kratos.LevelDebug,
        "msg", "gRPC请求开始",
        "method", "CreateUser",
        "request_id", getRequestID(ctx),
    )
    
    // 处理请求
    user, err := s.createUserLogic(ctx, req)
    duration := time.Since(start)
    
    if err != nil {
        // 记录错误
        s.logger.Log(kratos.LevelError,
            "msg", "gRPC请求失败",
            "method", "CreateUser",
            "duration_ms", float64(duration.Nanoseconds())/1e6,
            "error", err.Error(),
        )
        return nil, err
    }
    
    // 记录成功
    s.logger.Log(kratos.LevelInfo,
        "msg", "gRPC请求成功",
        "method", "CreateUser", 
        "duration_ms", float64(duration.Nanoseconds())/1e6,
        "user_id", user.ID,
    )
    
    return &pb.CreateUserResponse{User: user}, nil
}
```

## 📊 性能特征

### 基准测试结果

| 操作类型 | 时延 | 内存分配 | 说明 |
|----------|------|----------|------|
| 基础 Log 调用 | +12ns | +1 alloc | 键值对处理开销 |
| Helper 方法 | +8ns | +1 alloc | 便捷方法优化 |
| 过滤器处理 | +20ns | +2 allocs | 包含过滤逻辑 |
| With 子日志器 | +15ns | +1 alloc | 字段复制开销 |

### 内存优化

- **字段复用**: 高效的键值对处理和复用
- **消息提取**: 智能的消息字段识别和提取
- **级别映射**: 快速的日志级别转换

## 🔗 相关资源

- [Kratos 官方文档](https://go-kratos.dev/)
- [Kratos Log 包文档](https://github.com/go-kratos/kratos/tree/main/log)
- [`integrations`](../README.md) - 集成包总览
- [`core`](../../core/) - 核心接口定义
- [`factory`](../../factory/) - 日志器工厂
- [`example/comprehensive`](../../example/comprehensive/) - 完整使用示例

## ⚠️ 注意事项

### 接口兼容性

1. **键值对格式**: 必须成对出现，奇数个参数会自动补充 nil 值
2. **消息提取**: 自动查找 "msg", "message", "event", "description" 等字段作为消息
3. **类型转换**: 所有值都会转换为适合的字符串表示

### 性能考虑

1. **字段数量**: 大量键值对会增加处理开销
2. **子日志器**: 频繁创建子日志器可能影响性能
3. **过滤器**: 复杂的过滤逻辑会增加延迟

### 使用建议

1. **结构化日志**: 优先使用键值对格式而不是格式化字符串
2. **字段命名**: 使用下划线命名法保持一致性
3. **级别选择**: 根据环境选择合适的日志级别

## 🚀 最佳实践

### 结构化日志记录

```go
// ✅ 推荐：使用结构化键值对
logger.Log(kratos.LevelInfo, 
    "msg", "用户操作",
    "operation", "login",
    "user_id", "12345",
    "ip", "192.168.1.1",
    "user_agent", "Mozilla/5.0...",
)

// ❌ 避免：在消息中嵌入变量信息
logger.Log(kratos.LevelInfo, "msg", "用户 12345 从 192.168.1.1 登录")
```

### 服务标识

```go
// ✅ 推荐：创建带有服务标识的日志器
serviceLogger := kratosLogger.With(
    "service", "user-service",
    "version", os.Getenv("VERSION"),
    "instance", os.Getenv("HOSTNAME"),
    "env", os.Getenv("ENV"),
)
```

### 错误处理

```go
// ✅ 推荐：详细的错误上下文
func (s *Service) HandleRequest(ctx context.Context, req *Request) error {
    if err := s.processRequest(req); err != nil {
        s.logger.Log(kratos.LevelError,
            "msg", "请求处理失败",
            "request_id", getRequestID(ctx),
            "operation", "process_request",
            "error", err.Error(),
            "request_type", req.Type,
        )
        return err
    }
    return nil
}
```

### 监控集成

```go
// ✅ 推荐：添加监控友好的字段
logger.Log(kratos.LevelInfo,
    "msg", "API请求完成",
    "method", "POST",
    "path", "/api/users",
    "status", 201,
    "duration_ms", duration.Milliseconds(),
    "request_size", len(reqBody),
    "response_size", len(respBody),
)
```

选择 Kratos 集成，为你的微服务提供专业的结构化日志记录能力！ 🚀