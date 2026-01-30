package metrics

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	JobsEnqueuedTotal  = "jobs_enqueued_total"
	JobsCompletedTotal = "jobs_completed_total"
	JobsFailedTotal    = "jobs_failed_total"
	JobsCancelledTotal = "jobs_cancelled_total"
	PlaywrightSessions = "playwright_sessions_total"
	AlohaRunsTotal     = "aloha_runs_total"
)

var metricNames = []string{
	JobsEnqueuedTotal,
	JobsCompletedTotal,
	JobsFailedTotal,
	JobsCancelledTotal,
	PlaywrightSessions,
	AlohaRunsTotal,
}

// Metrics holds in-memory counters for emitted metrics.
type Metrics struct {
	mu       sync.Mutex
	counters map[string]uint64
}

// DefaultMetrics is the shared metrics instance.
var DefaultMetrics = NewMetrics()

// NewMetrics initializes a Metrics instance with known counters.
func NewMetrics() *Metrics {
	m := &Metrics{
		counters: make(map[string]uint64, len(metricNames)),
	}
	for _, name := range metricNames {
		m.counters[name] = 0
	}
	return m
}

// Inc increments the named counter when it is recognized.
func (m *Metrics) Inc(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.counters[name]; !ok {
		return
	}
	m.counters[name]++
}

// Emit writes all metrics in key=value format.
func (m *Metrics) Emit(w io.Writer) {
	if m == nil || w == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	ts := time.Now().UTC().Format(time.RFC3339)
	for _, name := range metricNames {
		fmt.Fprintf(w, "metric=%s value=%d ts=%s\n", name, m.counters[name], ts)
	}
}
