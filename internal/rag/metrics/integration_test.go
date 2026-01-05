package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/observability/metrics"
)

func TestRAGMetricsIntegration(t *testing.T) {
	// Reset registry before test
	metrics.DefaultRegistry.Reset()

	// Initialize metrics
	m := GetRAGMetrics()

	// Record some metrics
	m.RecordQuery(true, nil)
	m.RecordQuery(false, nil)
	m.RecordRetrieval(100*time.Millisecond, nil)
	m.RecordLLMCall(500*time.Millisecond, 10, 20, nil)

	// Check Stats (legacy API)
	stats := m.Stats()
	if stats["queries"].(map[string]interface{})["total"].(uint64) != 2 {
		t.Errorf("expected 2 queries, got %v", stats["queries"].(map[string]interface{})["total"])
	}

	// Check Registry Export
	out := metrics.Export()

	// Check for metric presence in Prometheus output
	if !strings.Contains(out, "rag_queries_total 2") {
		t.Errorf("expected rag_queries_total 2 in output")
	}
	if !strings.Contains(out, "rag_queries_cache_hits_total 1") {
		t.Errorf("expected rag_queries_cache_hits_total 1 in output")
	}
	if !strings.Contains(out, "rag_retrieval_duration_seconds_total 0.100000") {
		t.Errorf("expected rag_retrieval_duration_seconds_total 0.1 in output")
	}
}
