// Package metrics provides business metrics collection for the RAG service.
package metrics

import (
	"sync"
	"time"

	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
)

// RAGMetrics RAG service business metrics.
type RAGMetrics struct {
	// Query metrics
	queriesTotal       metrics.Counter
	queriesCacheHits   metrics.Counter
	queriesCacheMisses metrics.Counter
	queriesErrors      metrics.Counter

	// Retrieval metrics
	retrievalTotal    metrics.Counter
	retrievalDuration metrics.Counter // seconds
	retrievalErrors   metrics.Counter

	// LLM metrics
	llmCallsTotal       metrics.Counter
	llmCallsDuration    metrics.Counter // seconds
	llmCallsErrors      metrics.Counter
	llmCallsRetries     metrics.Counter
	llmTokensPrompt     metrics.Counter
	llmTokensCompletion metrics.Counter

	// Circuit breaker metrics
	circuitBreakerOpens metrics.Counter
	circuitBreakerState metrics.Gauge // 0=closed, 1=open, 2=half-open

	// Index metrics
	documentsIndexed metrics.Counter
	chunksIndexed    metrics.Counter
	indexErrors      metrics.Counter

	startTime time.Time
}

// globalRAGMetrics Global RAG metrics instance.
var (
	globalRAGMetrics *RAGMetrics
	ragMetricsOnce   sync.Once
)

// GetRAGMetrics returns the global RAG metrics instance.
func GetRAGMetrics() *RAGMetrics {
	ragMetricsOnce.Do(func() {
		m := &RAGMetrics{
			startTime: time.Now(),
		}
		prefix := "rag"

		// Query metrics
		m.queriesTotal = metrics.NewCounter(prefix+"_queries_total", "Total number of RAG queries.")
		metrics.Register(m.queriesTotal)

		m.queriesCacheHits = metrics.NewCounter(prefix+"_queries_cache_hits_total", "Number of cache hits.")
		metrics.Register(m.queriesCacheHits)

		m.queriesCacheMisses = metrics.NewCounter(prefix+"_queries_cache_misses_total", "Number of cache misses.")
		metrics.Register(m.queriesCacheMisses)

		m.queriesErrors = metrics.NewCounter(prefix+"_queries_errors_total", "Number of query errors.")
		metrics.Register(m.queriesErrors)

		// Derived metric: Cache hit rate is calculated on the fly in Export/Stats, no need to register gauge

		// Retrieval metrics
		m.retrievalTotal = metrics.NewCounter(prefix+"_retrieval_total", "Total number of retrievals.")
		metrics.Register(m.retrievalTotal)

		m.retrievalDuration = metrics.NewCounter(prefix+"_retrieval_duration_seconds_total", "Total retrieval duration.")
		metrics.Register(m.retrievalDuration)

		m.retrievalErrors = metrics.NewCounter(prefix+"_retrieval_errors_total", "Number of retrieval errors.")
		metrics.Register(m.retrievalErrors)

		// LLM metrics
		m.llmCallsTotal = metrics.NewCounter(prefix+"_llm_calls_total", "Total number of LLM calls.")
		metrics.Register(m.llmCallsTotal)

		m.llmCallsDuration = metrics.NewCounter(prefix+"_llm_calls_duration_seconds_total", "Total LLM call duration.")
		metrics.Register(m.llmCallsDuration)

		m.llmCallsErrors = metrics.NewCounter(prefix+"_llm_calls_errors_total", "Number of LLM call errors.")
		metrics.Register(m.llmCallsErrors)

		m.llmCallsRetries = metrics.NewCounter(prefix+"_llm_calls_retries_total", "Number of LLM call retries.")
		metrics.Register(m.llmCallsRetries)

		m.llmTokensPrompt = metrics.NewCounter(prefix+"_llm_tokens_prompt_total", "Total prompt tokens.")
		metrics.Register(m.llmTokensPrompt)

		m.llmTokensCompletion = metrics.NewCounter(prefix+"_llm_tokens_completion_total", "Total completion tokens.")
		metrics.Register(m.llmTokensCompletion)

		// Circuit breaker
		m.circuitBreakerOpens = metrics.NewCounter(prefix+"_circuit_breaker_opens_total", "Number of circuit breaker opens.")
		metrics.Register(m.circuitBreakerOpens)

		m.circuitBreakerState = metrics.NewGauge(prefix+"_circuit_breaker_state", "Circuit breaker state (0=closed, 1=open, 2=half-open).")
		metrics.Register(m.circuitBreakerState)

		// Indexing
		m.documentsIndexed = metrics.NewCounter(prefix+"_documents_indexed_total", "Total documents indexed.")
		metrics.Register(m.documentsIndexed)

		m.chunksIndexed = metrics.NewCounter(prefix+"_chunks_indexed_total", "Total chunks indexed.")
		metrics.Register(m.chunksIndexed)

		m.indexErrors = metrics.NewCounter(prefix+"_index_errors_total", "Number of indexing errors.")
		metrics.Register(m.indexErrors)

		// Uptime
		uptime := metrics.NewGauge(prefix+"_uptime_seconds", "Service uptime in seconds.")
		uptime.Set(time.Since(m.startTime).Seconds())
		metrics.Register(uptime) // Note: this is static at init time, real uptime is dynamic

		globalRAGMetrics = m
	})
	return globalRAGMetrics
}

// RecordQuery records a query.
func (m *RAGMetrics) RecordQuery(cacheHit bool, err error) {
	m.queriesTotal.Inc()
	if err != nil {
		m.queriesErrors.Inc()
		return
	}
	if cacheHit {
		m.queriesCacheHits.Inc()
	} else {
		m.queriesCacheMisses.Inc()
	}
}

// RecordRetrieval records a retrieval operation.
func (m *RAGMetrics) RecordRetrieval(duration time.Duration, err error) {
	m.retrievalTotal.Inc()
	m.retrievalDuration.Add(duration.Seconds())
	if err != nil {
		m.retrievalErrors.Inc()
		return
	}
}

// RecordLLMCall records an LLM call.
func (m *RAGMetrics) RecordLLMCall(duration time.Duration, promptTokens, completionTokens int, err error) {
	m.llmCallsTotal.Inc()
	m.llmCallsDuration.Add(duration.Seconds())
	if err != nil {
		m.llmCallsErrors.Inc()
		return
	}

	if promptTokens > 0 {
		m.llmTokensPrompt.Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		m.llmTokensCompletion.Add(float64(completionTokens))
	}
}

// RecordLLMRetry records an LLM retry.
func (m *RAGMetrics) RecordLLMRetry() {
	m.llmCallsRetries.Inc()
}

// RecordCircuitBreakerOpen records circuit breaker open.
func (m *RAGMetrics) RecordCircuitBreakerOpen() {
	m.circuitBreakerOpens.Inc()
	m.circuitBreakerState.Set(1) // Open
}

// RecordCircuitBreakerClosed records circuit breaker closed.
func (m *RAGMetrics) RecordCircuitBreakerClosed() {
	m.circuitBreakerState.Set(0) // Closed
}

// RecordCircuitBreakerHalfOpen records circuit breaker half-open.
func (m *RAGMetrics) RecordCircuitBreakerHalfOpen() {
	m.circuitBreakerState.Set(2) // Half-Open
}

// RecordIndexing records indexing operation.
func (m *RAGMetrics) RecordIndexing(documents, chunks int, err error) {
	if err != nil {
		m.indexErrors.Inc()
		return
	}
	if documents > 0 {
		m.documentsIndexed.Add(float64(documents))
	}
	if chunks > 0 {
		m.chunksIndexed.Add(float64(chunks))
	}
}

// Export exports metrics in Prometheus format.
//
// Deprecated: Use metrics.Export() via the global registry instead.
func (m *RAGMetrics) Export(_, _ string) string {
	// For backward compatibility if needed, or redirect to global export
	// Note: namespace/subsystem arguments are ignored as metrics are already registered with names
	return metrics.Export()
}

// Stats returns current statistics (for API).
func (m *RAGMetrics) Stats() map[string]interface{} {
	queriesTotal := uint64(m.queriesTotal.Get())
	cacheHits := uint64(m.queriesCacheHits.Get())
	cacheMisses := uint64(m.queriesCacheMisses.Get())
	cacheTotal := cacheHits + cacheMisses
	cacheHitRate := 0.0
	if cacheTotal > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheTotal)
	}

	retrievalTotal := uint64(m.retrievalTotal.Get())
	retrievalDuration := m.retrievalDuration.Get()
	avgRetrievalDuration := 0.0
	if retrievalTotal > 0 {
		avgRetrievalDuration = retrievalDuration / float64(retrievalTotal)
	}

	llmTotal := uint64(m.llmCallsTotal.Get())
	llmDuration := m.llmCallsDuration.Get()
	avgLLMDuration := 0.0
	if llmTotal > 0 {
		avgLLMDuration = llmDuration / float64(llmTotal)
	}

	cbState := int(m.circuitBreakerState.Get())
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
			"errors":         uint64(m.queriesErrors.Get()),
		},
		"retrieval": map[string]interface{}{
			"total":               retrievalTotal,
			"total_duration_secs": retrievalDuration,
			"avg_duration_secs":   avgRetrievalDuration,
			"errors":              uint64(m.retrievalErrors.Get()),
		},
		"llm": map[string]interface{}{
			"calls_total":         llmTotal,
			"total_duration_secs": llmDuration,
			"avg_duration_secs":   avgLLMDuration,
			"errors":              uint64(m.llmCallsErrors.Get()),
			"retries":             uint64(m.llmCallsRetries.Get()),
			"tokens_prompt":       uint64(m.llmTokensPrompt.Get()),
			"tokens_completion":   uint64(m.llmTokensCompletion.Get()),
		},
		"circuit_breaker": map[string]interface{}{
			"state": cbStateStr,
			"opens": uint64(m.circuitBreakerOpens.Get()),
		},
		"indexing": map[string]interface{}{
			"documents_indexed": uint64(m.documentsIndexed.Get()),
			"chunks_indexed":    uint64(m.chunksIndexed.Get()),
			"errors":            uint64(m.indexErrors.Get()),
		},
		"uptime_seconds": time.Since(m.startTime).Seconds(),
	}
}

// Reset resets all metrics (for testing).
func (m *RAGMetrics) Reset() {
	metrics.DefaultRegistry.Reset()
	// Re-initialize to register metrics again since Reset clears the registry
	ragMetricsOnce = sync.Once{}
	GetRAGMetrics()
}
