package errors

import "google.golang.org/grpc/codes"

// RAG 服务代码: 20 (业务服务范围 20-79)
// 错误码格式: AABBCCC
// - AA: 20 (RAG 服务)
// - BB: 类别代码
// - CCC: 序号

const (
	// ServiceRAG is for RAG service.
	ServiceRAG = 20
)

var (
	// 请求参数错误 (类别 01)
	ErrRAGInvalidRequest   = Register(New(MakeCode(ServiceRAG, CategoryRequest, 1), 400, codes.InvalidArgument, "Invalid request parameters", "请求参数无效"))
	ErrRAGInvalidURL       = Register(New(MakeCode(ServiceRAG, CategoryRequest, 2), 400, codes.InvalidArgument, "Invalid URL format", "URL 格式无效"))
	ErrRAGInvalidDirectory = Register(New(MakeCode(ServiceRAG, CategoryRequest, 3), 400, codes.InvalidArgument, "Invalid directory path", "目录路径无效"))

	// 查询相关错误 (类别 07 - Internal)
	ErrRAGQueryTimeout = Register(New(MakeCode(ServiceRAG, CategoryTimeout, 1), 408, codes.DeadlineExceeded, "Query timeout", "查询超时"))
	ErrRAGQueryFailed  = Register(New(MakeCode(ServiceRAG, CategoryInternal, 1), 500, codes.Internal, "Query failed", "查询失败"))
	ErrRAGNoResults    = Register(New(MakeCode(ServiceRAG, CategoryResource, 1), 404, codes.NotFound, "No results found", "未找到结果"))

	// 索引相关错误 (类别 07 - Internal)
	ErrRAGIndexFailed    = Register(New(MakeCode(ServiceRAG, CategoryInternal, 2), 500, codes.Internal, "Document indexing failed", "文档索引失败"))
	ErrRAGDownloadFailed = Register(New(MakeCode(ServiceRAG, CategoryNetwork, 1), 500, codes.Internal, "Document download failed", "文档下载失败"))
	ErrRAGProcessFailed  = Register(New(MakeCode(ServiceRAG, CategoryInternal, 3), 500, codes.Internal, "Document processing failed", "文档处理失败"))

	// 服务相关错误 (类别 10 - Network)
	ErrRAGServiceUnavailable = Register(New(MakeCode(ServiceRAG, CategoryNetwork, 2), 503, codes.Unavailable, "RAG service unavailable", "RAG 服务不可用"))
	ErrRAGEvaluatorNotInit   = Register(New(MakeCode(ServiceRAG, CategoryInternal, 4), 500, codes.FailedPrecondition, "Evaluator not initialized", "评估器未初始化"))
	ErrRAGStatsUnavailable   = Register(New(MakeCode(ServiceRAG, CategoryInternal, 5), 500, codes.Internal, "Statistics unavailable", "统计信息不可用"))
)
