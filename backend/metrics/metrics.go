package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ScriptSuccessTotal tracks total successful script runs
	ScriptSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "script_success_total",
			Help: "Total number of successful script executions.",
		},
		[]string{"script_name", "environment"},
	)

	// ScriptFailureTotal tracks total failed script runs
	ScriptFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "script_failure_total",
			Help: "Total number of failed script executions.",
		},
		[]string{"script_name", "environment"},
	)

	// ScriptDurationSeconds tracks execution duration of script runs in seconds
	ScriptDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "script_duration_seconds",
			Help:    "Execution duration of scripts in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"script_name", "environment"},
	)
)

func init() {
	prometheus.MustRegister(ScriptSuccessTotal)
	prometheus.MustRegister(ScriptFailureTotal)
	prometheus.MustRegister(ScriptDurationSeconds)
}
