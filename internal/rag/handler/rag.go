// Package handler provides HTTP handlers for RAG service.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/kart-io/sentinel-x/internal/pkg/rag/evaluator"
	"github.com/kart-io/sentinel-x/internal/rag/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// RAGHandler handles RAG HTTP requests.
type RAGHandler struct {
	service   biz.Service
	evaluator *evaluator.Evaluator
}

// NewRAGHandler creates a new RAGHandler.
func NewRAGHandler(service biz.Service, eval *evaluator.Evaluator) *RAGHandler {
	return &RAGHandler{
		service:   service,
		evaluator: eval,
	}
}

// SuccessResponse is a standard success response.
type SuccessResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// IndexRequest represents an index request.
type IndexRequest struct {
	SourceURL string `json:"source_url" binding:"required"`
}

// Index indexes documents from a URL.
func (h *RAGHandler) Index(c transport.Context) {
	var req IndexRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.service.IndexFromURL(c.Request(), req.SourceURL); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "Documents indexed successfully"})
}

// IndexFromURLRequest represents an index from URL request.
type IndexFromURLRequest struct {
	URL string `json:"url" binding:"required"`
}

// IndexFromURL downloads and indexes documents from a URL.
func (h *RAGHandler) IndexFromURL(c transport.Context) {
	var req IndexFromURLRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.service.IndexFromURL(c.Request(), req.URL); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "Documents indexed successfully"})
}

// IndexDirectoryRequest represents a directory index request.
type IndexDirectoryRequest struct {
	Directory string `json:"directory" binding:"required"`
}

// IndexDirectory indexes documents from a local directory.
func (h *RAGHandler) IndexDirectory(c transport.Context) {
	var req IndexDirectoryRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.service.IndexDirectory(c.Request(), req.Directory); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "Directory indexed successfully"})
}

// QueryRequest represents a query request.
type QueryRequest struct {
	Question string `json:"question" binding:"required"`
}

// Query performs a RAG query.
func (h *RAGHandler) Query(c transport.Context) {
	var req QueryRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	// 添加 60 秒超时控制
	ctx, cancel := context.WithTimeout(c.Request(), 60*time.Second)
	defer cancel()

	result, err := h.service.Query(ctx, req.Question)
	if err != nil {
		// 检查是否超时
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, ErrorResponse{
				Code:    408,
				Message: "Query timeout: the request took too long to process. Please try again or simplify your question.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: result})
}

// Stats returns knowledge base statistics.
func (h *RAGHandler) Stats(c transport.Context) {
	stats, err := h.service.GetStats(c.Request())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: stats})
}

// CollectionInfo 集合信息。
type CollectionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int64  `json:"count"`
}

// ListCollections 列出所有集合。
func (h *RAGHandler) ListCollections(c transport.Context) {
	// 获取统计信息
	stats, err := h.service.GetStats(c.Request())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	// 从 stats 中提取信息
	collectionName, _ := stats["collection"].(string)
	chunkCount, _ := stats["chunk_count"].(int64)

	// 返回集合列表(目前只有一个默认集合)
	collections := []CollectionInfo{
		{
			Name:        collectionName,
			Description: "RAG knowledge base collection",
			Count:       chunkCount,
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: collections})
}

// EvaluateRequest 评估请求。
type EvaluateRequest struct {
	Question    string   `json:"question" binding:"required"`
	Answer      string   `json:"answer" binding:"required"`
	Contexts    []string `json:"contexts" binding:"required"`
	GroundTruth string   `json:"ground_truth,omitempty"`
}

// Evaluate 评估 RAG 输出质量。
func (h *RAGHandler) Evaluate(c transport.Context) {
	var req EvaluateRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if h.evaluator == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: "Evaluator not initialized"})
		return
	}

	input := &evaluator.Input{
		Question:    req.Question,
		Answer:      req.Answer,
		Contexts:    req.Contexts,
		GroundTruth: req.GroundTruth,
	}

	result, err := h.evaluator.Evaluate(c.Request(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: result})
}

// QueryAndEvaluateRequest 查询并评估请求。
type QueryAndEvaluateRequest struct {
	Question    string `json:"question" binding:"required"`
	GroundTruth string `json:"ground_truth,omitempty"`
}

// QueryAndEvaluateResponse 查询并评估响应。
type QueryAndEvaluateResponse struct {
	Answer     string            `json:"answer"`
	Sources    interface{}       `json:"sources"`
	Evaluation *evaluator.Result `json:"evaluation"`
}

// QueryAndEvaluate 执行 RAG 查询并评估结果。
func (h *RAGHandler) QueryAndEvaluate(c transport.Context) {
	var req QueryAndEvaluateRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	result, err := h.service.Query(c.Request(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: "Query failed: " + err.Error()})
		return
	}

	contexts := make([]string, len(result.Sources))
	for i, source := range result.Sources {
		contexts[i] = source.Content
	}

	var evaluation *evaluator.Result
	if h.evaluator != nil {
		input := &evaluator.Input{
			Question:    req.Question,
			Answer:      result.Answer,
			Contexts:    contexts,
			GroundTruth: req.GroundTruth,
		}
		evaluation, _ = h.evaluator.Evaluate(c.Request(), input)
	}

	response := QueryAndEvaluateResponse{
		Answer:     result.Answer,
		Sources:    result.Sources,
		Evaluation: evaluation,
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: response})
}
