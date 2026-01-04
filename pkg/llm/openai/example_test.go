package openai_test

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/sentinel-x/pkg/llm"
	_ "github.com/kart-io/sentinel-x/pkg/llm/openai"
)

// 演示如何使用基本配置创建 OpenAI 供应商并进行对话。
func ExampleNewProvider_basic() {
	// 创建供应商（使用默认配置）
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key": "your-api-key-here",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 进行对话
	ctx := context.Background()
	response, err := provider.Chat(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "你好，请介绍一下自己"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用高级配置来精细控制生成参数。
func ExampleNewProvider_advanced() {
	// 创建供应商（使用高级配置）
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key":           "your-api-key-here",
		"chat_model":        "gpt-4o",       // 使用 GPT-4o 模型
		"temperature":       0.7,            // 控制随机性（0.0-2.0）
		"top_p":             0.9,            // 核采样参数
		"max_tokens":        2000,           // 最大生成 token 数
		"frequency_penalty": 0.5,            // 频率惩罚，减少重复
		"presence_penalty":  0.5,            // 存在惩罚，增加话题多样性
		"stop":              []string{"\n"}, // 遇到换行符停止
	})
	if err != nil {
		log.Fatal(err)
	}

	// 进行多轮对话
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个专业的技术顾问"},
		{Role: llm.RoleUser, Content: "什么是微服务架构？"},
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用 Embedding API 生成文本向量。
func ExampleProvider_Embed() {
	// 创建供应商
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key":     "your-api-key-here",
		"embed_model": "text-embedding-3-large", // 使用更大的 Embedding 模型
	})
	if err != nil {
		log.Fatal(err)
	}

	// 生成批量文本的向量嵌入
	ctx := context.Background()
	texts := []string{
		"人工智能是未来的发展方向",
		"机器学习是人工智能的一个分支",
		"深度学习使用神经网络进行训练",
	}

	embeddings, err := provider.Embed(ctx, texts)
	if err != nil {
		log.Fatal(err)
	}

	// 打印向量维度
	for i, emb := range embeddings {
		fmt.Printf("文本 %d 的向量维度: %d\n", i+1, len(emb))
	}
}

// 演示如何使用 Chat 方法进行多轮对话。
func ExampleProvider_Chat() {
	// 创建供应商
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key":     "your-api-key-here",
		"temperature": 0.8, // 更高的随机性以获得更有创意的输出
		"max_tokens":  500,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 多轮对话
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个友好的助手"},
		{Role: llm.RoleUser, Content: "你好"},
		{Role: llm.RoleAssistant, Content: "你好！有什么可以帮助你的吗？"},
		{Role: llm.RoleUser, Content: "请告诉我今天的天气"},
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何配置 Azure OpenAI。
func ExampleNewProvider_azureOpenAI() {
	// 配置 Azure OpenAI
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key":  "your-azure-api-key",
		"base_url": "https://your-resource.openai.azure.com/openai/deployments/your-deployment", // Azure OpenAI 端点
		// Azure OpenAI 使用部署名称而不是模型名称
		"chat_model":  "gpt-4o", // 部署名称
		"embed_model": "text-embedding-3-small",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	response, err := provider.Chat(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用 Generate 方法进行简单的文本生成。
func ExampleProvider_Generate() {
	// 创建供应商
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key":     "your-api-key-here",
		"temperature": 0.9, // 高随机性以获得更有创意的输出
		"max_tokens":  500,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用 system prompt 和 user prompt 生成文本
	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"写一首关于春天的诗",
		"你是一位专业的诗人",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("生成的诗:", response)
}

// 演示如何使用停止序列来控制生成的文本。
func ExampleNewProvider_stopSequences() {
	// 创建供应商，配置停止序列
	provider, err := llm.NewProvider("openai", map[string]any{
		"api_key": "your-api-key-here",
		"stop":    []string{"\n\n", "END", "。"}, // 遇到这些字符串时停止生成
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"请用一句话介绍人工智能",
		"",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}
