# Sentinel-X 可维护性快速检查清单

**用途**: 在代码审查和开发时快速参考  
**更新频率**: 月度更新  

---

## 编码前清单

### 1. 模块设计
- [ ] 该功能是否应该存在于现有模块中，或需要创建新模块？
  - 查看 `/internal` 和 `/pkg` 的既有结构
  - 避免创建新的单一函数文件
- [ ] 该功能的输入输出是否清晰定义？
- [ ] 是否有现有的类似功能可复用？

### 2. 命名规范
```go
✓ 好的命名
- HandleUserCreation()           // 清晰的意图
- ValidateEmail()                // 动词+名词
- userRepository                 // 小驼峰
- ErrUserNotFound                // 错误以 Err 前缀
- Config struct with fields      // 配置结构体

✗ 应避免的命名
- Do()                           // 太模糊
- temp, msg, data               // 过于通用
- userrepo, Userrepo            // 大小写混乱
- error_user_not_found           // 下划线
```

### 3. 函数长度
- [ ] 函数长度是否超过 50 行？
  - 超过 50 行 → 考虑拆分
  - handler 方法 20-40 行是合理的
  - 工具函数 10-20 行较佳

### 4. 错误处理
- [ ] 是否处理了所有可能的错误路径？
  - 查看关键操作（数据库、网络、文件）
  - 使用 `errors.New()` 创建带上下文的错误
  - 避免 `panic()` 除非是不可恢复的启动错误

---

## 开发中清单

### 1. 测试
```go
// 对于每个新函数，至少编写：
- [ ] 正常路径测试
- [ ] 边界条件测试（空值、零值、极限值）
- [ ] 错误路径测试
- [ ] 并发测试（如涉及并发）

示例：
func TestUserService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        wantErr bool
    }{
        {name: "正常情况", input: validData, wantErr: false},
        {name: "输入为空", input: nil, wantErr: true},
        {name: "字段缺失", input: incompleteData, wantErr: true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### 2. 注释编写
```go
✓ 有效的注释
// UserService 处理所有用户相关的业务逻辑，包括创建、查询、更新、删除。
// 依赖 UserStore 进行数据访问，使用 bcrypt 进行密码哈希。
type UserService struct {...}

// Create 创建新用户并自动对密码进行哈希处理。
// 如果用户名或邮箱已存在，返回 ErrAlreadyExists。
// 密码长度必须在 8-72 字符之间（bcrypt 限制）。
func (s *UserService) Create(ctx context.Context, user *model.User) error {

✗ 无效的注释
// 创建用户            // 重复代码，无额外价值
// user = nil          // 说明代码本身已说明的内容
// TODO: 需要优化性能   // 过于模糊，无上下文
```

### 3. 代码重复检查
```go
// 如果写了相似的代码 3 次以上，考虑提取为公共函数

// 不好：重复 5 次
func (h *UserHandler) handleError(c Context, err error) {
    resp := response.Err(...)
    defer response.Release(resp)
    c.JSON(resp.HTTPStatus(), resp)
}

// 好：抽象公共方法
func (h *UserHandler) sendError(c Context, err error) {
    resp := response.Err(...)
    defer response.Release(resp)
    c.JSON(resp.HTTPStatus(), resp)
}
```

---

## 代码审查清单

### 关键检查点

#### Security (安全性)
- [ ] 是否有硬编码的密钥或敏感信息？
- [ ] 是否验证了所有用户输入？
- [ ] 是否检查了权限（if 需要）？
- [ ] 是否使用了安全的密码处理（bcrypt, not plain）？
- [ ] 是否有 SQL 注入风险？（使用参数化查询）

#### Correctness (正确性)
- [ ] 错误处理是否完整？
- [ ] 边界条件是否处理？
- [ ] 是否有 nil 指针的风险？
- [ ] 并发访问是否安全？
- [ ] 是否有内存泄漏？

#### Readability (可读性)
- [ ] 变量命名是否清晰？
- [ ] 函数长度是否合理？
- [ ] 注释是否准确？
- [ ] 代码流程是否易理解？

#### Performance (性能)
- [ ] 是否有 N+1 查询问题？
- [ ] 是否有不必要的内存分配？
- [ ] 是否有不必要的 goroutine？
- [ ] 是否有阻塞操作在热路径？

#### Maintainability (可维护性)
- [ ] 是否重复了现有代码？
- [ ] 是否遵循项目约定？
- [ ] 是否添加了测试？
- [ ] 是否更新了文档？

---

## 常见问题速查

### Handler 层

```go
// 问题 1: 重复的参数验证
❌ func (h *Handler) Get(c Context) {
    param := c.Param("id")
    if param == "" {
        // 错误处理...
        return
    }
}

✓ func (h *Handler) GetParam(c Context, name string) (string, error) {
    val := c.Param(name)
    if val == "" {
        return "", ErrBadRequest
    }
    return val, nil
}

// 问题 2: 响应释放管理混乱
❌ resp := response.Success(data)
   defer response.Release(resp)
   c.JSON(200, resp)  // c.JSON 可能也释放

✓ resp := response.Success(data)
   c.JSON(200, resp)  // 让 c.JSON 负责释放，Handler 不管

// 问题 3: 日志级别错误
❌ logger.Error("User not found")  // 不是系统错误
✓ logger.Info("User not found")    // 正常业务流程
```

### Service 层

```go
// 问题 1: 缺少输入验证
❌ func (s *Service) CreateUser(ctx Context, user *User) error {
    return s.store.Create(ctx, user)  // 没有验证 user
}

✓ func (s *Service) CreateUser(ctx Context, user *User) error {
    if err := s.validateUser(user); err != nil {
        return err
    }
    return s.store.Create(ctx, user)
}

// 问题 2: 错误信息过于详细
❌ return fmt.Errorf("failed to query user from database: %v", dbErr)

✓ return errors.ErrDatabase.WithCause(dbErr)  // 使用定义的错误类型
```

### Store 层

```go
// 问题 1: 线程安全隐患
❌ var factory Factory
   var once sync.Once
   
   once.Do(func() {
       factory = NewFactory()  // 竞态条件
   })
   if factory == nil { ... }

✓ var (
       result atomic.Value  // stores *factoryResult
       once   sync.Once
   )
   
   once.Do(func() {
       result.Store(&factoryResult{factory: f, err: err})
   })
   r := result.Load().(*factoryResult)
   return r.factory, r.err

// 问题 2: SQL 参数化不足
❌ query := fmt.Sprintf("SELECT * FROM users WHERE id = %d", id)

✓ query := "SELECT * FROM users WHERE id = ?"
   db.Query(query, id)
```

### Utils 和 工具函数

```go
// 问题 1: 缺少输入验证
❌ func Page(list interface{}, total int64, page, pageSize int) *Response {
    totalPages := int(total) / pageSize  // pageSize=0 会 panic
}

✓ func Page(list interface{}, total int64, page, pageSize int) (*Response, error) {
    if pageSize <= 0 || pageSize > 1000 {
        return nil, ErrBadRequest
    }
    // ...
}

// 问题 2: 对象池数据泄露
❌ func Release(r *Response) {
    r.Data = nil  // 如果 Data 含有敏感信息仍可能泄露
    pool.Put(r)
}

✓ func Release(r *Response) {
    // 深层清空所有引用
    clearSensitiveData(r.Data)
    r.Data = nil
    r.pooled = true
    pool.Put(r)
}
```

---

## Git 提交清单

在提交前，检查：

```bash
# 1. 代码格式
[ ] make fmt && make lint 无错误
[ ] 没有 TODO/FIXME 注释（除非已记录为 Issue）

# 2. 测试
[ ] 新增功能有单元测试
[ ] 所有测试通过: make test
[ ] 测试覆盖率未下降

# 3. 文档
[ ] 新增 public API 有注释
[ ] 如有 breaking change，已更新 README
[ ] 复杂逻辑有设计文档

# 4. 依赖
[ ] 没有引入不必要的新依赖
[ ] 版本号锁定在 go.mod
[ ] 运行 go mod tidy

# 5. 提交信息
[ ] 提交信息清晰说明改动内容
[ ] 遵循 conventional commit 格式
  feat: 新功能
  fix: 错误修复
  refactor: 重构
  test: 添加测试
  docs: 文档更新
```

---

## 性能基准

| 指标 | 目标 | 检查方法 |
|------|------|--------|
| 单个 API 请求 | < 100ms | 本地压测 `ab -c 100 -n 10000` |
| 数据库查询 | < 50ms | 查看 slow query log |
| 内存使用 | 稳定增长 | 长期运行监控 |
| Goroutine 泄漏 | 无增长 | runtime.NumGoroutine() |

---

## 月度审查清单

每个月进行一次：

```markdown
## 2025年12月可维护性月度审查

### 代码质量
- [ ] 是否有新增的重复代码？
- [ ] 是否有新增的超长函数（>100行）？
- [ ] 是否有未处理的 TODO？

### 测试
- [ ] 测试覆盖率是否下降？
- [ ] 是否有新的测试失败？
- [ ] 是否有新的 flaky 测试？

### 依赖
- [ ] 是否有已知的安全漏洞？
- [ ] 依赖数量是否持续增长？
- [ ] 是否有已弃用的依赖？

### 性能
- [ ] 是否有新的性能回归？
- [ ] 内存占用是否异常增长？
- [ ] 是否有新的 goroutine 泄漏？

### 文档
- [ ] 是否有新增的 API 但无文档？
- [ ] README 是否需要更新？
- [ ] 架构文档是否需要更新？

行动项:
- [ ] Item 1
- [ ] Item 2
```

---

## 反模式速查

### 常见反模式及改进

```go
// 1. God Object（上帝对象）
❌ type UserService struct {
    db *gorm.DB
    cache *redis.Client
    logger *log.Logger
    email *EmailService
    sms *SMSService
    payment *PaymentService
    // 做太多事情
}

✓ type UserService struct {
    repo UserRepository
    notifier UserNotifier
}

// 2. 循环导入
❌ package A import "B"
   package B import "A"

✓ 使用接口解耦，A 依赖 interface，B 实现 interface

// 3. 过度使用 interface
❌ type Reader interface { Read() []byte }
   type Writer interface { Write([]byte) error }
   type Closer interface { Close() error }
   // ... 50 个单函数接口

✓ type IO interface {
    Read() []byte
    Write([]byte) error
    Close() error
}

// 4. 在 init() 中做太多事情
❌ func init() {
    db = setupDatabase()        // 连接数据库
    cache = setupCache()        // 连接缓存
    logger = setupLogger()      // 初始化日志
    // 难以测试，难以控制顺序
}

✓ // 使用 Bootstrap 模式
bootstrapper := NewBootstrapper(config)
bootstrapper.Initialize(ctx)
```

---

## 快速查询表

### 文件组织
```
internal/
├── bootstrap/        启动逻辑（依赖注入、初始化）
├── model/           数据库模型（GORM）
├── domain/          领域模型、业务规则
├── user-center/
│   ├── biz/         业务逻辑层（service）
│   ├── handler/     HTTP 处理层
│   ├── store/       数据访问层（repository）
│   ├── router/      路由定义
│   └── dto/         数据传输对象
└── [其他模块]

pkg/
├── infra/           基础设施（服务器、中间件）
├── security/        认证、授权
├── utils/           工具函数
├── options/         配置结构体
└── component/       通用组件（数据库客户端等）
```

### 接口隔离原则
```go
// ❌ 不好：依赖太多
type UserHandler struct {
    userSvc UserService
    emailSvc EmailService
    smsSvc SMSService
    ...
}

// ✓ 好：仅依赖需要的接口
type UserHandler struct {
    userSvc UserService
}

type UserService interface {
    GetUser(ctx, username) (*User, error)
    UpdateUser(ctx, user) error
    // ...
}
```

---

