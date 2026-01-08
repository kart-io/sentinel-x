## 项目上下文摘要（Token 统计功能）
生成时间：2026-01-08

### 1. 相似实现分析

#### 实现1: pkg/llm/openai/provider.go:296-312 (chatResponse 结构)
- **模式**：OpenAI API 响应结构定义
- **关键发现**：已包含 Usage 字段，包含 PromptTokens、CompletionTokens、TotalTokens
- **可复用**：chatResponse.Usage 结构体定义
- **需注意**：OpenAI 兼容 API（SiliconFlow、DeepSeek）都返回相同格式的 usage 字段

#### 实现2: pkg/llm/gemini/provider.go:211-227 (chatResponse 结构)
- **模式**：Gemini API 响应结构定义
- **关键发现**：使用 UsageMetadata 字段，字段名略有不同（PromptTokenCount vs PromptTokens）
- **可复用**：需要适配不同 provider 的字段命名差异
- **需注意**：不同 LLM provider 的 token 统计字段命名不一致

#### 实现3: internal/rag/biz/service.go:118-122 (当前 TODO 注释)
- **模式**：Service 层调用 Generator，记录 metrics
- **关键发现**：已有 metrics.RecordLLMCall 接口，接受 promptTokens 和 completionTokens 参数
- **可复用**：metrics 记录接口已就绪，只需传递实际 token 数据
- **需注意**：当前硬编码为 0，需要从 Generator 传递真实数据

#### 实现4: internal/rag/metrics/metrics.go:153-168 (RecordLLMCall 方法)
- **模式**：业务指标收集
- **关键发现**：已实现 token 统计逻辑，使用 Counter 累加
- **可复用**：完整的 metrics 收集框架
- **需注意**：只统计成功调用的 token（err == nil 时才记录）

### 2. 项目约定

#### 命名约定
- 包名使用小写，如 `llm`, `biz`, `metrics`
- 结构体使用大驼峰，如 `TokenUsage`, `GenerateResponse`
- 字段使用大驼峰（导出）或小驼峰（私有）
- 接口方法使用大驼峰，如 `Generate`, `Chat`

#### 文件组织
- `pkg/llm/`: 通用 LLM 抽象层
- `pkg/llm/{provider}/`: 具体 provider 实现
- `internal/rag/biz/`: RAG 业务逻辑
- `internal/rag/metrics/`: RAG 业务指标

#### 代码风格
- 所有注释使用简体中文
- 错误返回使用 `fmt.Errorf` 包装
- 结构体字段使用 JSON tag 标注
- 导入顺序：标准库 -> 第三方库 -> 项目内部库

### 3. 可复用组件清单

#### pkg/llm/provider.go
- `ChatProvider` 接口：需要扩展以返回 token 信息
- `Message` 结构体：对话消息定义
- `Role` 类型：消息角色常量

#### internal/rag/metrics/metrics.go
- `RAGMetrics.RecordLLMCall(duration, promptTokens, completionTokens, err)`: 已就绪的 metrics 记录接口
- `llmTokensPrompt` 和 `llmTokensCompletion`: Counter 指标

#### pkg/llm/openai/provider.go
- `chatResponse.Usage` 结构体：OpenAI 格式的 token 统计
- 字段命名：`prompt_tokens`, `completion_tokens`, `total_tokens`

### 4. 测试策略

#### 测试框架
- Go 标准 testing 包
- 表驱动测试（table-driven tests）

#### 测试模式
- **单元测试**：测试新增的 TokenUsage 和 GenerateResponse 结构
- **Mock 测试**：使用 mockProvider 测试接口变更
- **集成测试**：端到端测试 RAG 服务的 token 统计

#### 参考文件
- `pkg/llm/provider_test.go`: mockProvider 实现模式
- `pkg/llm/openai/provider_test.go`: 真实 provider 测试示例

#### 覆盖要求
- 核心接口变更必须有单元测试
- 至少一个 provider 实现完整的 token 统计测试
- RAG service 层的集成测试验证 metrics 正确记录

### 5. 依赖和集成点

#### 外部依赖
- 无新增外部依赖，使用现有的 Go 标准库

#### 内部依赖
- `pkg/llm` → `internal/rag/biz` → `internal/rag/metrics`
- 数据流向：LLM API Response → Generator → RAGService → Metrics

#### 集成方式
1. **接口扩展**：修改 `ChatProvider.Generate` 方法签名
2. **向后兼容**：保持现有方法，添加新的返回结构
3. **渐进式迁移**：先实现 OpenAI provider，再扩展到其他 provider

#### 配置来源
- 无需配置变更，使用现有配置

### 6. 技术选型理由

#### 为什么使用新的 GenerateResponse 结构？
- **理由1**：Go 不支持多返回值的接口演化，必须通过结构体包装
- **理由2**：结构体可扩展，未来可添加其他元数据（如 finish_reason）
- **理由3**：符合 Go 社区最佳实践，避免破坏性变更

#### 为什么定义独立的 TokenUsage 结构？
- **理由1**：不同 provider 的 token 字段命名不同，需要统一抽象
- **理由2**：可复用于未来的 Chat 方法扩展
- **理由3**：清晰的领域模型，便于理解和维护

#### 优势
- 最小化接口变更影响
- 向后兼容性好
- 易于测试和验证
- 符合项目既有架构风格

#### 劣势和风险
- **风险1**：需要修改所有 provider 实现（6 个 provider）
- **风险2**：Gemini 的字段名不同，需要适配转换
- **风险3**：某些 provider（如 Ollama）可能不返回 token 统计，需要估算或返回 nil

### 7. 关键风险点

#### 并发问题
- **问题**：metrics Counter 的并发安全性
- **解决**：已由 `pkg/observability/metrics` 保证，无需额外处理

#### 边界条件
- **问题1**：provider API 不返回 token 信息时如何处理？
- **解决1**：TokenUsage 设为 nil，metrics 层已处理（promptTokens > 0 才记录）

- **问题2**：API 返回错误时是否记录 token？
- **解决2**：遵循现有逻辑，错误时不记录 token（只记录 error）

#### 性能瓶颈
- **问题**：新增结构体是否影响性能？
- **评估**：影响极小，仅增加一个指针字段和 3 个 int 字段
- **优化**：TokenUsage 使用指针，未使用时为 nil，零成本

#### Provider 兼容性
- **OpenAI**：完全支持，直接映射 Usage 字段 ✅
- **SiliconFlow**：兼容 OpenAI 格式，完全支持 ✅
- **DeepSeek**：兼容 OpenAI 格式，完全支持 ✅
- **Gemini**：部分支持，需要字段名转换（PromptTokenCount → PromptTokens）⚠️
- **Ollama**：可能不支持，需要估算或返回 nil ⚠️
- **HuggingFace**：取决于具体 API，需要验证 ❓

### 8. 实施计划

#### 阶段1：核心接口定义（pkg/llm/provider.go）
- 定义 TokenUsage 结构体
- 定义 GenerateResponse 结构体
- 修改 ChatProvider 接口的 Generate 方法签名

#### 阶段2：Generator 层适配（internal/rag/biz/generator.go）
- 修改 GenerateAnswer 方法返回值
- 接收并传递 token 信息

#### 阶段3：Service 层集成（internal/rag/biz/service.go）
- 从 Generator 获取 token 信息
- 传递给 metrics.RecordLLMCall

#### 阶段4：Provider 实现更新
- **优先级1**：OpenAI（最常用，完全支持）
- **优先级2**：SiliconFlow、DeepSeek（兼容 OpenAI）
- **优先级3**：Gemini（需要字段转换）
- **优先级4**：Ollama、HuggingFace（可能需要估算）

#### 阶段5：测试和验证
- 单元测试：接口和结构体
- 集成测试：端到端 token 统计
- 回归测试：确保现有功能不受影响

### 9. 待确认问题

#### Q1：是否需要同时修改 Chat 方法？
- **分析**：Chat 方法也调用 LLM API，理论上也应返回 token
- **建议**：先实现 Generate，Chat 方法作为后续增强

#### Q2：是否需要估算不支持 token 统计的 provider？
- **分析**：Ollama 等本地模型可能不返回 token
- **建议**：返回 nil，由 metrics 层判断，不强制估算

#### Q3：是否需要向后兼容旧的 Generate 方法签名？
- **分析**：Go 接口变更是破坏性的，所有实现都必须更新
- **建议**：一次性更新所有 provider，不做向后兼容

### 10. 成功标准

- ✅ ChatProvider 接口已扩展，支持 token 统计
- ✅ Generator 和 Service 层成功传递 token 信息
- ✅ 至少 OpenAI provider 实现完整的 token 统计
- ✅ Metrics 正确记录 prompt 和 completion tokens
- ✅ 所有现有测试通过
- ✅ 新增测试覆盖核心功能
- ✅ 代码编译无错误
