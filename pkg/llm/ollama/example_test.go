package ollama_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kart-io/sentinel-x/pkg/llm"
	_ "github.com/kart-io/sentinel-x/pkg/llm/ollama"
)

// 演示如何使用基本配置创建 Ollama 供应商。
// Ollama 是本地部署的 LLM 服务，需要先启动 Ollama 服务。
func ExampleNewProvider_basic() {
	// 创建供应商（连接本地 Ollama 服务）
	provider, err := llm.NewProvider("ollama", map[string]any{
		"base_url":    "http://localhost:11434",
		"chat_model":  "llama3",
		"embed_model": "nomic-embed-text",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Ollama 供应商名称:", provider.Name())
	// Output: Ollama 供应商名称: ollama
}

// 演示如何使用 Chat 方法进行对话。
func ExampleProvider_Chat() {
	if os.Getenv("OLLAMA_BASE_URL") == "" {
		fmt.Println("跳过示例：需要设置 OLLAMA_BASE_URL 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("ollama", map[string]any{
		"base_url":   os.Getenv("OLLAMA_BASE_URL"),
		"chat_model": "llama3",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 进行多轮对话
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个友好的助手"},
		{Role: llm.RoleUser, Content: "你好，请介绍一下 Ollama"},
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用 Embed 方法生成文本向量嵌入。
func ExampleProvider_Embed() {
	if os.Getenv("OLLAMA_BASE_URL") == "" {
		fmt.Println("跳过示例：需要设置 OLLAMA_BASE_URL 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("ollama", map[string]any{
		"base_url":    os.Getenv("OLLAMA_BASE_URL"),
		"embed_model": "nomic-embed-text",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 生成批量文本的向量嵌入
	ctx := context.Background()
	texts := []string{
		"人工智能是未来的发展方向",
		"Ollama 是本地部署的 LLM 服务",
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
	if os.Getenv("OLLAMA_BASE_URL") == "" {
		fmt.Println("跳过示例：需要设置 OLLAMA_BASE_URL 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("ollama", map[string]any{
		"base_url":   os.Getenv("OLLAMA_BASE_URL"),
		"chat_model": "llama3",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用 system prompt 和 user prompt 生成文本
	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"写一首关于 Go 语言的短诗",
		"你是一位专业的诗人",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("生成的诗:", response)
}
