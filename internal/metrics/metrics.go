// Package metrics registers Prometheus metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Request response time in seconds.
	RequestHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ratus_request_duration_seconds",
		Help:    "Request response time in seconds",
		Buckets: []float64{0.01, 0.1, 0.5, 1, 2, 5},
	}, []string{"topic", "endpoint", "status_code"})

	// Periodic background jobs execution time in seconds.
	ChoreHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ratus_chore_duration_seconds",
		Help:    "Periodic background jobs execution time in seconds",
		Buckets: []float64{0.01, 0.1, 0.5, 1, 2, 5},
	})

	// Task schedule delay in seconds.
	DelayGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ratus_task_schedule_delay_seconds",
		Help: "Task schedule delay in seconds",
	}, []string{"topic", "producer", "consumer"})

	// Task execution time in seconds.
	ExecutionGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ratus_task_execution_duration_seconds",
		Help: "Task execution time in seconds",
	}, []string{"topic", "producer", "consumer"})

	// Total number of tasks produced.
	ProducedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ratus_task_produced_count_total",
		Help: "Total number of tasks produced",
	}, []string{"topic", "producer"})

	// Total number of tasks consumed.
	ConsumedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ratus_task_consumed_count_total",
		Help: "Total number of tasks consumed",
	}, []string{"topic", "producer", "consumer"})

	// Total number of tasks committed.
	CommittedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ratus_task_committed_count_total",
		Help: "Total number of tasks committed",
	}, []string{"topic", "producer", "consumer"})
)
