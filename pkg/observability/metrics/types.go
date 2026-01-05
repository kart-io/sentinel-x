// Package metrics provides a unified metrics collection system for the project.
package metrics

// MetricType represents the type of metric.
type MetricType string

// Metric type constants define the supported metric types.
const (
	TypeCounter   MetricType = "counter"
	TypeGauge     MetricType = "gauge"
	TypeHistogram MetricType = "histogram"
	TypeSummary   MetricType = "summary"
)

// Metric is the base interface for all metrics.
type Metric interface {
	Name() string
	Help() string
	Type() MetricType
	// Describe returns the metric description in Prometheus format.
	Describe() string
}

// Counter is a cumulative metric that represents a single monotonically increasing counter.
type Counter interface {
	Metric
	Inc()
	Add(float64)
	Get() float64
}

// Gauge is a metric that represents a single numerical value that can arbitrarily go up and down.
type Gauge interface {
	Metric
	Set(float64)
	Inc()
	Dec()
	Add(float64)
	Sub(float64)
	Get() float64
}

// Histogram samples observations (usually things like request durations or response sizes)
// and counts them in configurable buckets.
type Histogram interface {
	Metric
	Observe(float64)
}

// Vector represents a collection of metrics with the same name but different label values.
type Vector interface {
	Metric
	// WithLabels returns the metric with the given label values.
	// This generic method allows treating vectors uniformly.
	WithLabels(labels map[string]string) Metric
}

// CounterVec is a vector of counters.
type CounterVec interface {
	Vector
	// With returns the counter with the given label values.
	With(labels map[string]string) Counter
}

// GaugeVec is a vector of gauges.
type GaugeVec interface {
	Vector
	// With returns the gauge with the given label values.
	With(labels map[string]string) Gauge
}

// HistogramVec is a vector of histograms.
type HistogramVec interface {
	Vector
	// With returns the histogram with the given label values.
	With(labels map[string]string) Histogram
}
