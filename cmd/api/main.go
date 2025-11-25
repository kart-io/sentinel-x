package main

import (
	"time"

	"github.com/kart-io/k8s-agent/pkg/agent/builder"
	"github.com/kart-io/k8s-agent/pkg/llm"
)

func main() {
	// Entry point for the API server
	// Create LLM client
	llmClient := llm.NewOpenAIClient("your-api-key")

	// Build agent with fluent API
	agent, err := builder.NewAgentBuilder(llmClient).
		WithSystemPrompt("You are a helpful assistant").
		WithMaxIterations(10).
		WithTimeout(30 * time.Second).
		Build()
	if err != nil {
		panic(err)
	}

	// Start the agent
	if err := agent.Start(); err != nil {
		panic(err)
	}
}
