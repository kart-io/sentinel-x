// Package handler provides HTTP handlers for RAG service.
package handler

import (
	"net/http"

	"github.com/kart-io/sentinel-x/internal/rag/biz"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
)

// RAGHandler handles RAG HTTP requests.
type RAGHandler struct {
	ragService *biz.RAGService
}

// NewRAGHandler creates a new RAGHandler.
func NewRAGHandler(ragService *biz.RAGService) *RAGHandler {
	return &RAGHandler{
		ragService: ragService,
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
// @Summary      Index documents
// @Description  Download and index documents from a URL
// @Tags         RAG
// @Accept       json
// @Produce      json
// @Param        request  body      IndexRequest  true  "Index request"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/rag/index [post]
func (h *RAGHandler) Index(c transport.Context) {
	var req IndexRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.ragService.IndexFromURL(c.Request(), req.SourceURL); err != nil {
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
// @Summary      Index from URL
// @Description  Download and index documents from a URL
// @Tags         RAG
// @Accept       json
// @Produce      json
// @Param        request  body      IndexFromURLRequest  true  "Index from URL request"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/rag/index/url [post]
func (h *RAGHandler) IndexFromURL(c transport.Context) {
	var req IndexFromURLRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.ragService.IndexFromURL(c.Request(), req.URL); err != nil {
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
// @Summary      Index directory
// @Description  Index documents from a local directory
// @Tags         RAG
// @Accept       json
// @Produce      json
// @Param        request  body      IndexDirectoryRequest  true  "Index directory request"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/rag/index/directory [post]
func (h *RAGHandler) IndexDirectory(c transport.Context) {
	var req IndexDirectoryRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.ragService.IndexDirectory(c.Request(), req.Directory); err != nil {
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
// @Summary      Query knowledge base
// @Description  Ask a question and get an answer based on the knowledge base
// @Tags         RAG
// @Accept       json
// @Produce      json
// @Param        request  body      QueryRequest  true  "Query request"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/rag/query [post]
func (h *RAGHandler) Query(c transport.Context) {
	var req QueryRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}

	result, err := h.ragService.Query(c.Request(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: result})
}

// Stats returns knowledge base statistics.
// @Summary      Get statistics
// @Description  Get statistics about the knowledge base
// @Tags         RAG
// @Produce      json
// @Success      200      {object}  SuccessResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/rag/stats [get]
func (h *RAGHandler) Stats(c transport.Context) {
	stats, err := h.ragService.GetStats(c.Request())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Code: 0, Message: "success", Data: stats})
}
