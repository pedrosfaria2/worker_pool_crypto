package pool

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	tasksSubmitted *prometheus.CounterVec
	tasksCompleted *prometheus.CounterVec
	tasksFailed    *prometheus.CounterVec
	taskDuration   *prometheus.HistogramVec
	queueSize      prometheus.Gauge
	workersBusy    prometheus.Gauge
}

func NewMetrics(namespace string) Metrics {
	m := &metrics{
		tasksSubmitted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tasks_submitted_total",
			},
			[]string{"type"},
		),
		tasksCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tasks_completed_total",
			},
			[]string{"type"},
		),
		tasksFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tasks_failed_total",
			},
			[]string{"type"},
		),
		taskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "task_duration_milliseconds",
				Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 10000},
			},
			[]string{"type"},
		),
		queueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_size_current",
			},
		),
		workersBusy: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "workers_busy_current",
			},
		),
	}

	return m
}

func (m *metrics) TaskSubmitted(taskType string) {
	m.tasksSubmitted.WithLabelValues(taskType).Inc()
}

func (m *metrics) TaskCompleted(taskType string, durationMs float64) {
	m.tasksCompleted.WithLabelValues(taskType).Inc()
	m.taskDuration.WithLabelValues(taskType).Observe(durationMs)
}

func (m *metrics) TaskFailed(taskType string) {
	m.tasksFailed.WithLabelValues(taskType).Inc()
}

func (m *metrics) QueueSize(size int) {
	m.queueSize.Set(float64(size))
}

func (m *metrics) WorkersBusy(count int) {
	m.workersBusy.Set(float64(count))
}
