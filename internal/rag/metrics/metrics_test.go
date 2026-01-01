package metrics

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetRAGMetrics(t *testing.T) {
	// 获取全局单例
	m1 := GetRAGMetrics()
	m2 := GetRAGMetrics()

	// 应该返回同一个实例
	assert.Equal(t, m1, m2, "应该返回同一个单例实例")
}

func TestRecordQuery(t *testing.T) {
	m := &RAGMetrics{}

	// 测试成功查询（缓存命中）
	m.RecordQuery(true, nil)
	assert.Equal(t, uint64(1), m.queriesTotal)
	assert.Equal(t, uint64(1), m.queriesCacheHits)
	assert.Equal(t, uint64(0), m.queriesCacheMisses)
	assert.Equal(t, uint64(0), m.queriesErrors)

	// 测试成功查询（缓存未命中）
	m.RecordQuery(false, nil)
	assert.Equal(t, uint64(2), m.queriesTotal)
	assert.Equal(t, uint64(1), m.queriesCacheHits)
	assert.Equal(t, uint64(1), m.queriesCacheMisses)
	assert.Equal(t, uint64(0), m.queriesErrors)

	// 测试失败查询
	m.RecordQuery(false, assert.AnError)
	assert.Equal(t, uint64(3), m.queriesTotal)
	assert.Equal(t, uint64(1), m.queriesErrors)
}

func TestRecordRetrieval(t *testing.T) {
	m := &RAGMetrics{}

	// 测试成功检索
	m.RecordRetrieval(100*time.Millisecond, nil)
	assert.Equal(t, uint64(1), m.retrievalTotal)
	assert.InDelta(t, 0.1, m.retrievalDuration, 0.01)
	assert.Equal(t, uint64(0), m.retrievalErrors)

	// 测试失败检索
	m.RecordRetrieval(50*time.Millisecond, assert.AnError)
	assert.Equal(t, uint64(2), m.retrievalTotal)
	assert.Equal(t, uint64(1), m.retrievalErrors)
	// 失败时不记录耗时
	assert.InDelta(t, 0.1, m.retrievalDuration, 0.01)
}

func TestRecordLLMCall(t *testing.T) {
	m := &RAGMetrics{}

	// 测试成功的LLM调用
	m.RecordLLMCall(500*time.Millisecond, 100, 50, nil)
	assert.Equal(t, uint64(1), m.llmCallsTotal)
	assert.InDelta(t, 0.5, m.llmCallsDuration, 0.01)
	assert.Equal(t, uint64(100), m.llmTokensPrompt)
	assert.Equal(t, uint64(50), m.llmTokensCompletion)
	assert.Equal(t, uint64(0), m.llmCallsErrors)

	// 测试失败的LLM调用
	m.RecordLLMCall(200*time.Millisecond, 0, 0, assert.AnError)
	assert.Equal(t, uint64(2), m.llmCallsTotal)
	assert.Equal(t, uint64(1), m.llmCallsErrors)
	// 失败时不记录耗时和tokens
	assert.InDelta(t, 0.5, m.llmCallsDuration, 0.01)
	assert.Equal(t, uint64(100), m.llmTokensPrompt)
}

func TestRecordLLMRetry(t *testing.T) {
	m := &RAGMetrics{}

	m.RecordLLMRetry()
	assert.Equal(t, uint64(1), m.llmCallsRetries)

	m.RecordLLMRetry()
	assert.Equal(t, uint64(2), m.llmCallsRetries)
}

func TestCircuitBreakerStates(t *testing.T) {
	m := &RAGMetrics{}

	// 初始状态应该是Closed (0)
	assert.Equal(t, int32(0), m.circuitBreakerState)

	// 测试打开熔断器
	m.RecordCircuitBreakerOpen()
	assert.Equal(t, int32(1), m.circuitBreakerState)
	assert.Equal(t, uint64(1), m.circuitBreakerOpens)

	// 测试半开状态
	m.RecordCircuitBreakerHalfOpen()
	assert.Equal(t, int32(2), m.circuitBreakerState)

	// 测试关闭熔断器
	m.RecordCircuitBreakerClosed()
	assert.Equal(t, int32(0), m.circuitBreakerState)

	// 再次打开
	m.RecordCircuitBreakerOpen()
	assert.Equal(t, uint64(2), m.circuitBreakerOpens)
}

func TestRecordIndexing(t *testing.T) {
	m := &RAGMetrics{}

	// 测试成功索引
	m.RecordIndexing(5, 50, nil)
	assert.Equal(t, uint64(5), m.documentsIndexed)
	assert.Equal(t, uint64(50), m.chunksIndexed)
	assert.Equal(t, uint64(0), m.indexErrors)

	// 测试失败索引
	m.RecordIndexing(2, 20, assert.AnError)
	assert.Equal(t, uint64(1), m.indexErrors)
	// 失败时不增加计数
	assert.Equal(t, uint64(5), m.documentsIndexed)
	assert.Equal(t, uint64(50), m.chunksIndexed)
}

func TestExport(t *testing.T) {
	m := &RAGMetrics{
		queriesTotal:       100,
		queriesCacheHits:   80,
		queriesCacheMisses: 20,
		queriesErrors:      5,
		retrievalTotal:     100,
		retrievalDuration:  50.0,
		retrievalErrors:    2,
		llmCallsTotal:      50,
		llmCallsDuration:   100.0,
		llmCallsErrors:     3,
		llmCallsRetries:    5,
		llmTokensPrompt:    10000,
		llmTokensCompletion: 5000,
		circuitBreakerOpens: 2,
		circuitBreakerState: 0,
		documentsIndexed:    10,
		chunksIndexed:       100,
		indexErrors:         1,
		startTime:           time.Now().Add(-1 * time.Hour),
	}

	output := m.Export("sentinel_x", "rag")

	// 验证输出包含关键指标
	assert.Contains(t, output, "sentinel_x_rag_queries_total 100")
	assert.Contains(t, output, "sentinel_x_rag_cache_hit_rate 0.8000")
	assert.Contains(t, output, "sentinel_x_rag_llm_calls_total 50")
	assert.Contains(t, output, "sentinel_x_rag_circuit_breaker_state 0")
	assert.Contains(t, output, "sentinel_x_rag_documents_indexed_total 10")

	// 验证包含HELP和TYPE注释
	assert.Contains(t, output, "# HELP sentinel_x_rag_queries_total")
	assert.Contains(t, output, "# TYPE sentinel_x_rag_queries_total counter")

	// 验证运行时间大于0
	assert.Contains(t, output, "sentinel_x_rag_uptime_seconds")
}

func TestStats(t *testing.T) {
	m := &RAGMetrics{
		queriesTotal:       100,
		queriesCacheHits:   75,
		queriesCacheMisses: 25,
		queriesErrors:      5,
		retrievalTotal:     100,
		retrievalDuration:  50.0,
		retrievalErrors:    2,
		llmCallsTotal:      50,
		llmCallsDuration:   100.0,
		llmCallsErrors:     3,
		llmCallsRetries:    5,
		llmTokensPrompt:    10000,
		llmTokensCompletion: 5000,
		circuitBreakerOpens: 2,
		circuitBreakerState: 1, // Open
		documentsIndexed:    10,
		chunksIndexed:       100,
		indexErrors:         1,
		startTime:           time.Now().Add(-1 * time.Hour),
	}

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
	assert.Greater(t, uptime, 3600.0)
}

func TestReset(t *testing.T) {
	m := &RAGMetrics{
		queriesTotal:        100,
		queriesCacheHits:    80,
		queriesCacheMisses:  20,
		queriesErrors:       5,
		retrievalTotal:      100,
		retrievalDuration:   50.0,
		retrievalErrors:     2,
		llmCallsTotal:       50,
		llmCallsDuration:    100.0,
		llmCallsErrors:      3,
		llmCallsRetries:     5,
		llmTokensPrompt:     10000,
		llmTokensCompletion: 5000,
		circuitBreakerOpens: 2,
		circuitBreakerState: 1,
		documentsIndexed:    10,
		chunksIndexed:       100,
		indexErrors:         1,
		startTime:           time.Now().Add(-1 * time.Hour),
	}

	oldStartTime := m.startTime
	m.Reset()

	// 验证所有计数器都已重置
	assert.Equal(t, uint64(0), m.queriesTotal)
	assert.Equal(t, uint64(0), m.queriesCacheHits)
	assert.Equal(t, uint64(0), m.queriesCacheMisses)
	assert.Equal(t, uint64(0), m.queriesErrors)
	assert.Equal(t, uint64(0), m.retrievalTotal)
	assert.Equal(t, float64(0), m.retrievalDuration)
	assert.Equal(t, uint64(0), m.retrievalErrors)
	assert.Equal(t, uint64(0), m.llmCallsTotal)
	assert.Equal(t, float64(0), m.llmCallsDuration)
	assert.Equal(t, uint64(0), m.llmCallsErrors)
	assert.Equal(t, uint64(0), m.llmCallsRetries)
	assert.Equal(t, uint64(0), m.llmTokensPrompt)
	assert.Equal(t, uint64(0), m.llmTokensCompletion)
	assert.Equal(t, uint64(0), m.circuitBreakerOpens)
	assert.Equal(t, int32(0), m.circuitBreakerState)
	assert.Equal(t, uint64(0), m.documentsIndexed)
	assert.Equal(t, uint64(0), m.chunksIndexed)
	assert.Equal(t, uint64(0), m.indexErrors)

	// 验证startTime已更新
	assert.True(t, m.startTime.After(oldStartTime))
}

func TestConcurrentAccess(t *testing.T) {
	m := GetRAGMetrics()
	m.Reset() // 重置以获得干净的状态

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
	expected := uint64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expected, m.queriesTotal)

	// 并发记录LLM调用
	m.Reset()
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

	assert.Equal(t, expected, m.llmCallsTotal)
	assert.Equal(t, expected*10, m.llmTokensPrompt)
	assert.Equal(t, expected*5, m.llmTokensCompletion)
}

func TestExportWithEmptySubsystem(t *testing.T) {
	m := &RAGMetrics{
		queriesTotal: 50,
	}

	output := m.Export("sentinel_x", "")

	// 当subsystem为空时，应该只使用namespace
	assert.Contains(t, output, "sentinel_x_queries_total 50")
	assert.NotContains(t, output, "sentinel_x__queries_total") // 不应该有双下划线
}

func TestCacheHitRateCalculation(t *testing.T) {
	testCases := []struct {
		name               string
		cacheHits          uint64
		cacheMisses        uint64
		expectedHitRate    float64
		expectedHitRateStr string
	}{
		{
			name:               "完全命中",
			cacheHits:          100,
			cacheMisses:        0,
			expectedHitRate:    1.0,
			expectedHitRateStr: "1.0000",
		},
		{
			name:               "完全未命中",
			cacheHits:          0,
			cacheMisses:        100,
			expectedHitRate:    0.0,
			expectedHitRateStr: "0.0000",
		},
		{
			name:               "50%命中",
			cacheHits:          50,
			cacheMisses:        50,
			expectedHitRate:    0.5,
			expectedHitRateStr: "0.5000",
		},
		{
			name:               "无数据",
			cacheHits:          0,
			cacheMisses:        0,
			expectedHitRate:    0.0,
			expectedHitRateStr: "0.0000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &RAGMetrics{
				queriesCacheHits:   tc.cacheHits,
				queriesCacheMisses: tc.cacheMisses,
			}

			// 测试Stats方法
			stats := m.Stats()
			queries := stats["queries"].(map[string]interface{})
			assert.InDelta(t, tc.expectedHitRate, queries["cache_hit_rate"], 0.0001)

			// 测试Export方法
			output := m.Export("test", "rag")
			assert.Contains(t, output, "test_rag_cache_hit_rate "+tc.expectedHitRateStr)
		})
	}
}

func TestAverageDurationCalculation(t *testing.T) {
	m := &RAGMetrics{
		retrievalTotal:    10,
		retrievalDuration: 50.0, // 总耗时50秒
		llmCallsTotal:     5,
		llmCallsDuration:  25.0, // 总耗时25秒
	}

	stats := m.Stats()

	// 验证平均检索耗时
	retrieval := stats["retrieval"].(map[string]interface{})
	assert.InDelta(t, 5.0, retrieval["avg_duration_secs"], 0.01) // 50/10 = 5

	// 验证平均LLM耗时
	llm := stats["llm"].(map[string]interface{})
	assert.InDelta(t, 5.0, llm["avg_duration_secs"], 0.01) // 25/5 = 5
}

func TestCircuitBreakerStateMapping(t *testing.T) {
	testCases := []struct {
		state         int32
		expectedState string
	}{
		{0, "closed"},
		{1, "open"},
		{2, "half-open"},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedState, func(t *testing.T) {
			m := &RAGMetrics{
				circuitBreakerState: tc.state,
			}

			stats := m.Stats()
			cb := stats["circuit_breaker"].(map[string]interface{})
			assert.Equal(t, tc.expectedState, cb["state"])
		})
	}
}

func TestExportPrometheusFormat(t *testing.T) {
	m := &RAGMetrics{
		queriesTotal: 100,
		startTime:    time.Now(),
	}

	output := m.Export("test", "rag")

	// 验证Prometheus格式规范
	lines := strings.Split(output, "\n")

	var helpLines, typeLines, metricLines int
	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			helpLines++
		} else if strings.HasPrefix(line, "# TYPE") {
			typeLines++
		} else if len(line) > 0 && !strings.HasPrefix(line, "#") {
			metricLines++
		}
	}

	// 应该有相同数量的HELP和TYPE行
	assert.Equal(t, helpLines, typeLines, "HELP和TYPE行数应该相等")

	// 指标行数应该等于HELP/TYPE行数
	assert.Equal(t, helpLines, metricLines, "指标行数应该等于HELP/TYPE行数")
}
