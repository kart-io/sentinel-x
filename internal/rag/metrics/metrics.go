// Package metrics 提供 RAG 服务的业务指标收集。
package metrics

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RAGMetrics RAG 服务业务指标。
type RAGMetrics struct {
	// 查询指标
	queriesTotal       uint64 // 总查询次数
	queriesCacheHits   uint64 // 缓存命中次数
	queriesCacheMisses uint64 // 缓存未命中次数
	queriesErrors      uint64 // 查询错误次数

	// 检索指标
	retrievalTotal    uint64  // 总检索次数
	retrievalDuration float64 // 检索总耗时（秒）
	retrievalErrors   uint64  // 检索错误次数

	// LLM 调用指标
	llmCallsTotal     uint64  // LLM 总调用次数
	llmCallsDuration  float64 // LLM 调用总耗时（秒）
	llmCallsErrors    uint64  // LLM 调用错误次数
	llmCallsRetries   uint64  // LLM 重试次数
	llmTokensPrompt   uint64  // Prompt tokens 总数
	llmTokensCompletion uint64 // Completion tokens 总数

	// 熔断器指标
	circuitBreakerOpens uint64 // 熔断器打开次数
	circuitBreakerState int32  // 熔断器当前状态 (0=closed, 1=open, 2=half-open)

	// 索引指标
	documentsIndexed uint64 // 已索引文档数
	chunksIndexed    uint64 // 已索引分块数
	indexErrors      uint64 // 索引错误次数

	mu         sync.RWMutex
	startTime  time.Time
	durationMu sync.Mutex
}

// globalRAGMetrics 全局 RAG 指标实例。
var (
	globalRAGMetrics *RAGMetrics
	ragMetricsOnce   sync.Once
)

// GetRAGMetrics 获取全局 RAG 指标实例。
func GetRAGMetrics() *RAGMetrics {
	ragMetricsOnce.Do(func() {
		globalRAGMetrics = &RAGMetrics{
			startTime: time.Now(),
		}
	})
	return globalRAGMetrics
}

// RecordQuery 记录查询。
func (m *RAGMetrics) RecordQuery(cacheHit bool, err error) {
	atomic.AddUint64(&m.queriesTotal, 1)
	if err != nil {
		atomic.AddUint64(&m.queriesErrors, 1)
		return
	}
	if cacheHit {
		atomic.AddUint64(&m.queriesCacheHits, 1)
	} else {
		atomic.AddUint64(&m.queriesCacheMisses, 1)
	}
}

// RecordRetrieval 记录检索操作。
func (m *RAGMetrics) RecordRetrieval(duration time.Duration, err error) {
	atomic.AddUint64(&m.retrievalTotal, 1)
	if err != nil {
		atomic.AddUint64(&m.retrievalErrors, 1)
		return
	}

	m.durationMu.Lock()
	m.retrievalDuration += duration.Seconds()
	m.durationMu.Unlock()
}

// RecordLLMCall 记录 LLM 调用。
func (m *RAGMetrics) RecordLLMCall(duration time.Duration, promptTokens, completionTokens int, err error) {
	atomic.AddUint64(&m.llmCallsTotal, 1)
	if err != nil {
		atomic.AddUint64(&m.llmCallsErrors, 1)
		return
	}

	m.durationMu.Lock()
	m.llmCallsDuration += duration.Seconds()
	m.durationMu.Unlock()

	if promptTokens > 0 {
		atomic.AddUint64(&m.llmTokensPrompt, uint64(promptTokens))
	}
	if completionTokens > 0 {
		atomic.AddUint64(&m.llmTokensCompletion, uint64(completionTokens))
	}
}

// RecordLLMRetry 记录 LLM 重试。
func (m *RAGMetrics) RecordLLMRetry() {
	atomic.AddUint64(&m.llmCallsRetries, 1)
}

// RecordCircuitBreakerOpen 记录熔断器打开。
func (m *RAGMetrics) RecordCircuitBreakerOpen() {
	atomic.AddUint64(&m.circuitBreakerOpens, 1)
	atomic.StoreInt32(&m.circuitBreakerState, 1) // Open
}

// RecordCircuitBreakerClosed 记录熔断器关闭。
func (m *RAGMetrics) RecordCircuitBreakerClosed() {
	atomic.StoreInt32(&m.circuitBreakerState, 0) // Closed
}

// RecordCircuitBreakerHalfOpen 记录熔断器半开。
func (m *RAGMetrics) RecordCircuitBreakerHalfOpen() {
	atomic.StoreInt32(&m.circuitBreakerState, 2) // Half-Open
}

// RecordIndexing 记录索引操作。
func (m *RAGMetrics) RecordIndexing(documents, chunks int, err error) {
	if err != nil {
		atomic.AddUint64(&m.indexErrors, 1)
		return
	}
	atomic.AddUint64(&m.documentsIndexed, uint64(documents))
	atomic.AddUint64(&m.chunksIndexed, uint64(chunks))
}

// Export 导出 Prometheus 格式指标。
func (m *RAGMetrics) Export(namespace, subsystem string) string {
	var sb strings.Builder
	prefix := namespace
	if subsystem != "" {
		prefix = prefix + "_" + subsystem
	}

	// 查询指标
	sb.WriteString(fmt.Sprintf("# HELP %s_queries_total Total number of RAG queries.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_queries_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_queries_total %d\n", prefix, atomic.LoadUint64(&m.queriesTotal)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_queries_cache_hits_total Number of cache hits.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_queries_cache_hits_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_queries_cache_hits_total %d\n", prefix, atomic.LoadUint64(&m.queriesCacheHits)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_queries_cache_misses_total Number of cache misses.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_queries_cache_misses_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_queries_cache_misses_total %d\n", prefix, atomic.LoadUint64(&m.queriesCacheMisses)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_queries_errors_total Number of query errors.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_queries_errors_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_queries_errors_total %d\n", prefix, atomic.LoadUint64(&m.queriesErrors)))
	sb.WriteString("\n")

	// 缓存命中率
	cacheHits := atomic.LoadUint64(&m.queriesCacheHits)
	cacheMisses := atomic.LoadUint64(&m.queriesCacheMisses)
	total := cacheHits + cacheMisses
	cacheHitRate := 0.0
	if total > 0 {
		cacheHitRate = float64(cacheHits) / float64(total)
	}
	sb.WriteString(fmt.Sprintf("# HELP %s_cache_hit_rate Cache hit rate (0-1).\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_cache_hit_rate gauge\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_cache_hit_rate %.4f\n", prefix, cacheHitRate))
	sb.WriteString("\n")

	// 检索指标
	sb.WriteString(fmt.Sprintf("# HELP %s_retrieval_total Total number of retrievals.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_retrieval_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_retrieval_total %d\n", prefix, atomic.LoadUint64(&m.retrievalTotal)))
	sb.WriteString("\n")

	m.durationMu.Lock()
	retrievalDuration := m.retrievalDuration
	llmDuration := m.llmCallsDuration
	m.durationMu.Unlock()

	sb.WriteString(fmt.Sprintf("# HELP %s_retrieval_duration_seconds_total Total retrieval duration.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_retrieval_duration_seconds_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_retrieval_duration_seconds_total %.6f\n", prefix, retrievalDuration))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_retrieval_errors_total Number of retrieval errors.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_retrieval_errors_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_retrieval_errors_total %d\n", prefix, atomic.LoadUint64(&m.retrievalErrors)))
	sb.WriteString("\n")

	// LLM 调用指标
	sb.WriteString(fmt.Sprintf("# HELP %s_llm_calls_total Total number of LLM calls.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_llm_calls_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_llm_calls_total %d\n", prefix, atomic.LoadUint64(&m.llmCallsTotal)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_llm_calls_duration_seconds_total Total LLM call duration.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_llm_calls_duration_seconds_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_llm_calls_duration_seconds_total %.6f\n", prefix, llmDuration))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_llm_calls_errors_total Number of LLM call errors.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_llm_calls_errors_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_llm_calls_errors_total %d\n", prefix, atomic.LoadUint64(&m.llmCallsErrors)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_llm_calls_retries_total Number of LLM call retries.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_llm_calls_retries_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_llm_calls_retries_total %d\n", prefix, atomic.LoadUint64(&m.llmCallsRetries)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_llm_tokens_prompt_total Total prompt tokens.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_llm_tokens_prompt_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_llm_tokens_prompt_total %d\n", prefix, atomic.LoadUint64(&m.llmTokensPrompt)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_llm_tokens_completion_total Total completion tokens.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_llm_tokens_completion_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_llm_tokens_completion_total %d\n", prefix, atomic.LoadUint64(&m.llmTokensCompletion)))
	sb.WriteString("\n")

	// 熔断器指标
	sb.WriteString(fmt.Sprintf("# HELP %s_circuit_breaker_opens_total Number of circuit breaker opens.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_circuit_breaker_opens_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_circuit_breaker_opens_total %d\n", prefix, atomic.LoadUint64(&m.circuitBreakerOpens)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_circuit_breaker_state Circuit breaker state (0=closed, 1=open, 2=half-open).\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_circuit_breaker_state gauge\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_circuit_breaker_state %d\n", prefix, atomic.LoadInt32(&m.circuitBreakerState)))
	sb.WriteString("\n")

	// 索引指标
	sb.WriteString(fmt.Sprintf("# HELP %s_documents_indexed_total Total documents indexed.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_documents_indexed_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_documents_indexed_total %d\n", prefix, atomic.LoadUint64(&m.documentsIndexed)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_chunks_indexed_total Total chunks indexed.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_chunks_indexed_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_chunks_indexed_total %d\n", prefix, atomic.LoadUint64(&m.chunksIndexed)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("# HELP %s_index_errors_total Number of indexing errors.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_index_errors_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_index_errors_total %d\n", prefix, atomic.LoadUint64(&m.indexErrors)))
	sb.WriteString("\n")

	// 运行时间
	uptime := time.Since(m.startTime).Seconds()
	sb.WriteString(fmt.Sprintf("# HELP %s_uptime_seconds Service uptime in seconds.\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_uptime_seconds gauge\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_uptime_seconds %.2f\n", prefix, uptime))
	sb.WriteString("\n")

	return sb.String()
}

// Stats 返回当前统计信息（用于 API）。
func (m *RAGMetrics) Stats() map[string]interface{} {
	m.durationMu.Lock()
	retrievalDuration := m.retrievalDuration
	llmDuration := m.llmCallsDuration
	m.durationMu.Unlock()

	queriesTotal := atomic.LoadUint64(&m.queriesTotal)
	cacheHits := atomic.LoadUint64(&m.queriesCacheHits)
	cacheMisses := atomic.LoadUint64(&m.queriesCacheMisses)
	cacheTotal := cacheHits + cacheMisses
	cacheHitRate := 0.0
	if cacheTotal > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheTotal)
	}

	retrievalTotal := atomic.LoadUint64(&m.retrievalTotal)
	avgRetrievalDuration := 0.0
	if retrievalTotal > 0 {
		avgRetrievalDuration = retrievalDuration / float64(retrievalTotal)
	}

	llmTotal := atomic.LoadUint64(&m.llmCallsTotal)
	avgLLMDuration := 0.0
	if llmTotal > 0 {
		avgLLMDuration = llmDuration / float64(llmTotal)
	}

	cbState := atomic.LoadInt32(&m.circuitBreakerState)
	cbStateStr := "closed"
	switch cbState {
	case 1:
		cbStateStr = "open"
	case 2:
		cbStateStr = "half-open"
	}

	return map[string]interface{}{
		"queries": map[string]interface{}{
			"total":          queriesTotal,
			"cache_hits":     cacheHits,
			"cache_misses":   cacheMisses,
			"cache_hit_rate": cacheHitRate,
			"errors":         atomic.LoadUint64(&m.queriesErrors),
		},
		"retrieval": map[string]interface{}{
			"total":                retrievalTotal,
			"total_duration_secs":  retrievalDuration,
			"avg_duration_secs":    avgRetrievalDuration,
			"errors":               atomic.LoadUint64(&m.retrievalErrors),
		},
		"llm": map[string]interface{}{
			"calls_total":          llmTotal,
			"total_duration_secs":  llmDuration,
			"avg_duration_secs":    avgLLMDuration,
			"errors":               atomic.LoadUint64(&m.llmCallsErrors),
			"retries":              atomic.LoadUint64(&m.llmCallsRetries),
			"tokens_prompt":        atomic.LoadUint64(&m.llmTokensPrompt),
			"tokens_completion":    atomic.LoadUint64(&m.llmTokensCompletion),
		},
		"circuit_breaker": map[string]interface{}{
			"state": cbStateStr,
			"opens": atomic.LoadUint64(&m.circuitBreakerOpens),
		},
		"indexing": map[string]interface{}{
			"documents_indexed": atomic.LoadUint64(&m.documentsIndexed),
			"chunks_indexed":    atomic.LoadUint64(&m.chunksIndexed),
			"errors":            atomic.LoadUint64(&m.indexErrors),
		},
		"uptime_seconds": time.Since(m.startTime).Seconds(),
	}
}

// Reset 重置所有指标（仅用于测试）。
func (m *RAGMetrics) Reset() {
	atomic.StoreUint64(&m.queriesTotal, 0)
	atomic.StoreUint64(&m.queriesCacheHits, 0)
	atomic.StoreUint64(&m.queriesCacheMisses, 0)
	atomic.StoreUint64(&m.queriesErrors, 0)
	atomic.StoreUint64(&m.retrievalTotal, 0)
	atomic.StoreUint64(&m.retrievalErrors, 0)
	atomic.StoreUint64(&m.llmCallsTotal, 0)
	atomic.StoreUint64(&m.llmCallsErrors, 0)
	atomic.StoreUint64(&m.llmCallsRetries, 0)
	atomic.StoreUint64(&m.llmTokensPrompt, 0)
	atomic.StoreUint64(&m.llmTokensCompletion, 0)
	atomic.StoreUint64(&m.circuitBreakerOpens, 0)
	atomic.StoreInt32(&m.circuitBreakerState, 0)
	atomic.StoreUint64(&m.documentsIndexed, 0)
	atomic.StoreUint64(&m.chunksIndexed, 0)
	atomic.StoreUint64(&m.indexErrors, 0)

	m.durationMu.Lock()
	m.retrievalDuration = 0
	m.llmCallsDuration = 0
	m.startTime = time.Now()
	m.durationMu.Unlock()
}
