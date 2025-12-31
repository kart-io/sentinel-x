package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kart-io/sentinel-x/internal/rag/biz"
	ragv1 "github.com/kart-io/sentinel-x/pkg/api/rag/v1"
)

// Handler implements the ragv1.RAGServiceServer interface.
type Handler struct {
	ragv1.UnimplementedRAGServiceServer
	service biz.Service
}

// NewHandler creates a new gRPC handler.
func NewHandler(service biz.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// IndexFromURL indexes documents from a URL.
func (h *Handler) IndexFromURL(ctx context.Context, req *ragv1.IndexFromURLRequest) (*emptypb.Empty, error) {
	if err := h.service.IndexFromURL(ctx, req.Url); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Query performs a RAG query.
func (h *Handler) Query(ctx context.Context, req *ragv1.QueryRequest) (*ragv1.QueryResponse, error) {
	result, err := h.service.Query(ctx, req.Question)
	if err != nil {
		return nil, err
	}

	sources := make([]*ragv1.ChunkSource, len(result.Sources))
	for i, source := range result.Sources {
		sources[i] = &ragv1.ChunkSource{
			DocumentId:   source.DocumentID,
			DocumentName: source.DocumentName,
			Section:      source.Section,
			Content:      source.Content,
			Score:        source.Score,
		}
	}

	return &ragv1.QueryResponse{
		Answer:  result.Answer,
		Sources: sources,
	}, nil
}

// GetStats returns statistics about the knowledge base.
func (h *Handler) GetStats(ctx context.Context, _ *emptypb.Empty) (*ragv1.GetStatsResponse, error) {
	stats, err := h.service.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	resp := &ragv1.GetStatsResponse{}
	if val, ok := stats["collection"].(string); ok {
		resp.Collection = val
	}
	if val, ok := stats["chunk_count"].(int64); ok {
		resp.ChunkCount = val
	}

	return resp, nil
}
