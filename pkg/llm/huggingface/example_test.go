package huggingface_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kart-io/sentinel-x/pkg/llm"
	_ "github.com/kart-io/sentinel-x/pkg/llm/huggingface"
)

// 演示如何使用基本配置创建 HuggingFace 供应商。
// HuggingFace 提供免费的 Inference API,可使用 Hub 上的开源模型。
func ExampleNewProvider_basic() {
	// 创建供应商（使用默认模型）
	provider, err := llm.NewProvider("huggingface", map[string]any{
		"api_key": "YOUR_HUGGINGFACE_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("HuggingFace 供应商名称:", provider.Name())
	// Output: HuggingFace 供应商名称: huggingface
}

// 演示如何使用自定义模型配置创建 HuggingFace 供应商。
func ExampleNewProvider_customModels() {
	// 创建供应商（指定自定义模型）
	provider, err := llm.NewProvider("huggingface", map[string]any{
		"api_key":        "YOUR_HUGGINGFACE_API_KEY",
		"embed_model":    "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2",
		"chat_model":     "mistralai/Mistral-7B-Instruct-v0.2",
		"wait_for_model": true,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("HuggingFace 供应商名称:", provider.Name())
	// Output: HuggingFace 供应商名称: huggingface
}

// 演示如何使用 Embed 方法生成文本向量嵌入。
func ExampleProvider_Embed() {
	if os.Getenv("HUGGINGFACE_API_KEY") == "" {
		fmt.Println("跳过示例：需要设置 HUGGINGFACE_API_KEY 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("huggingface", map[string]any{
		"api_key":     os.Getenv("HUGGINGFACE_API_KEY"),
		"embed_model": "sentence-transformers/all-MiniLM-L6-v2",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 生成批量文本的向量嵌入
	ctx := context.Background()
	texts := []string{
		"人工智能正在改变世界",
		"HuggingFace 提供强大的模型库",
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

// 演示如何使用 Chat 方法进行对话。
func ExampleProvider_Chat() {
	if os.Getenv("HUGGINGFACE_API_KEY") == "" {
		fmt.Println("跳过示例：需要设置 HUGGINGFACE_API_KEY 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("huggingface", map[string]any{
		"api_key":    os.Getenv("HUGGINGFACE_API_KEY"),
		"chat_model": "mistralai/Mistral-7B-Instruct-v0.2",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 进行多轮对话
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个友好的助手"},
		{Role: llm.RoleUser, Content: "你好，请介绍一下 HuggingFace"},
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用 Generate 方法进行简单的文本生成。
func ExampleProvider_Generate() {
	if os.Getenv("HUGGINGFACE_API_KEY") == "" {
		fmt.Println("跳过示例：需要设置 HUGGINGFACE_API_KEY 环境变量")
		return
	}

	// 创建供应商
	provider, err := llm.NewProvider("huggingface", map[string]any{
		"api_key":    os.Getenv("HUGGINGFACE_API_KEY"),
		"chat_model": "mistralai/Mistral-7B-Instruct-v0.2",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用 system prompt 和 user prompt 生成文本
	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"写一首关于开源的短诗",
		"你是一位专业的诗人",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("生成的诗:", response)
}
