# Store Query Helper (数据库查询助手)

`pkg/store` 包提供了一套用于简化 GORM 数据库查询操作的工具库，旨在通过链式调用（Fluent Interface）的方式，让查询条件的构建更加直观、简洁且易于维护。

## 核心功能 (Features)

- **链式调用 (Fluent Interface)**: 支持通过链式方法流畅地构建查询条件。
- **分页支持 (Pagination)**: 提供 `P` (Page/PageSize), `L` (Limit), `O` (Offset) 方法，轻松处理分页逻辑。
- **查询过滤 (Filtering)**:
  - `F`: 简单的 Key-Value Map 过滤。
  - `Q`: 支持原生 SQL 条件（如 `price > ?`）。
  - `C`: 支持 GORM 的原生 `clause` 表达式。
- **多租户支持 (Multi-tenancy)**: 提供 `T` 方法，结合全局注册的租户获取逻辑，自动注入租户隔离条件。

## 快速开始 (Quick Start)

### 1. 基础用法

```go
import "github.com/kart-io/sentinel-x/pkg/store"

// 创建查询选项：第 1 页，每页 10 条，且 status = 'active'
opts := store.NewWhere().P(1, 10).F("status", "active")

// 应用到 GORM DB 并查询
var users []User
opts.Where(db).Find(&users)
```

### 2. 快捷函数

为了进一步简化代码，包级函数（如 `store.P`, `store.F` 等）可以直接作为构建器的起始点：

```go
// 直接从分页开始构建
store.P(1, 10).F("role", "admin").Where(db).Find(&users)
```

### 3. 复杂查询组合

支持多种条件的自由组合：

```go
import (
    "github.com/kart-io/sentinel-x/pkg/store"
    "gorm.io/gorm/clause"
)

store.NewWhere().
    P(1, 20).                     // 分页: 第1页, 20条
    F("category", "books").       // 精确匹配: category = 'books'
    Q("price > ?", 100).          // 自定义查询: price > 100
    C(clause.OrderBy{             // 自定义 Clause: 按创建时间倒序
        Expression: clause.Expr{SQL: "created_at DESC"},
    }).
    Where(db).
    Find(&items)
```

### 4. 多租户支持 (Multi-tenancy)

如果系统需要支持多租户数据隔离，可以全局注册租户键和获取逻辑：

**初始化（通常在应用启动阶段）:**

```go
store.RegisterTenant("tenant_id", func(ctx context.Context) string {
    // 从 context 中获取当前请求的 tenant_id
    // 例如从 JWT Claims 或 Context Value 中获取
    return GetTenantIDFromContext(ctx)
})
```

**业务中使用:**

```go
func ListData(ctx context.Context) {
    // T(ctx) 会自动调用注册的函数获取值，并添加 WHERE tenant_id = '...' 条件
    store.T(ctx).Where(db).Find(&data)
}
```

## API 参考 (API Reference)

### 构建函数

- `NewWhere()`: 创建一个新的 `Options` 构建器。
- `P(page, pageSize)`: 创建构建器并设置分页。
- `L(limit)`: 创建构建器并设置 Limit。
- `O(offset)`: 创建构建器并设置 Offset。
- `F(kvs...)`: 创建构建器并设置 Key-Value 过滤条件。
- `Q(query, args...)`: 创建构建器并设置 SQL 查询条件。
- `C(conds...)`: 创建构建器并设置 GORM Clauses。
- `T(ctx)`: 创建构建器并设置租户过滤。

### 链式方法

- `.P(page, pageSize)`: 设置分页 (Page 1-based)。
- `.L(limit)`: 设置 Limit。
- `.O(offset)`: 设置 Offset。
- `.F(kvs...)`: 添加 Key-Value 过滤条件 (支持多个键值对)。
- `.Q(query, args...)`: 添加 SQL 查询条件 (如 `Q("age > ?", 18)`).
- `.C(conds...)`: 添加 GORM Clauses.
- `.T(ctx)`: 添加租户过滤条件.
- `.Where(db *gorm.DB)`: 将构建好的条件应用到 GORM DB 实例上，返回 `*gorm.DB`.
