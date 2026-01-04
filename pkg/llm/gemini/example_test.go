package gemini_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kart-io/sentinel-x/pkg/llm"
	_ "github.com/kart-io/sentinel-x/pkg/llm/gemini"
)

// 演示如何使用基本配置创建 Gemini 供应商。
// Gemini 是 Google 提供的强大多模态 LLM 服务。
func ExampleNewProvider_basic() {
	// 创建供应商（使用默认配置）
	provider, err := llm.NewProvider("gemini", map[string]any{
		"api_key": "YOUR_GEMINI_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Gemini 供应商名称:", provider.Name())
	// Output: Gemini 供应商名称: gemini
}

// 演示如何使用 Chat 方法进行对话。
func ExampleProvider_Chat() {
	if os.Getenv("GEMINI_API_KEY") == "" {
		fmt.Println("跳过示例：需要设置 GEMINI_API_KEY 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("gemini", map[string]any{
		"api_key":    os.Getenv("GEMINI_API_KEY"),
		"chat_model": "gemini-1.5-flash-latest",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 进行多轮对话
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个友好的助手"},
		{Role: llm.RoleUser, Content: "你好，请介绍一下 Gemini"},
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用 Embed 方法生成文本向量嵌入。
func ExampleProvider_Embed() {
	if os.Getenv("GEMINI_API_KEY") == "" {
		fmt.Println("跳过示例：需要设置 GEMINI_API_KEY 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("gemini", map[string]any{
		"api_key":     os.Getenv("GEMINI_API_KEY"),
		"embed_model": "text-embedding-004",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 生成批量文本的向量嵌入
	ctx := context.Background()
	texts := []string{
		"人工智能正在改变世界",
		"Gemini 是 Google 的多模态 LLM",
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

// 演示如何使用 Generate 方法进行简单的文本生成。
func ExampleProvider_Generate() {
	if os.Getenv("GEMINI_API_KEY") == "" {
		fmt.Println("跳过示例：需要设置 GEMINI_API_KEY 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("gemini", map[string]any{
		"api_key":    os.Getenv("GEMINI_API_KEY"),
		"chat_model": "gemini-1.5-flash-latest",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用 system prompt 和 user prompt 生成文本
	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"写一首关于 AI 的短诗",
		"你是一位专业的诗人",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("生成的诗:", response)
}
