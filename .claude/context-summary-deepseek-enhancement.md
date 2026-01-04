## 项目上下文摘要（DeepSeek 供应商改进）

生成时间：2026-01-04 15:30:00

### 1. 相似实现分析

**实现1：SiliconFlow 供应商（pkg/llm/siliconflow/provider.go）**
- 模式：完整的高级参数支持
- 可复用：
  - Config 结构体的参数定义模式（Temperature, TopP, MaxTokens 等）
  - NewProvider 函数的 configMap 解析逻辑
  - chatRequest 结构体的可选字段设计（omitempty）
  - Chat 方法中的条件应用参数逻辑（仅在非零值时设置）
- 需注意：
  - SiliconFlow 特有参数（TopK, MinP, RepetitionPenalty）DeepSeek 可能不支持
  - DeepSeek 支持的参数需要参考官方文档

**实现2：SiliconFlow 示例文件（pkg/llm/siliconflow/example_test.go）**
- 模式：Example 测试风格
- 可复用：
  - ExampleNewProvider_basic - 基本用法示例
  - ExampleNewProvider_advanced - 高级配置示例
  - ExampleProvider_Generate - 文本生成示例
  - 示例命名约定：`Example[类型]_[场景]`
- 需注意：
  - DeepSeek 不支持 Embedding，所以不需要 Embed 相关示例
  - 示例中的 API key 使用 "your-api-key-here" 占位符

**实现3：OpenAI 供应商（pkg/llm/openai/provider.go）**
- 模式：简化版实现（仅基础参数）
- 可复用：基础结构设计
- 需注意：OpenAI 供应商没有高级参数，SiliconFlow 是更好的参考

### 2. 项目约定

- **命名约定**：
  - 结构体字段使用驼峰命名
  - JSON tag 使用 snake_case（如 `api_key`, `max_tokens`）
  - mapstructure tag 与 JSON tag 一致
  - 常量使用 PascalCase（如 ProviderName）

- **文件组织**：
  - 每个供应商一个独立包（pkg/llm/[provider]/）
  - provider.go 包含核心实现
  - example_test.go 包含示例代码

- **注释规范**：
  - 包级 godoc 使用多行注释
  - 包含基本用法和高级配置两个示例
  - 所有注释使用简体中文
  - 参数字段必须注释说明用途、范围和默认值

- **代码风格**：
  - 使用 gofmt 格式化
  - 导入顺序：标准库 -> 第三方库 -> 项目库
  - 错误处理使用 fmt.Errorf 包装

### 3. 可复用组件清单

- `pkg/llm.Provider` - 供应商接口
- `pkg/llm.Message` - 消息结构体
- `pkg/llm.Role*` - 角色常量（RoleSystem, RoleUser, RoleAssistant）
- `pkg/utils/httpclient.Client` - HTTP 客户端（支持重试）
- `pkg/utils/json.Marshal` - JSON 序列化工具

### 4. 测试策略

- **测试框架**：Go testing（标准库）
- **测试模式**：Example 测试（可运行的示例代码）
- **示例类型**：
  - ExampleNewProvider_basic - 基本配置和使用
  - ExampleNewProvider_advanced - 高级参数配置
  - ExampleProvider_Chat - 多轮对话
  - ExampleProvider_Generate - 单轮文本生成
- **覆盖要求**：每个公开 API 至少一个示例

### 5. 依赖和集成点

- **外部依赖**：无（使用标准库和项目内部工具）
- **内部依赖**：
  - pkg/llm - 接口定义
  - pkg/utils/httpclient - HTTP 客户端
  - pkg/utils/json - JSON 工具
- **集成方式**：
  - init() 函数注册供应商：`llm.RegisterProvider(ProviderName, NewProvider)`
  - 通过工厂函数创建：`llm.NewProvider("deepseek", configMap)`
- **配置来源**：通过 configMap 传入（map[string]any）

### 6. 技术选型理由

- **为什么支持高级参数**：
  - 提供更精细的生成控制
  - 满足不同场景的需求（创意写作 vs 技术问答）
  - 与 DeepSeek API 官方支持保持一致

- **优势**：
  - 兼容 OpenAI API 格式，易于集成
  - 参数可选，向后兼容现有代码
  - 类型安全的配置解析

- **劣势和风险**：
  - 需要验证 DeepSeek 支持哪些参数（避免发送不支持的参数）
  - 参数验证逻辑可能需要后续完善

### 7. 关键风险点

- **API 兼容性**：
  - DeepSeek API 虽然兼容 OpenAI 格式，但支持的参数可能有差异
  - 需要确认 DeepSeek 支持的具体参数列表

- **参数范围验证**：
  - Temperature、TopP 等参数有合法范围
  - 当前实现仅检查非零值，不验证范围

- **向后兼容**：
  - 新增参数必须是可选的（使用 omitempty）
  - 默认值为 0 时不影响现有行为

- **测试覆盖**：
  - Example 测试不会真正执行（除非提供真实 API key）
  - 需要确保示例代码的正确性
