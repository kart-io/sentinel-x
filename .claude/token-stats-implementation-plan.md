# Token 统计功能实施计划

生成时间: 2026-01-08 16:53

## 📋 计划概述

**目标**: 实现 LLM 调用的 Token 使用统计功能，准确记录每次调用的 token 消耗。

**优先级**: 🔥 高（P0 - 本周必须完成）

**预估工作量**: 6 小时

**破坏性变更**: 是（需要修改接口签名）

---

## 🏗️ 架构设计

### 1. 数据结构设计

```go
// pkg/llm/provider.go

// TokenUsage Token 使用统计
type TokenUsage struct {
    PromptTokens     int // 提示词 token 数量
    CompletionTokens int // 生成内容 token 数量
    TotalTokens      int // 总 token 数量
}

// GenerateResponse LLM 生成响应
type GenerateResponse struct {
    Content    string      // 生成的文本内容
    TokenUsage *TokenUsage // Token 使用统计（可能为 nil）
}
```

### 2. 接口变更策略

**策略**: 破坏性变更（直接修改接口）

**理由**:
- 项目处于早期阶段，向后兼容性不是首要考虑
- 清晰的接口比兼容性包袱更重要
- 所有调用方都在项目内部，可控

**变更范围**:
```go
// 修改前
type ChatProvider interface {
    Chat(ctx context.Context, messages []Message) (string, error)
    Generate(ctx context.Context, prompt string, systemPrompt string) (string, error)
    Name() string
}

// 修改后
type ChatProvider interface {
    Chat(ctx context.Context, messages []Message) (*GenerateResponse, error)
    Generate(ctx context.Context, prompt string, systemPrompt string) (*GenerateResponse, error)
    Name() string
}
```

### 3. 错误处理策略

- TokenUsage 字段可以为 `nil`（兼容不支持 token 统计的提供商）
- 上层代码必须检查 `nil` 避免 panic
- 如果提供商不返回 token 信息，使用 0 作为默认值

---

## 📝 实施步骤

### 步骤 1: 定义数据结构 (15 分钟)

**文件**: `pkg/llm/provider.go`

**修改内容**:
```go
// 在 Message 结构体后添加

// TokenUsage Token 使用统计
type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// GenerateResponse LLM 生成响应
type GenerateResponse struct {
    Content    string
    TokenUsage *TokenUsage
}
```

**依赖**: 无

**验证**: 代码编译通过

---

### 步骤 2: 更新 ChatProvider 接口 (15 分钟)

**文件**: `pkg/llm/provider.go`

**修改内容**:
```go
// ChatProvider 定义 Chat 供应商接口。
type ChatProvider interface {
    // Chat 进行多轮对话。
    Chat(ctx context.Context, messages []Message) (*GenerateResponse, error)

    // Generate 根据提示生成文本（单轮）。
    Generate(ctx context.Context, prompt string, systemPrompt string) (*GenerateResponse, error)

    // Name 返回供应商名称。
    Name() string
}
```

**依赖**: 步骤 1

**影响**: 所有实现 ChatProvider 的代码将编译失败（预期行为）

**验证**: 运行 `go build ./...` 查看编译错误

---

### 步骤 3: 更新 Generator (30 分钟)

**文件**: `internal/rag/biz/generator.go`

**修改内容**:
```go
// GenerateAnswer 根据检索结果生成答案。
func (g *Generator) GenerateAnswer(ctx context.Context, question string, results []*store.SearchResult) (*GenerateResponse, error) {
    if len(results) == 0 {
        return &GenerateResponse{
            Content:    "I couldn't find any relevant information in the knowledge base.",
            TokenUsage: nil,
        }, nil
    }

    // 检查 context 是否已取消
    if ctx.Err() != nil {
        return nil, fmt.Errorf("context cancelled before generation: %w", ctx.Err())
    }

    // 构建上下文
    var contextBuilder strings.Builder
    for i, result := range results {
        contextBuilder.WriteString(fmt.Sprintf("[%d] From %s - %s:\n%s\n\n",
            i+1, result.DocumentName, result.Section, result.Content))
    }

    // 生成提示词
    prompt := strings.ReplaceAll(g.config.SystemPrompt, "{{context}}", contextBuilder.String())
    prompt = strings.ReplaceAll(prompt, "{{question}}", question)

    // 调用 LLM 生成答案
    logger.Info("Calling LLM to generate answer...")
    resp, err := g.chatProvider.Generate(ctx, prompt, "")
    if err != nil {
        logger.Errorf("LLM generation failed: %v", err)
        return nil, fmt.Errorf("failed to generate answer: %w", err)
    }

    if resp.TokenUsage != nil {
        logger.Infof("LLM answer generated (length: %d, tokens: %d)",
            len(resp.Content), resp.TokenUsage.TotalTokens)
    } else {
        logger.Infof("LLM answer generated (length: %d)", len(resp.Content))
    }

    return resp, nil
}
```

**依赖**: 步骤 2

**验证**: 编译通过，逻辑正确

---

### 步骤 4: 更新 RAGService (30 分钟)

**文件**: `internal/rag/biz/service.go`

**修改内容**:
```go
// 3. 生成答案
llmStart := time.Now()
resp, err := s.generator.GenerateAnswer(ctx, question, retrievalResult.Results)
llmDuration := time.Since(llmStart)

// 从响应中获取 token 使用信息
promptTokens := 0
completionTokens := 0
if resp != nil && resp.TokenUsage != nil {
    promptTokens = resp.TokenUsage.PromptTokens
    completionTokens = resp.TokenUsage.CompletionTokens
}
s.metrics.RecordLLMCall(llmDuration, promptTokens, completionTokens, err)

if err != nil {
    queryErr = err
    return nil, err
}

// 4. 构建响应
sources := make([]model.ChunkSource, len(retrievalResult.Results))
for i, result := range retrievalResult.Results {
    sources[i] = model.ChunkSource{
        DocumentID:   result.DocumentID,
        DocumentName: result.DocumentName,
        Section:      result.Section,
        Content:      result.Content,
        Score:        result.Score,
    }
}

queryResult := &model.QueryResult{
    Answer:  resp.Content,  // 使用 resp.Content 而不是 answer
    Sources: sources,
}
```

**依赖**: 步骤 3

**验证**: 编译通过，逻辑正确

---

### 步骤 5: 更新 LLM 提供商实现 (3 小时)

#### 5.1 OpenAI Provider (45 分钟)

**文件**: `pkg/llm/openai/provider.go`

**修改内容**:
- 更新 `Generate` 方法返回 `*llm.GenerateResponse`
- 从 OpenAI API 响应中提取 token 信息
- 填充 TokenUsage 结构

**代码示例**:
```go
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string, systemPrompt string) (*llm.GenerateResponse, error) {
    // ... 构建请求 ...

    // 调用 OpenAI API
    resp, err := p.client.CreateChatCompletion(ctx, req)
    if err != nil {
        return nil, err
    }

    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no response from OpenAI")
    }

    return &llm.GenerateResponse{
        Content: resp.Choices[0].Message.Content,
        TokenUsage: &llm.TokenUsage{
            PromptTokens:     resp.Usage.PromptTokens,
            CompletionTokens: resp.Usage.CompletionTokens,
            TotalTokens:      resp.Usage.TotalTokens,
        },
    }, nil
}
```

#### 5.2 DeepSeek Provider (45 分钟)

**文件**: `pkg/llm/deepseek/provider.go`

**修改内容**: 类似 OpenAI，从 API 响应中提取 token 信息

#### 5.3 SiliconFlow Provider (45 分钟)

**文件**: `pkg/llm/siliconflow/provider.go`

**修改内容**: 类似 OpenAI，从 API 响应中提取 token 信息

#### 5.4 Ollama Provider (30 分钟)

**文件**: `pkg/llm/ollama/provider.go`

**修改内容**:
- Ollama 可能不返回 token 信息
- 使用简单估算或返回 `nil`

**代码示例**:
```go
return &llm.GenerateResponse{
    Content:    answer,
    TokenUsage: nil, // Ollama 不提供 token 统计
}, nil
```

#### 5.5 Gemini Provider (30 分钟)

**文件**: `pkg/llm/gemini/provider.go`

**修改内容**: 从 Gemini API 响应中提取 token 信息

---

### 步骤 6: 更新测试 (1 小时)

#### 6.1 更新 Provider 测试

**文件**: `pkg/llm/provider_test.go`

**修改内容**:
- 更新 mock 返回值为 `*GenerateResponse`
- 验证 TokenUsage 字段

#### 6.2 更新 Generator 测试

**文件**: `internal/rag/biz/generator_test.go` (如果存在)

**修改内容**:
- 更新断言检查 `*GenerateResponse`
- 测试 TokenUsage 为 nil 的情况

#### 6.3 更新 RAGService 测试

**文件**: `internal/rag/biz/service_test.go` (如果存在)

**修改内容**:
- Mock Generator 返回带 TokenUsage 的响应
- 验证 metrics 记录正确

---

## ✅ 验收标准

### 功能验收
- [ ] 所有 LLM 提供商实现已更新
- [ ] OpenAI、DeepSeek、SiliconFlow 返回真实 token 统计
- [ ] Ollama、Gemini 至少返回有效的响应（TokenUsage 可为 nil）
- [ ] RAG Service 正确记录 token 使用到 metrics
- [ ] 缓存的响应也包含 token 信息（如果原始响应有）

### 技术验收
- [ ] 所有代码编译通过 (`go build ./...`)
- [ ] 所有测试通过 (`go test ./...`)
- [ ] 没有引入数据竞争 (`go test -race ./...`)
- [ ] 代码格式化正确 (`go fmt ./...`)

### 质量验收
- [ ] 所有注释使用简体中文
- [ ] 错误处理完善（检查 nil）
- [ ] 日志记录清晰
- [ ] 无明显性能退化

---

## ⚠️ 风险评估

### 高风险
1. **破坏性变更**: 修改接口签名会导致所有实现和调用方编译失败
   - **缓解**: 一次性完成所有修改，确保编译通过后再提交

2. **API 差异**: 不同 LLM 提供商返回的 token 信息格式不同
   - **缓解**: 统一映射到 TokenUsage 结构，提供清晰的文档

### 中风险
3. **测试覆盖不足**: 可能遗漏某些边界情况
   - **缓解**: 编写全面的单元测试和集成测试

4. **性能影响**: 增加结构体字段可能影响性能
   - **缓解**: 使用指针避免不必要的拷贝，进行性能测试

### 低风险
5. **向后兼容性**: 旧代码无法使用
   - **缓解**: 项目内部控制，可接受

---

## 🔄 回滚策略

如果实施过程中遇到重大问题：

1. **快速回滚**: 使用 `git revert` 回滚提交
2. **数据保护**: token 统计是新增功能，不影响现有数据
3. **服务可用性**: 不影响服务正常运行，只是指标不准确

---

## 📊 工作量分配

| 步骤 | 工作量 | 关键路径 |
|------|--------|----------|
| 1. 定义数据结构 | 15 分钟 | ✅ |
| 2. 更新接口 | 15 分钟 | ✅ |
| 3. 更新 Generator | 30 分钟 | ✅ |
| 4. 更新 RAGService | 30 分钟 | ✅ |
| 5. 更新提供商 | 3 小时 | ✅ |
| 6. 更新测试 | 1 小时 | ✅ |
| **总计** | **6 小时** | |

---

## 📋 执行检查清单

### 准备阶段
- [x] 制定实施计划
- [ ] 审查计划可行性
- [ ] 确认工作环境就绪

### 实施阶段
- [ ] 步骤 1: 定义数据结构
- [ ] 步骤 2: 更新接口
- [ ] 步骤 3: 更新 Generator
- [ ] 步骤 4: 更新 RAGService
- [ ] 步骤 5.1: 更新 OpenAI Provider
- [ ] 步骤 5.2: 更新 DeepSeek Provider
- [ ] 步骤 5.3: 更新 SiliconFlow Provider
- [ ] 步骤 5.4: 更新 Ollama Provider
- [ ] 步骤 5.5: 更新 Gemini Provider
- [ ] 步骤 6: 更新测试

### 验证阶段
- [ ] 编译验证
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能测试
- [ ] 代码审查

### 完成阶段
- [ ] 提交代码
- [ ] 更新文档
- [ ] 通知相关方

---

**计划制定时间**: 2026-01-08 16:53
**计划审查人**: Claude Code
**预计开始时间**: 2026-01-08 17:00
**预计完成时间**: 2026-01-08 23:00

---

*本计划将指导 Token 统计功能的完整实施过程*
