// Package metrics registers Prometheus metrics for the freeradius-operator.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconcileTotal counts reconciliation loops, labeled by namespace, name, kind, and result.
	ReconcileTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "freeradius_operator_reconcile_total",
		Help: "Total number of reconciliation loops executed, labeled by namespace, name, kind, and result.",
	}, []string{"namespace", "name", "kind", "result"})

	// ReconcileDuration measures reconciliation loop duration in seconds.
	ReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "freeradius_operator_reconcile_duration_seconds",
		Help:    "Duration of reconciliation loops in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"namespace", "name", "kind"})
)

func init() {
	metrics.Registry.MustRegister(ReconcileTotal, ReconcileDuration)
}
