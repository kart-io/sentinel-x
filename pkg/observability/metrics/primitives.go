package metrics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// --- Base Metric Implementation ---

type baseMetric struct {
	name string
	help string
	typ  MetricType
}

func (m *baseMetric) Name() string {
	return m.name
}

func (m *baseMetric) Help() string {
	return m.help
}

func (m *baseMetric) Type() MetricType {
	return m.typ
}

// --- Counter Implementation ---

type counter struct {
	baseMetric
	val uint64
}

// NewCounter creates a new Counter metric with the given name and help text.
func NewCounter(name, help string) Counter {
	return &counter{
		baseMetric: baseMetric{
			name: name,
			help: help,
			typ:  TypeCounter,
		},
	}
}

func (c *counter) Inc() {
	c.Add(1)
}

func (c *counter) Add(v float64) {
	if v < 0 {
		return
	}
	for {
		oldBits := atomic.LoadUint64(&c.val)
		newVal := math.Float64frombits(oldBits) + v
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapUint64(&c.val, oldBits, newBits) {
			return
		}
	}
}

func (c *counter) Get() float64 {
	return math.Float64frombits(atomic.LoadUint64(&c.val))
}

func (c *counter) Describe() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", c.name, c.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", c.name, c.typ))
	sb.WriteString(fmt.Sprintf("%s %.6f\n", c.name, c.Get()))
	return sb.String()
}

// --- Gauge Implementation ---

type gauge struct {
	baseMetric
	val uint64 // Using uint64 bits for float64 atomic operations
}

// NewGauge creates a new Gauge metric with the given name and help text.
func NewGauge(name, help string) Gauge {
	return &gauge{
		baseMetric: baseMetric{
			name: name,
			help: help,
			typ:  TypeGauge,
		},
	}
}

func (g *gauge) Set(v float64) {
	atomic.StoreUint64(&g.val, math.Float64bits(v))
}

func (g *gauge) Inc() {
	g.Add(1)
}

func (g *gauge) Dec() {
	g.Sub(1)
}

func (g *gauge) Add(v float64) {
	for {
		oldBits := atomic.LoadUint64(&g.val)
		newVal := math.Float64frombits(oldBits) + v
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapUint64(&g.val, oldBits, newBits) {
			return
		}
	}
}

func (g *gauge) Sub(v float64) {
	g.Add(-v)
}

func (g *gauge) Get() float64 {
	return math.Float64frombits(atomic.LoadUint64(&g.val))
}

func (g *gauge) Describe() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", g.name, g.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", g.name, g.typ))
	sb.WriteString(fmt.Sprintf("%s %.6f\n", g.name, g.Get()))
	return sb.String()
}

// --- Histogram Implementation ---

type histogram struct {
	baseMetric
	buckets []float64
	counts  []uint64
	sum     uint64 // Using uint64 bits for float64
	count   uint64
	mu      sync.RWMutex
}

// NewHistogram creates a new Histogram metric with the given name, help text, and bucket boundaries.
func NewHistogram(name, help string, buckets []float64) Histogram {
	if len(buckets) == 0 {
		buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	}
	sort.Float64s(buckets)
	return &histogram{
		baseMetric: baseMetric{
			name: name,
			help: help,
			typ:  TypeHistogram,
		},
		buckets: buckets,
		counts:  make([]uint64, len(buckets)),
	}
}

func (h *histogram) Observe(v float64) {
	// Update total count
	atomic.AddUint64(&h.count, 1)

	// Update sum atomically
	for {
		oldBits := atomic.LoadUint64(&h.sum)
		newVal := math.Float64frombits(oldBits) + v
		newBits := math.Float64bits(newVal)
		if atomic.CompareAndSwapUint64(&h.sum, oldBits, newBits) {
			break
		}
	}

	// Update buckets
	// Note: This is not strictly atomic across buckets but sufficient for metrics
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, bucket := range h.buckets {
		if v <= bucket {
			h.counts[i]++
		}
	}
}

func (h *histogram) Describe() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", h.name, h.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", h.name, h.typ))

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Buckets
	for i, bucket := range h.buckets {
		sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%.6g\"} %d\n", h.name, bucket, h.counts[i]))
	}
	sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", h.name, atomic.LoadUint64(&h.count)))

	// Sum
	sum := math.Float64frombits(atomic.LoadUint64(&h.sum))
	sb.WriteString(fmt.Sprintf("%s_sum %.6f\n", h.name, sum))

	// Count
	sb.WriteString(fmt.Sprintf("%s_count %d\n", h.name, atomic.LoadUint64(&h.count)))

	return sb.String()
}

// --- Vector Implementation Helpers ---

func formatLabels(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	var pairs []string
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=\"%s\"", k, v))
	}
	sort.Strings(pairs)
	return fmt.Sprintf("%s{%s}", name, strings.Join(pairs, ","))
}

// --- CounterVec Implementation ---

type counterVec struct {
	baseMetric
	metrics sync.Map // map[string]*counter
}

// NewCounterVec creates a new CounterVec metric with the given name and help text.
func NewCounterVec(name, help string) CounterVec {
	return &counterVec{
		baseMetric: baseMetric{
			name: name,
			help: help,
			typ:  TypeCounter,
		},
	}
}

func (v *counterVec) WithLabels(labels map[string]string) Metric {
	return v.With(labels)
}

func (v *counterVec) With(labels map[string]string) Counter {
	key := formatLabels(v.name, labels)
	// We store the full formatted string as key, but the counter itself doesn't need to know about labels for value tracking
	// However, for Export, we need to know the labels.
	// For simplicity in this implementation, we'll store counters and reconstruct the export output.
	// A more complex implementation would store the labels in the counter.

	// Check if exists
	if val, ok := v.metrics.Load(key); ok {
		return val.(*counter)
	}

	// Create new
	c := &counter{
		baseMetric: baseMetric{
			name: key, // Store the full name with labels as the name for the individual counter
			help: v.help,
			typ:  TypeCounter,
		},
	}
	actual, _ := v.metrics.LoadOrStore(key, c)
	return actual.(*counter)
}

func (v *counterVec) Describe() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", v.name, v.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", v.name, v.typ))

	var keys []string
	v.metrics.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		if val, ok := v.metrics.Load(key); ok {
			c := val.(*counter)
			// The key is already name{labels}, so we just output name{labels} value
			sb.WriteString(fmt.Sprintf("%s %.6f\n", key, c.Get()))
		}
	}
	return sb.String()
}

// --- GaugeVec Implementation ---

type gaugeVec struct {
	baseMetric
	metrics sync.Map // map[string]*gauge
}

// NewGaugeVec creates a new GaugeVec metric with the given name and help text.
func NewGaugeVec(name, help string) GaugeVec {
	return &gaugeVec{
		baseMetric: baseMetric{
			name: name,
			help: help,
			typ:  TypeGauge,
		},
	}
}

func (v *gaugeVec) WithLabels(labels map[string]string) Metric {
	return v.With(labels)
}

func (v *gaugeVec) With(labels map[string]string) Gauge {
	key := formatLabels(v.name, labels)

	if val, ok := v.metrics.Load(key); ok {
		return val.(*gauge)
	}

	g := &gauge{
		baseMetric: baseMetric{
			name: key,
			help: v.help,
			typ:  TypeGauge,
		},
	}
	actual, _ := v.metrics.LoadOrStore(key, g)
	return actual.(*gauge)
}

func (v *gaugeVec) Describe() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", v.name, v.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", v.name, v.typ))

	var keys []string
	v.metrics.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		if val, ok := v.metrics.Load(key); ok {
			g := val.(*gauge)
			sb.WriteString(fmt.Sprintf("%s %.6f\n", key, g.Get()))
		}
	}
	return sb.String()
}

// --- HistogramVec Implementation ---

type histogramVec struct {
	baseMetric
	buckets []float64
	metrics sync.Map // map[string]*histogram
}

// NewHistogramVec creates a new HistogramVec metric with the given name, help text, and bucket boundaries.
func NewHistogramVec(name, help string, buckets []float64) HistogramVec {
	return &histogramVec{
		baseMetric: baseMetric{
			name: name,
			help: help,
			typ:  TypeHistogram,
		},
		buckets: buckets,
	}
}

func (v *histogramVec) WithLabels(labels map[string]string) Metric {
	return v.With(labels)
}

func (v *histogramVec) With(labels map[string]string) Histogram {
	key := formatLabels("", labels) // only get {labels} part
	fullKey := v.name + key

	if val, ok := v.metrics.Load(fullKey); ok {
		return val.(*histogram)
	}

	h := NewHistogram(fullKey, v.help, v.buckets).(*histogram)

	actual, _ := v.metrics.LoadOrStore(fullKey, h)
	return actual.(*histogram)
}

func (v *histogramVec) Describe() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", v.name, v.help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", v.name, v.typ))

	var keys []string
	v.metrics.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		if val, ok := v.metrics.Load(key); ok {
			h := val.(*histogram)
			h.mu.RLock()

			// Parse labels from key: name{labels} -> {labels}
			// This is a bit hacky string manipulation but works for this simplified implementation
			labelPart := ""
			if idx := strings.Index(key, "{"); idx != -1 {
				labelPart = key[idx:]
			}

			// Buckets
			for i, bucket := range h.buckets {
				// We need to merge labels: {le="0.1", method="GET"}
				bucketLabel := fmt.Sprintf("le=\"%.6g\"", bucket)
				if labelPart != "" {
					bucketLabel = bucketLabel + "," + labelPart[1:len(labelPart)-1]
				}
				sb.WriteString(fmt.Sprintf("%s_bucket{%s} %d\n", v.name, bucketLabel, h.counts[i]))
			}
			// Inf bucket
			infLabel := "le=\"+Inf\""
			if labelPart != "" {
				infLabel = infLabel + "," + labelPart[1:len(labelPart)-1]
			}
			sb.WriteString(fmt.Sprintf("%s_bucket{%s} %d\n", v.name, infLabel, atomic.LoadUint64(&h.count)))

			// Sum
			sum := math.Float64frombits(atomic.LoadUint64(&h.sum))
			sumLabels := labelPart
			sb.WriteString(fmt.Sprintf("%s_sum%s %.6f\n", v.name, sumLabels, sum))

			// Count
			countLabels := labelPart
			sb.WriteString(fmt.Sprintf("%s_count%s %d\n", v.name, countLabels, atomic.LoadUint64(&h.count)))

			h.mu.RUnlock()
		}
	}
	return sb.String()
}
