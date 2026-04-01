package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
	"github.com/example/freeradius-operator/internal/metrics"
	"github.com/example/freeradius-operator/internal/status"
)

// RadiusClientReconciler reconciles a RadiusClient object.
type RadiusClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Status *status.StatusReporter
}

// +kubebuilder:rbac:groups=radius.operator.io,resources=radiusclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=radius.operator.io,resources=radiusclients/status,verbs=get;update;patch

func (r *RadiusClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	start := time.Now()
	result := "success"

	defer func() {
		duration := time.Since(start).Seconds()
		metrics.ReconcileTotal.WithLabelValues(req.Namespace, req.Name, "RadiusClient", result).Inc()
		metrics.ReconcileDuration.WithLabelValues(req.Namespace, req.Name, "RadiusClient").Observe(duration)
	}()

	// Fetch the RadiusClient
	rc := &radiusv1alpha1.RadiusClient{}
	if err := r.Get(ctx, req.NamespacedName, rc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		result = "error"
		return ctrl.Result{}, err
	}

	// Validate clusterRef exists in the same namespace
	cluster := &radiusv1alpha1.RadiusCluster{}
	clusterKey := types.NamespacedName{Namespace: req.Namespace, Name: rc.Spec.ClusterRef}
	if err := r.Get(ctx, clusterKey, cluster); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("clusterRef not found, setting Invalid", "clusterRef", rc.Spec.ClusterRef)
			if statusErr := r.Status.SetClientInvalid(ctx, rc, true, "ClusterNotFound",
				fmt.Sprintf("RadiusCluster %q not found in namespace %q", rc.Spec.ClusterRef, req.Namespace)); statusErr != nil {
				logger.Error(statusErr, "unable to set Invalid status")
			}
			// Re-fetch and set Ready=false
			if err := r.Get(ctx, req.NamespacedName, rc); err == nil {
				_ = r.Status.SetClientReady(ctx, rc, false, "ClusterNotFound", "Waiting for RadiusCluster")
			}
			result = "error"
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		result = "error"
		return ctrl.Result{}, fmt.Errorf("fetching RadiusCluster %q: %w", rc.Spec.ClusterRef, err)
	}

	// ClusterRef is valid — set Ready and clear Invalid
	if statusErr := r.Status.SetClientReady(ctx, rc, true, "Valid", "RadiusClient is valid"); statusErr != nil {
		logger.Error(statusErr, "unable to set Ready status")
	}
	if err := r.Get(ctx, req.NamespacedName, rc); err == nil {
		if statusErr := r.Status.SetClientInvalid(ctx, rc, false, "Valid", "RadiusClient is valid"); statusErr != nil {
			logger.Error(statusErr, "unable to clear Invalid status")
		}
	}

	logger.Info("reconciliation complete")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RadiusClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&radiusv1alpha1.RadiusClient{}).
		Complete(r)
}
