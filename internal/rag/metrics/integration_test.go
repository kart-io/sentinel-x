package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
)

func TestRAGMetricsIntegration(t *testing.T) {
	// Initialize metrics first
	m := GetRAGMetrics()

	// Reset all metrics for clean test
	m.Reset()

	// 重新获取metrics实例，因为Reset()会创建新的实例
	m = GetRAGMetrics()

	// Record some metrics
	m.RecordQuery(true, nil)
	m.RecordQuery(false, nil)
	m.RecordRetrieval(100*time.Millisecond, nil)
	m.RecordLLMCall(500*time.Millisecond, 10, 20, nil)

	// Check Stats (legacy API)
	stats := m.Stats()
	if stats["queries"].(map[string]any)["total"].(uint64) != 2 {
		t.Errorf("expected 2 queries, got %v", stats["queries"].(map[string]any)["total"])
	}

	// Check Registry Export
	out := metrics.Export()

	// Check for metric presence in Prometheus output
	// 注意：Prometheus 格式可能包含标签和其他信息
	if !strings.Contains(out, "rag_queries_total") || !strings.Contains(out, " 2") {
		t.Errorf("expected rag_queries_total with value 2 in output")
	}
	if !strings.Contains(out, "rag_queries_cache_hits_total") || !strings.Contains(out, " 1") {
		t.Errorf("expected rag_queries_cache_hits_total with value 1 in output")
	}
	if !strings.Contains(out, "rag_retrieval_duration_seconds_total") || !strings.Contains(out, "0.1") {
		t.Errorf("expected rag_retrieval_duration_seconds_total with value ~0.1 in output")
	}
}
