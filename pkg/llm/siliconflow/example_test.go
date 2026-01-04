package siliconflow_test

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/sentinel-x/pkg/llm"
	_ "github.com/kart-io/sentinel-x/pkg/llm/siliconflow"
)

// 演示如何使用基本配置创建 SiliconFlow 供应商并进行对话。
func ExampleNewProvider_basic() {
	// 创建供应商（使用默认配置）
	provider, err := llm.NewProvider("siliconflow", map[string]any{
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
	provider, err := llm.NewProvider("siliconflow", map[string]any{
		"api_key":            "your-api-key-here",
		"chat_model":         "Qwen/Qwen2.5-72B-Instruct", // 使用更大的模型
		"temperature":        0.7,                         // 控制随机性
		"top_p":              0.9,                         // 核采样
		"top_k":              50,                          // Top-K 采样
		"min_p":              0.05,                        // 最小概率阈值（SiliconFlow 特有）
		"max_tokens":         2000,                        // 最大生成 token 数
		"repetition_penalty": 1.1,                         // 重复惩罚
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
	provider, err := llm.NewProvider("siliconflow", map[string]any{
		"api_key":     "your-api-key-here",
		"embed_model": "BAAI/bge-m3", // 使用支持 8192 tokens 的模型
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

// 演示如何使用 EmbedSingle 为单个文本生成向量。
func ExampleProvider_EmbedSingle() {
	// 创建供应商
	provider, err := llm.NewProvider("siliconflow", map[string]any{
		"api_key": "your-api-key-here",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 为单个文本生成向量嵌入
	ctx := context.Background()
	embedding, err := provider.EmbedSingle(ctx, "这是一个测试文本")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("向量维度: %d\n", len(embedding))
	fmt.Printf("前5个元素: %v\n", embedding[:5])
}

// 演示如何使用 Generate 方法进行简单的文本生成。
func ExampleProvider_Generate() {
	// 创建供应商
	provider, err := llm.NewProvider("siliconflow", map[string]any{
		"api_key":     "your-api-key-here",
		"temperature": 0.8, // 更高的随机性以获得更有创意的输出
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

// 演示如何切换到国际区 API 端点。
func ExampleNewProvider_internationalEndpoint() {
	// 使用国际区 API（如果在国外部署）
	provider, err := llm.NewProvider("siliconflow", map[string]any{
		"api_key":  "your-api-key-here",
		"base_url": "https://api.siliconflow.com/v1", // 国际区地址
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
