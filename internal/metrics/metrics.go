package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	ReconcileTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "freeradius_operator_reconcile_total",
		Help: "Total reconciliation loops by namespace, name, kind, and result.",
	}, []string{"namespace", "name", "kind", "result"})

	ReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "freeradius_operator_reconcile_duration_seconds",
		Help:    "Reconciliation loop duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"namespace", "name", "kind"})
)

func init() {
	metrics.Registry.MustRegister(ReconcileTotal, ReconcileDuration)
}
