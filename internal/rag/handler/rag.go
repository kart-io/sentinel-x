// Package handler provides HTTP handlers for RAG service.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/evaluator"
	"github.com/kart-io/sentinel-x/internal/rag/biz"
	observability "github.com/kart-io/sentinel-x/pkg/observability/metrics"
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
	Code    int         `json:"code" example:"0"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"bad request"`
}

// IndexRequest represents an index request.
type IndexRequest struct {
	SourceURL string `json:"source_url" binding:"required" example:"https://milvus.io/docs/overview.md"`
}

// Index indexes documents from a URL.
// @Summary 从 URL 索引文档
// @Description 下载并处理指定 URL 的文档，将其切片并存储到向量数据库中。
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body IndexRequest true "索引请求"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/index [post]
func (h *RAGHandler) Index(c *gin.Context) {
	var req IndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.service.IndexFromURL(c.Request.Context(), req.SourceURL); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "Documents indexed successfully"})
}

// IndexFromURLRequest represents an index from URL request.
type IndexFromURLRequest struct {
	URL string `json:"url" binding:"required" example:"https://milvus.io/docs/overview.md"`
}

// IndexFromURL downloads and indexes documents from a URL.
// @Summary 从 URL 索引文档 (别名)
// @Description 下载并处理指定 URL 的文档，将其切片并存储到向量数据库中。
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body IndexFromURLRequest true "索引请求"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/index/url [post]
func (h *RAGHandler) IndexFromURL(c *gin.Context) {
	var req IndexFromURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.service.IndexFromURL(c.Request.Context(), req.URL); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "Documents indexed successfully"})
}

// IndexDirectoryRequest represents a directory index request.
type IndexDirectoryRequest struct {
	Directory string `json:"directory" binding:"required" example:"/data/docs"`
}

// IndexDirectory indexes documents from a local directory.
// @Summary 从本地目录索引文档
// @Description 处理指定本地目录下的所有支持的文档，将其切片并存储到向量数据库中。
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body IndexDirectoryRequest true "目录索引请求"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/index/directory [post]
func (h *RAGHandler) IndexDirectory(c *gin.Context) {
	var req IndexDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.service.IndexDirectory(c.Request.Context(), req.Directory); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "Directory indexed successfully"})
}

// QueryRequest represents a query request.
type QueryRequest struct {
	Question string `json:"question" binding:"required" example:"什么是 Milvus？"`
}

// Query performs a RAG query.
// @Summary 执行 RAG 查询
// @Description 根据用户提出的问题，从知识库中检索相关上下文，并生成回答。
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body QueryRequest true "查询请求"
// @Success 200 {object} SuccessResponse{data=biz.QueryResult}
// @Failure 400 {object} ErrorResponse
// @Failure 408 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/query [post]
func (h *RAGHandler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	// 添加 60 秒超时控制
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
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
// @Summary 获取知识库统计信息
// @Description 返回当前向量数据库的集合名称、文本片段总数等统计数据。
// @Tags RAG
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/stats [get]
func (h *RAGHandler) Stats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: stats})
}

// CollectionInfo 集合信息。
type CollectionInfo struct {
	Name        string `json:"name" example:"milvus_docs"`
	Description string `json:"description" example:"RAG knowledge base collection"`
	Count       int64  `json:"count" example:"5000"`
}

// ListCollections 列出所有集合。
// @Summary 列出所有知识库集合
// @Description 返回当前系统中配置的所有知识库集合及其详细信息。
// @Tags RAG
// @Produce json
// @Success 200 {object} SuccessResponse{data=[]CollectionInfo}
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/collections [get]
func (h *RAGHandler) ListCollections(c *gin.Context) {
	// 获取统计信息
	stats, err := h.service.GetStats(c.Request.Context())
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
	Question    string   `json:"question" binding:"required" example:"什么是 RAG？"`
	Answer      string   `json:"answer" binding:"required" example:"RAG 是检索增强生成。"`
	Contexts    []string `json:"contexts" binding:"required" example:"['上下文片段1']"`
	GroundTruth string   `json:"ground_truth,omitempty" example:"检索增强生成 (Retrieval-Augmented Generation)"`
}

// Evaluate 评估 RAG 输出 quality。
// @Summary 评估 RAG 回答质量
// @Description 使用评估器（通常是另一个 LLM）对生成的回答、检索的上下文进行评分。
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body EvaluateRequest true "评估请求"
// @Success 200 {object} SuccessResponse{data=evaluator.Result}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/evaluate [post]
func (h *RAGHandler) Evaluate(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	result, err := h.evaluator.Evaluate(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: result})
}

// QueryAndEvaluateRequest 查询并评估请求。
type QueryAndEvaluateRequest struct {
	Question    string `json:"question" binding:"required" example:"Sentinel-X 是什么？"`
	GroundTruth string `json:"ground_truth,omitempty" example:"微服务平台"`
}

// QueryAndEvaluateResponse 查询并评估响应。
type QueryAndEvaluateResponse struct {
	Answer     string            `json:"answer" example:"回答内容"`
	Sources    interface{}       `json:"sources"`
	Evaluation *evaluator.Result `json:"evaluation"`
}

// QueryAndEvaluate 执行 RAG 查询并评估结果。
// @Summary 查询并即时评估
// @Description 执行 RAG 查询流程，并在生成结果后立即进行质量评估。
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body QueryAndEvaluateRequest true "查询评估请求"
// @Success 200 {object} SuccessResponse{data=QueryAndEvaluateResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/rag/query-evaluate [post]
func (h *RAGHandler) QueryAndEvaluate(c *gin.Context) {
	var req QueryAndEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	result, err := h.service.Query(c.Request.Context(), req.Question)
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
		evaluation, _ = h.evaluator.Evaluate(c.Request.Context(), input)
	}

	response := QueryAndEvaluateResponse{
		Answer:     result.Answer,
		Sources:    result.Sources,
		Evaluation: evaluation,
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: response})
}

// Metrics 导出 Prometheus 格式的业务指标。
// @Summary 导出业务指标
// @Description 返回 RAG 服务运行期间产生的各种业务指标（Prometheus 格式）。
// @Tags RAG
// @Produce plain
// @Success 200 {string} string "metrics data"
// @Router /v1/rag/metrics [get]
func (h *RAGHandler) Metrics(c *gin.Context) {
	metricsData := observability.Export()
	c.String(http.StatusOK, metricsData)
}
