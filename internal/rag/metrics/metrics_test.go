package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a new RAGMetrics with initialized counters
func newTestMetrics() *RAGMetrics {
	// Use GetRAGMetrics to ensure proper initialization through sync.Once
	// We need to reset it first to ensure clean state
	m := GetRAGMetrics()
	m.Reset()
	// Reset creates a NEW global instance and updates globalRAGMetrics
	// So we must call GetRAGMetrics AGAIN to get the new instance
	return GetRAGMetrics()
}

func TestGetRAGMetrics(t *testing.T) {
	// 获取全局单例
	m1 := GetRAGMetrics()
	m2 := GetRAGMetrics()

	// 应该返回同一个实例
	assert.Equal(t, m1, m2, "应该返回同一个单例实例")
}

func TestRecordQuery(t *testing.T) {
	m := newTestMetrics()

	// 测试成功查询（缓存命中）
	m.RecordQuery(true, nil)
	assert.Equal(t, float64(1), m.queriesTotal.Get())
	assert.Equal(t, float64(1), m.queriesCacheHits.Get())
	assert.Equal(t, float64(0), m.queriesCacheMisses.Get())
	assert.Equal(t, float64(0), m.queriesErrors.Get())

	// 测试成功查询（缓存未命中）
	m.RecordQuery(false, nil)
	assert.Equal(t, float64(2), m.queriesTotal.Get())
	assert.Equal(t, float64(1), m.queriesCacheHits.Get())
	assert.Equal(t, float64(1), m.queriesCacheMisses.Get())
	assert.Equal(t, float64(0), m.queriesErrors.Get())

	// 测试失败查询
	m.RecordQuery(false, assert.AnError)
	assert.Equal(t, float64(3), m.queriesTotal.Get())
	assert.Equal(t, float64(1), m.queriesErrors.Get())
}

func TestRecordRetrieval(t *testing.T) {
	m := newTestMetrics()

	// 测试成功检索
	m.RecordRetrieval(100*time.Millisecond, nil)
	assert.Equal(t, float64(1), m.retrievalTotal.Get())
	assert.InDelta(t, 0.1, m.retrievalDuration.Get(), 0.01)
	assert.Equal(t, float64(0), m.retrievalErrors.Get())

	// 测试失败检索
	m.RecordRetrieval(50*time.Millisecond, assert.AnError)
	assert.Equal(t, float64(2), m.retrievalTotal.Get())
	assert.Equal(t, float64(1), m.retrievalErrors.Get())
	// 失败时也记录耗时
	assert.InDelta(t, 0.15, m.retrievalDuration.Get(), 0.01)
}

func TestRecordLLMCall(t *testing.T) {
	m := newTestMetrics()

	// 测试成功的LLM调用
	m.RecordLLMCall(500*time.Millisecond, 100, 50, nil)
	assert.Equal(t, float64(1), m.llmCallsTotal.Get())
	assert.InDelta(t, 0.5, m.llmCallsDuration.Get(), 0.01)
	assert.Equal(t, float64(100), m.llmTokensPrompt.Get())
	assert.Equal(t, float64(50), m.llmTokensCompletion.Get())
	assert.Equal(t, float64(0), m.llmCallsErrors.Get())

	// 测试失败的LLM调用
	m.RecordLLMCall(200*time.Millisecond, 0, 0, assert.AnError)
	assert.Equal(t, float64(2), m.llmCallsTotal.Get())
	assert.Equal(t, float64(1), m.llmCallsErrors.Get())
	// 失败时也记录耗时 (based on implementation)
	assert.InDelta(t, 0.7, m.llmCallsDuration.Get(), 0.01)
	assert.Equal(t, float64(100), m.llmTokensPrompt.Get())
}

func TestRecordLLMRetry(t *testing.T) {
	m := newTestMetrics()

	m.RecordLLMRetry()
	assert.Equal(t, float64(1), m.llmCallsRetries.Get())

	m.RecordLLMRetry()
	assert.Equal(t, float64(2), m.llmCallsRetries.Get())
}

func TestCircuitBreakerStates(t *testing.T) {
	m := newTestMetrics()

	// 初始状态应该是Closed (0)
	assert.Equal(t, float64(0), m.circuitBreakerState.Get())

	// 测试打开熔断器
	m.RecordCircuitBreakerOpen()
	assert.Equal(t, float64(1), m.circuitBreakerState.Get())
	assert.Equal(t, float64(1), m.circuitBreakerOpens.Get())

	// 测试半开状态
	m.RecordCircuitBreakerHalfOpen()
	assert.Equal(t, float64(2), m.circuitBreakerState.Get())

	// 测试关闭熔断器
	m.RecordCircuitBreakerClosed()
	assert.Equal(t, float64(0), m.circuitBreakerState.Get())

	// 再次打开
	m.RecordCircuitBreakerOpen()
	assert.Equal(t, float64(2), m.circuitBreakerOpens.Get())
}

func TestRecordIndexing(t *testing.T) {
	m := newTestMetrics()

	// 测试成功索引
	m.RecordIndexing(5, 50, nil)
	assert.Equal(t, float64(5), m.documentsIndexed.Get())
	assert.Equal(t, float64(50), m.chunksIndexed.Get())
	assert.Equal(t, float64(0), m.indexErrors.Get())

	// 测试失败索引
	m.RecordIndexing(2, 20, assert.AnError)
	assert.Equal(t, float64(1), m.indexErrors.Get())
	// 失败时不增加计数
	assert.Equal(t, float64(5), m.documentsIndexed.Get())
	assert.Equal(t, float64(50), m.chunksIndexed.Get())
}

func TestExport(t *testing.T) {
	m := newTestMetrics()

	// Populate metrics
	m.queriesTotal.Add(100)
	m.queriesCacheHits.Add(80)
	m.queriesCacheMisses.Add(20)
	m.queriesErrors.Add(5)
	m.retrievalTotal.Add(100)
	m.retrievalDuration.Add(50.0)
	m.retrievalErrors.Add(2)
	m.llmCallsTotal.Add(50)
	m.llmCallsDuration.Add(100.0)
	m.llmCallsErrors.Add(3)
	m.llmCallsRetries.Add(5)
	m.llmTokensPrompt.Add(10000)
	m.llmTokensCompletion.Add(5000)
	m.circuitBreakerOpens.Add(2)
	m.circuitBreakerState.Set(0)
	m.documentsIndexed.Add(10)
	m.chunksIndexed.Add(100)
	m.indexErrors.Add(1)

	output := metrics.Export()

	// 验证输出包含关键指标
	assert.Contains(t, output, "rag_queries_total 100")
	assert.Contains(t, output, "rag_llm_calls_total 50")
	assert.Contains(t, output, "rag_circuit_breaker_state 0")
	assert.Contains(t, output, "rag_documents_indexed_total 10")

	// 验证包含HELP和TYPE注释
	assert.Contains(t, output, "# HELP rag_queries_total")
	assert.Contains(t, output, "# TYPE rag_queries_total counter")

	// 验证运行时间大于0
	assert.Contains(t, output, "rag_uptime_seconds")
}

func TestStats(t *testing.T) {
	m := newTestMetrics()

	// Populate metrics
	m.queriesTotal.Add(100)
	m.queriesCacheHits.Add(75)
	m.queriesCacheMisses.Add(25)
	m.queriesErrors.Add(5)
	m.retrievalTotal.Add(100)
	m.retrievalDuration.Add(50.0)
	m.retrievalErrors.Add(2)
	m.llmCallsTotal.Add(50)
	m.llmCallsDuration.Add(100.0)
	m.llmCallsErrors.Add(3)
	m.llmCallsRetries.Add(5)
	m.llmTokensPrompt.Add(10000)
	m.llmTokensCompletion.Add(5000)
	m.circuitBreakerOpens.Add(2)
	m.circuitBreakerState.Set(1) // Open
	m.documentsIndexed.Add(10)
	m.chunksIndexed.Add(100)
	m.indexErrors.Add(1)

	stats := m.Stats()

	// 验证查询统计
	queries := stats["queries"].(map[string]interface{})
	assert.Equal(t, uint64(100), queries["total"])
	assert.Equal(t, uint64(75), queries["cache_hits"])
	assert.Equal(t, uint64(25), queries["cache_misses"])
	assert.InDelta(t, 0.75, queries["cache_hit_rate"], 0.01)

	// 验证检索统计
	retrieval := stats["retrieval"].(map[string]interface{})
	assert.Equal(t, uint64(100), retrieval["total"])
	assert.InDelta(t, 50.0, retrieval["total_duration_secs"], 0.01)
	assert.InDelta(t, 0.5, retrieval["avg_duration_secs"], 0.01)

	// 验证LLM统计
	llm := stats["llm"].(map[string]interface{})
	assert.Equal(t, uint64(50), llm["calls_total"])
	assert.InDelta(t, 2.0, llm["avg_duration_secs"], 0.01)
	assert.Equal(t, uint64(10000), llm["tokens_prompt"])

	// 验证熔断器统计
	cb := stats["circuit_breaker"].(map[string]interface{})
	assert.Equal(t, "open", cb["state"])
	assert.Equal(t, uint64(2), cb["opens"])

	// 验证索引统计
	indexing := stats["indexing"].(map[string]interface{})
	assert.Equal(t, uint64(10), indexing["documents_indexed"])
	assert.Equal(t, uint64(100), indexing["chunks_indexed"])

	// 验证运行时间
	uptime := stats["uptime_seconds"].(float64)
	assert.Greater(t, uptime, 0.0)
}

func TestReset(t *testing.T) {
	m := newTestMetrics()
	// Populate some data
	m.queriesTotal.Add(100)
	oldStartTime := m.startTime

	m.Reset()
	// MUST get the new instance after reset
	mNew := GetRAGMetrics()

	// 验证所有计数器都已重置
	assert.Equal(t, float64(0), mNew.queriesTotal.Get())
	assert.Equal(t, float64(0), mNew.queriesCacheHits.Get())
	assert.Equal(t, float64(0), mNew.circuitBreakerState.Get())

	// 验证startTime已更新
	assert.True(t, mNew.startTime.After(oldStartTime) || mNew.startTime.Equal(oldStartTime))
}

func TestConcurrentAccess(t *testing.T) {
	m := newTestMetrics()

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 100

	// 并发记录查询
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				m.RecordQuery(j%2 == 0, nil)
			}
		}()
	}
	wg.Wait()

	// 验证计数正确
	expected := float64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expected, m.queriesTotal.Get())

	// 并发记录LLM调用
	m.Reset()
	m = GetRAGMetrics()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				m.RecordLLMCall(10*time.Millisecond, 10, 5, nil)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, expected, m.llmCallsTotal.Get())
	assert.Equal(t, expected*10, m.llmTokensPrompt.Get())
	assert.Equal(t, expected*5, m.llmTokensCompletion.Get())
}

func TestCacheHitRateCalculation(t *testing.T) {
	testCases := []struct {
		name               string
		cacheHits          float64
		cacheMisses        float64
		expectedHitRate    float64
	}{
		{
			name:            "完全命中",
			cacheHits:       100,
			cacheMisses:     0,
			expectedHitRate: 1.0,
		},
		{
			name:            "完全未命中",
			cacheHits:       0,
			cacheMisses:     100,
			expectedHitRate: 0.0,
		},
		{
			name:            "50%命中",
			cacheHits:       50,
			cacheMisses:     50,
			expectedHitRate: 0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestMetrics()
			m.queriesCacheHits.Add(tc.cacheHits)
			m.queriesCacheMisses.Add(tc.cacheMisses)

			// 测试Stats方法
			stats := m.Stats()
			queries := stats["queries"].(map[string]interface{})
			assert.InDelta(t, tc.expectedHitRate, queries["cache_hit_rate"], 0.0001)
		})
	}
}

func TestAverageDurationCalculation(t *testing.T) {
	m := newTestMetrics()

	// Simulate data
	// 10 retrievals, total 50s
	m.retrievalTotal.Add(10)
	m.retrievalDuration.Add(50.0)

	// 5 llm calls, total 25s
	m.llmCallsTotal.Add(5)
	m.llmCallsDuration.Add(25.0)

	stats := m.Stats()

	// 验证平均检索耗时
	retrieval := stats["retrieval"].(map[string]interface{})
	assert.InDelta(t, 5.0, retrieval["avg_duration_secs"], 0.01) // 50/10 = 5

	// 验证平均LLM耗时
	llm := stats["llm"].(map[string]interface{})
	assert.InDelta(t, 5.0, llm["avg_duration_secs"], 0.01) // 25/5 = 5
}
