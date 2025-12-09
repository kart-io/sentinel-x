package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/goagent/llm"
	"github.com/kart-io/goagent/llm/providers"
)

func main() {
	provider, err := providers.NewKimiWithOptions(
		llm.WithModel("moonshot-v1-8k"))
	if err != nil {
		log.Fatalf("Failed to create Kimi provider: %v", err)
	}

	ctx := context.Background()
	resp, err := provider.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "What is deep learning?"},
			{Role: "assistant", Content: "Deep learning is a subset of machine learning that is concerned with algorithms and models that learn from data that is unstructured or unlabeled. It is a type of machine learning that is used to find patterns in data."},
			{Role: "user", Content: "What is the capital of France?"},
		},
	})
	if err != nil {
		log.Fatalf("Failed to complete: %v", err)
	}
	fmt.Println(resp.Content)
}
