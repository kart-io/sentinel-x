# Protobuf 参数验证指南

本项目使用 `protoc-gen-validate` (PGV) 插件直接在 `.proto` 文件中定义参数验证规则。这种方式确立了 API 定义文件（Schema）作为“单一事实来源（Single Source of Truth）”的地位，保证了 gRPC 和 HTTP 接口验证逻辑的一致性。

此外，为了提供更灵活的验证能力，我们的框架支持 `protoc-gen-validate` 与 `go-playground/validator` (Struct Tag) 混合使用。

## 1. 在 Proto 中定义验证规则

在编写 `.proto` 文件时，可以通过 `validate.rules` 选项为字段添加约束。

### 引入依赖

首先确保你的 proto 文件头部引入了 `validate/validate.proto`：

```protobuf
syntax = "proto3";

package api.user.v1;

import "validate/validate.proto"; // 必须引入
```

### 常用规则示例

#### 字符串 (String)
*   **min_len / max_len**: 最小/最大长度（字符数）
*   **pattern**: 正则表达式
*   **email**: 邮箱格式
*   **ipv4 / ipv6 / uri / uuid**: 特殊格式
*   **prefix / suffix / contains**: 前缀/后缀/包含

```protobuf
message CreateUserRequest {
  // 长度限制
  string username = 1 [(validate.rules).string = {min_len: 3, max_len: 32}];
  
  // 正则表达式 (手机号)
  string mobile = 2 [(validate.rules).string = {pattern: "^$|^1[3-9]\\d{9}$"}];
  
  // 邮箱格式
  string email = 3 [(validate.rules).string = {email: true}];
  
  // 必须非空 (对于 string 默认非空即为要求 len > 0，如果确实需要显式非空可配合 min_len)
  string nickname = 4 [(validate.rules).string = {min_len: 1}];
}
```

#### 数值 (Number)
*   **lt / lte / gt / gte**: 小于/小于等于/大于/大于等于
*   **const**: 必须等于某值
*   **in / not_in**: 枚举值范围

```protobuf
message ListUsersRequest {
  // 必须大于 0
  int32 page = 1 [(validate.rules).int32 = {gt: 0}];
  
  // 必须在 1 到 100 之间
  int32 page_size = 2 [(validate.rules).int32 = {gt: 0, lte: 100}];
}
```

#### 集合 (Repeated)
*   **min_items / max_items**: 元素数量限制
*   **unique**: 元素是否唯一
*   **items**: 对集合内元素的约束

```protobuf
message BatchDeleteRequest {
  // 至少包含 1 个元素，最多 100 个，且不能重复
  repeated string ids = 1 [(validate.rules).repeated = {min_items: 1, max_items: 100, unique: true}];
}
```

#### 嵌套消息 (Message)
*   **required**: 且必须存在（用于指针类型或 WKTs）

```protobuf
message UpdateProfileRequest {
    // 嵌套结构体验证
    Profile info = 1 [(validate.rules).message = {required: true}];
}
```

## 2. 验证原理

### 代码生成
`protoc` 编译时会生成对应的 `.pb.go` 文件，其中包含一个 `Validate() error` 方法。这是强类型的 Go 代码验证逻辑。

### 运行时调用
在 HTTP Handler 中，使用 `transport.Context` 的 `ShouldBindAndValidate` 方法时，框架会自动执行验证：

```go
func (h *AuthHandler) Login(c transport.Context) {
    var req v1.LoginRequest
    // 1. 绑定参数 (JSON/Form)
    // 2. 自动调用 req.Validate() (Protoc 生成)
    // 3. (可选) 调用 validator 标签验证
    if err := c.ShouldBindAndValidate(&req); err != nil {
        httputils.WriteResponse(c, errors.ErrBadRequest.WithMessage(err.Error()), nil)
        return
    }
    // ...
}
```

## 3. 混合使用 Go Struct Tag (高级)

虽然推荐在 Proto 中定义所有规则，但在某些特殊场景下（如需要复杂的跨字段验证或特定于 Go 的验证逻辑），你可以结合使用 `go-playground/validator` 的 Tag。

为了在 `protoc` 生成的 Go 结构体中注入 Tag，我们需要使用第三方工具 `protoc-go-inject-tag`。

### 3.1 工具安装

```bash
go install github.com/favadi/protoc-go-inject-tag@latest
```

### 3.2 在 Proto 中声明注入

在 `.proto` 文件的字段上添加注释 `// @gotags: ...` 来声明需要注入的 Tag。

```protobuf
message LoginRequest {
  string username = 1;
  
  // 使用 @gotags 注入 validator 的 required 标签，以及自定义的 json 标签
  // @gotags: json:"password" validate:"required,min=6,max=32"
  string password = 2;
  
  // 仅注入 validator 标签
  // @gotags: validate:"email"
  string email = 3;
}
```

### 3.3 生成代码

只需运行标准的生成命令：

```bash
make gen.proto
```

该命令会自动：
1.  检查并安装 `protoc-go-inject-tag` 工具。
2.  调用 `buf generate` 生成 .pb.go 文件。
3.  自动对 `pkg/api` 下的 .pb.go 文件执行 `protoc-go-inject-tag` 进行注入。

生成的 Go 结构体将会包含你注入的标签：

```go
type LoginRequest struct {
    Username string `protobuf:"..." json:"username,omitempty"`
    
    // 注意：这里被注入了 password 和 validate 标签
    Password string `protobuf:"..." json:"password" validate:"required,min=6,max=32"`
    
    Email    string `protobuf:"..." json:"email,omitempty" validate:"email"`
}
```

### 3.4 运行时验证流程

由于框架统一了验证入口，以下流程是自动支持的：
1.  **优先执行** Proto 定义的 `Validate()` 规则（如果使用了 `validate.rules`）。
2.  **通过后执行** Go Struct Tag 定义的 `validate:"..."` 规则。

两者可以并存，互为补充。例如使用 `validate.rules` 做基础格式校验，使用 `@gotags` 注入复杂的 Go 验证标签（如 `validate:"required_if=..."`）。

## 4. 实战案例：用户模块 (`pkg/api/user-center/v1/user.proto`)

以本项目真实代码为例：

```protobuf
message CreateUserRequest {
  // 要求用户名长度 3-32 位
  string username = 1 [(validate.rules).string = {min_len: 3, max_len: 32}];
  
  // 要求密码长度 8-64 位
  string password = 2 [(validate.rules).string = {min_len: 8, max_len: 64}];
  
  // 必须是合法的邮箱格式
  string email = 3 [(validate.rules).string = {email: true}];
  
  // 手机号正则匹配 (允许为空，或者符合中国大陆手机号格式)
  // ^$ 表示匹配空字符串，| 表示或，后面是手机号正则
  string mobile = 4 [(validate.rules).string = {pattern: "^$|^1[3-9]\\d{9}$"}];
}
```

### 错误处理
验证失败会返回 `400 Bad Request`，错误信息通常包含具体的字段和违反的规则。
例如提交用户名 "ab" (长度2)，会收到错误提示：`invalid field Username: value length must be at least 3 characters`。
