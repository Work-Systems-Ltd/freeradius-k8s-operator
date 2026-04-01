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

// validStages is the set of allowed FreeRADIUS processing stages.
var validStages = map[string]bool{
	"authorize":    true,
	"authenticate": true,
	"preacct":      true,
	"accounting":   true,
	"post-auth":    true,
	"pre-proxy":    true,
	"post-proxy":   true,
	"session":      true,
}

// validActionTypes is the set of allowed policy action types.
var validActionTypes = map[string]bool{
	"set":    true,
	"call":   true,
	"reject": true,
	"accept": true,
}

// RadiusPolicyReconciler reconciles a RadiusPolicy object.
type RadiusPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Status *status.StatusReporter
}

// +kubebuilder:rbac:groups=radius.operator.io,resources=radiuspolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=radius.operator.io,resources=radiuspolicies/status,verbs=get;update;patch

func (r *RadiusPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	start := time.Now()
	result := "success"

	defer func() {
		duration := time.Since(start).Seconds()
		metrics.ReconcileTotal.WithLabelValues(req.Namespace, req.Name, "RadiusPolicy", result).Inc()
		metrics.ReconcileDuration.WithLabelValues(req.Namespace, req.Name, "RadiusPolicy").Observe(duration)
	}()

	// Fetch the RadiusPolicy
	policy := &radiusv1alpha1.RadiusPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		result = "error"
		return ctrl.Result{}, err
	}

	// Validate stage
	if !validStages[policy.Spec.Stage] {
		logger.Info("invalid stage value", "stage", policy.Spec.Stage)
		if statusErr := r.Status.SetPolicyInvalid(ctx, policy, true, "InvalidStage",
			fmt.Sprintf("unrecognized stage %q", policy.Spec.Stage)); statusErr != nil {
			logger.Error(statusErr, "unable to set Invalid status")
		}
		if err := r.Get(ctx, req.NamespacedName, policy); err == nil {
			_ = r.Status.SetPolicyReady(ctx, policy, false, "InvalidStage", "Invalid stage value")
		}
		result = "error"
		return ctrl.Result{}, nil
	}

	// Validate action types
	for _, action := range policy.Spec.Actions {
		if !validActionTypes[action.Type] {
			logger.Info("invalid action type", "actionType", action.Type)
			if statusErr := r.Status.SetPolicyInvalid(ctx, policy, true, "InvalidActionType",
				fmt.Sprintf("unrecognized action type %q", action.Type)); statusErr != nil {
				logger.Error(statusErr, "unable to set Invalid status")
			}
			if err := r.Get(ctx, req.NamespacedName, policy); err == nil {
				_ = r.Status.SetPolicyReady(ctx, policy, false, "InvalidActionType", "Invalid action type")
			}
			result = "error"
			return ctrl.Result{}, nil
		}
	}

	// Validate clusterRef exists
	cluster := &radiusv1alpha1.RadiusCluster{}
	clusterKey := types.NamespacedName{Namespace: req.Namespace, Name: policy.Spec.ClusterRef}
	if err := r.Get(ctx, clusterKey, cluster); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("clusterRef not found", "clusterRef", policy.Spec.ClusterRef)
			if statusErr := r.Status.SetPolicyInvalid(ctx, policy, true, "ClusterNotFound",
				fmt.Sprintf("RadiusCluster %q not found in namespace %q", policy.Spec.ClusterRef, req.Namespace)); statusErr != nil {
				logger.Error(statusErr, "unable to set Invalid status")
			}
			if err := r.Get(ctx, req.NamespacedName, policy); err == nil {
				_ = r.Status.SetPolicyReady(ctx, policy, false, "ClusterNotFound", "Waiting for RadiusCluster")
			}
			result = "error"
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		result = "error"
		return ctrl.Result{}, fmt.Errorf("fetching RadiusCluster %q: %w", policy.Spec.ClusterRef, err)
	}

	// All valid — set Ready and clear Invalid
	if statusErr := r.Status.SetPolicyReady(ctx, policy, true, "Valid", "RadiusPolicy is valid"); statusErr != nil {
		logger.Error(statusErr, "unable to set Ready status")
	}
	if err := r.Get(ctx, req.NamespacedName, policy); err == nil {
		if statusErr := r.Status.SetPolicyInvalid(ctx, policy, false, "Valid", "RadiusPolicy is valid"); statusErr != nil {
			logger.Error(statusErr, "unable to clear Invalid status")
		}
	}

	logger.Info("reconciliation complete")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RadiusPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&radiusv1alpha1.RadiusPolicy{}).
		Complete(r)
}
