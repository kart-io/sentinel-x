# ADR-001: 移除 Factory 模式和 Bootstrap 层

> **状态**: 已接受
> **日期**: 2025-12-XX（根据 commit `512fe73`）
> **决策者**: 开发团队
> **更新**: 2026-01-17

---

## 背景

在项目初期，为了实现"灵活性"和"可扩展性"，我们引入了以下设计模式：

1. **Factory 模式**：用于创建服务实例（UserService, AuthService 等）
2. **Bootstrap 层**：统一的服务初始化和依赖注入框架
3. **依赖注入容器**：管理服务间的依赖关系

**初始设计的假设**：
- 未来可能需要多种服务实现（如不同的存储后端）
- 需要在运行时动态切换实现
- 需要复杂的依赖管理

**实际情况**：
- 每个服务只有一个实现
- 从未需要运行时切换实现
- 依赖关系简单且固定
- 增加了代码复杂度和维护成本

---

## 决策

**移除以下组件**：

1. **Factory 模式**：
   - 删除 `internal/user-center/factory/`
   - 删除所有 `NewXXXFactory()` 函数

2. **Bootstrap 层**：
   - 删除 `internal/bootstrap/` 目录
   - 移除统一的初始化框架

3. **依赖注入容器**：
   - 移除 DI 容器相关代码
   - 改为直接在 `app.go` 中初始化

**新的初始化方式**：

```go
// cmd/user-center/app.go
func NewApp(cfg *config.Config) (*App, error) {
    // 1. 初始化数据库
    db, err := mysql.New(cfg.MySQL)
    if err != nil {
        return nil, err
    }

    // 2. 初始化 Store 层
    userStore := store.NewUserStore(db)
    roleStore := store.NewRoleStore(db)

    // 3. 初始化 Biz 层
    userService := biz.NewUserService(userStore)
    authService := biz.NewAuthService(userStore, cfg.JWT)
    roleService := biz.NewRoleService(roleStore, userStore)

    // 4. 初始化 Handler 层
    userHandler := handler.NewUserHandler(userService)
    authHandler := handler.NewAuthHandler(authService)
    roleHandler := handler.NewRoleHandler(roleService)

    // 5. 注册路由
    router := setupRouter(userHandler, authHandler, roleHandler)

    return &App{
        router: router,
        db:     db,
    }, nil
}
```

---

## 理由

### 1. 遵循 YAGNI 原则

**YAGNI (You Aren't Gonna Need It)**：不要实现未来可能需要的功能

- Factory 模式是为了"未来可能的多实现"而设计
- 但实际上从未需要多实现
- 过早的抽象增加了复杂度

### 2. 提高代码可读性

**对比**：

```go
// 旧代码（Factory 模式）
factory := factory.NewUserServiceFactory(cfg)
userService, err := factory.Create()
if err != nil {
    return nil, err
}

// 新代码（直接初始化）
userService := biz.NewUserService(userStore)
```

新代码更直观，依赖关系一目了然。

### 3. 减少维护成本

**移除的代码量**：
- Factory 相关代码：~500 行
- Bootstrap 层代码：~800 行
- 测试代码：~300 行
- **总计**：~1600 行

**维护成本降低**：
- 减少了需要理解的抽象层
- 减少了测试代码
- 减少了文档维护

### 4. 符合项目规范

根据 `CLAUDE.md` 的规范：

> **架构优先级**：
> - "标准化 + 生态复用"拥有最高优先级
> - 禁止新增或维护自研方案，除非已有实践无法满足需求
> - 必须删除自研实现以减少维护面

Factory 模式和 Bootstrap 层是自研的，应该移除。

---

## 后果

### 正面影响

1. **代码更简洁**：
   - 减少了 ~1600 行代码
   - 依赖关系更清晰
   - 新手更容易理解

2. **测试更简单**：
   - 不需要 Mock Factory
   - 直接测试服务实例
   - 减少了测试代码

3. **性能略有提升**：
   - 减少了一层抽象
   - 减少了运行时开销（虽然很小）

### 负面影响

1. **如果未来需要多实现**：
   - 需要重新引入抽象
   - 但可以在需要时再做（YAGNI）

2. **初始化代码略显冗长**：
   - `app.go` 中的初始化代码较长
   - 但依赖关系更清晰

### 迁移成本

- **代码修改**：已完成（commit `512fe73`）
- **测试更新**：已完成
- **文档更新**：进行中

---

## 经验教训

1. **避免过早抽象**：
   - 在有具体需求之前，不要引入复杂的抽象
   - 简单的问题用简单的方案解决

2. **遵循 KISS 原则**：
   - Keep It Simple, Stupid
   - 简洁性优于灵活性

3. **定期审查架构**：
   - 定期评估现有抽象是否仍然必要
   - 勇于删除不必要的代码

4. **数据驱动决策**：
   - 基于实际使用情况做决策
   - 而不是基于"可能需要"

---

## 参考资料

- [Commit 512fe73](https://github.com/kart-io/sentinel-x/commit/512fe73): refactor(user-center): 简化架构，移除Factory模式和bootstrap依赖
- [YAGNI 原则](https://martinfowler.com/bliki/Yagni.html)
- [KISS 原则](https://en.wikipedia.org/wiki/KISS_principle)
- [代码设计分析报告](../../.gemini/antigravity/brain/ea770fd5-aebd-4f6f-a870-ab0b295c39b1/code_design_analysis.md)
