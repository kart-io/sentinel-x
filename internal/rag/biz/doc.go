// Package biz 提供 RAG 服务的业务逻辑层。
//
// 该包采用分层架构，将业务逻辑拆分为以下组件：
//   - Indexer: 负责文档索引（下载、解析、分块、嵌入）
//   - Retriever: 负责检索（向量搜索、增强、重排序）
//   - Generator: 负责生成（上下文构建、LLM 回答生成）
//   - Service: 组合以上组件，提供统一的服务接口
package biz
