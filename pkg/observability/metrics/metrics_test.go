package metrics

import (
	"strings"
	"testing"
)

func TestCounter(t *testing.T) {
	c := NewCounter("test_counter", "Test counter")

	if c.Name() != "test_counter" {
		t.Errorf("expected name test_counter, got %s", c.Name())
	}
	if c.Type() != TypeCounter {
		t.Errorf("expected type counter, got %s", c.Type())
	}

	c.Inc()
	if c.Get() != 1 {
		t.Errorf("expected value 1, got %.0f", c.Get())
	}

	c.Add(5)
	if c.Get() != 6 {
		t.Errorf("expected value 6, got %.0f", c.Get())
	}
}

func TestGauge(t *testing.T) {
	g := NewGauge("test_gauge", "Test gauge")

	g.Set(10)
	if g.Get() != 10 {
		t.Errorf("expected value 10, got %.0f", g.Get())
	}

	g.Inc()
	if g.Get() != 11 {
		t.Errorf("expected value 11, got %.0f", g.Get())
	}

	g.Dec()
	if g.Get() != 10 {
		t.Errorf("expected value 10, got %.0f", g.Get())
	}

	g.Sub(5)
	if g.Get() != 5 {
		t.Errorf("expected value 5, got %.0f", g.Get())
	}
}

func TestHistogram(t *testing.T) {
	h := NewHistogram("test_histogram", "Test histogram", []float64{1, 5, 10})

	h.Observe(2)
	h.Observe(7)
	h.Observe(12)

	desc := h.Describe()
	if !strings.Contains(desc, "test_histogram_bucket{le=\"1\"} 0") {
		t.Errorf("expected bucket le=1 to be 0")
	}
	if !strings.Contains(desc, "test_histogram_bucket{le=\"5\"} 1") {
		t.Errorf("expected bucket le=5 to be 1")
	}
	if !strings.Contains(desc, "test_histogram_bucket{le=\"10\"} 2") {
		t.Errorf("expected bucket le=10 to be 2")
	}
	if !strings.Contains(desc, "test_histogram_bucket{le=\"+Inf\"} 3") {
		t.Errorf("expected bucket le=+Inf to be 3")
	}
	if !strings.Contains(desc, "test_histogram_count 3") {
		t.Errorf("expected count 3")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	c := NewCounter("test_counter", "help")
	r.Register(c)

	c.Inc()

	out := r.Export()
	if !strings.Contains(out, "# HELP test_counter help") {
		t.Errorf("expected help text in output")
	}
	if !strings.Contains(out, "test_counter 1") {
		t.Errorf("expected value in output")
	}

	r.Reset()
	if r.Export() != "" {
		t.Errorf("expected empty output after reset")
	}
}

func TestVectors(t *testing.T) {
	cv := NewCounterVec("http_requests", "HTTP Requests")

	cv.With(map[string]string{"method": "GET"}).Inc()
	cv.With(map[string]string{"method": "POST"}).Add(2)

	out := cv.Describe()
	if !strings.Contains(out, "http_requests{method=\"GET\"} 1") {
		t.Errorf("expected GET count 1")
	}
	if !strings.Contains(out, "http_requests{method=\"POST\"} 2") {
		t.Errorf("expected POST count 2")
	}
}
