package metrics

import (
	"sort"
	"strings"
	"sync"
)

// Registry manages a collection of metrics.
type Registry struct {
	metrics sync.Map // map[string]Metric
}

// NewRegistry creates a new metrics registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// DefaultRegistry is the default global registry.
var DefaultRegistry = NewRegistry()

// Register registers a metric with the registry.
func (r *Registry) Register(m Metric) {
	r.metrics.Store(m.Name(), m)
}

// Register registers a metric with the default registry.
func Register(m Metric) {
	DefaultRegistry.Register(m)
}

// Export returns all metrics in Prometheus text format.
func (r *Registry) Export() string {
	var sb strings.Builder
	var names []string

	r.metrics.Range(func(key, _ interface{}) bool {
		names = append(names, key.(string))
		return true
	})

	sort.Strings(names)

	for _, name := range names {
		if val, ok := r.metrics.Load(name); ok {
			if m, ok := val.(Metric); ok {
				sb.WriteString(m.Describe())
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// Export returns all metrics from the default registry in Prometheus text format.
func Export() string {
	return DefaultRegistry.Export()
}

// Unregister removes a metric from the registry.
func (r *Registry) Unregister(name string) {
	r.metrics.Delete(name)
}

// Reset clears all metrics from the registry.
func (r *Registry) Reset() {
	r.metrics.Range(func(key, _ interface{}) bool {
		r.metrics.Delete(key)
		return true
	})
}
