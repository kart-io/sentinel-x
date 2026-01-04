package deepseek_test

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/sentinel-x/pkg/llm"
	_ "github.com/kart-io/sentinel-x/pkg/llm/deepseek"
)

// 演示如何使用基本配置创建 DeepSeek 供应商并进行对话。
func ExampleNewProvider_basic() {
	// 创建供应商（使用默认配置）
	provider, err := llm.NewProvider("deepseek", map[string]any{
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
	provider, err := llm.NewProvider("deepseek", map[string]any{
		"api_key":           "your-api-key-here",
		"chat_model":        "deepseek-chat",       // 使用 deepseek-chat 模型
		"temperature":       0.7,                   // 控制随机性
		"top_p":             0.9,                   // 核采样
		"max_tokens":        2000,                  // 最大生成 token 数
		"frequency_penalty": 0.5,                   // 频率惩罚，减少重复
		"presence_penalty":  0.5,                   // 存在惩罚，鼓励新话题
		"stop":              []string{"结束", "END"}, // 停止序列
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

// 演示如何进行多轮对话。
func ExampleProvider_Chat() {
	// 创建供应商
	provider, err := llm.NewProvider("deepseek", map[string]any{
		"api_key":     "your-api-key-here",
		"temperature": 0.7, // 控制输出的随机性
	})
	if err != nil {
		log.Fatal(err)
	}

	// 构建多轮对话
	ctx := context.Background()
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个有帮助的助手"},
		{Role: llm.RoleUser, Content: "请解释什么是 Go 语言的 goroutine"},
		{Role: llm.RoleAssistant, Content: "Goroutine 是 Go 语言的轻量级线程..."},
		{Role: llm.RoleUser, Content: "如何创建一个 goroutine？"},
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用 Generate 方法进行简单的文本生成。
func ExampleProvider_Generate() {
	// 创建供应商
	provider, err := llm.NewProvider("deepseek", map[string]any{
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

// 演示如何使用停止序列控制生成。
func ExampleNewProvider_withStopSequences() {
	// 创建供应商，配置停止序列
	provider, err := llm.NewProvider("deepseek", map[string]any{
		"api_key": "your-api-key-here",
		"stop":    []string{"结束", "END", "\n\n"}, // 遇到这些序列时停止生成
	})
	if err != nil {
		log.Fatal(err)
	}

	// 生成文本
	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"请列出 Go 语言的优点",
		"你是一个技术专家，回答要简洁",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}

// 演示如何使用频率和存在惩罚来控制重复。
func ExampleNewProvider_withPenalties() {
	// 创建供应商，配置惩罚参数
	provider, err := llm.NewProvider("deepseek", map[string]any{
		"api_key":           "your-api-key-here",
		"frequency_penalty": 1.0, // 强力减少重复词汇
		"presence_penalty":  1.0, // 强力鼓励新话题
		"temperature":       0.9, // 较高的随机性
	})
	if err != nil {
		log.Fatal(err)
	}

	// 生成文本
	ctx := context.Background()
	response, err := provider.Generate(
		ctx,
		"描述一个未来城市",
		"你是一个科幻作家",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", response)
}
