package biz_test

import (
	"context"
	"testing"

	"github.com/kart-io/sentinel-x/internal/rag/biz"
	"github.com/stretchr/testify/assert"
)

func TestRAGConfig(t *testing.T) {
	cfg := &biz.RAGConfig{
		ChunkSize:    512,
		ChunkOverlap: 64,
		TopK:         5,
		Collection:   "test_collection",
		EmbeddingDim: 768,
		DataDir:      "/tmp/rag_data",
		SystemPrompt: "You are a helpful assistant.",
	}

	assert.Equal(t, 512, cfg.ChunkSize)
	assert.Equal(t, 64, cfg.ChunkOverlap)
	assert.Equal(t, "test_collection", cfg.Collection)
}

func TestProcessResult(t *testing.T) {
	// A placeholder test to verify testing infrastructure works
	ctx := context.Background()
	assert.NotNil(t, ctx)
}
