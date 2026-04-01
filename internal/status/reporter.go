package status

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/example/freeradius-operator/api/v1alpha1"
)

type StatusReporter struct {
	client client.Client
}

func New(c client.Client) *StatusReporter {
	return &StatusReporter{client: c}
}

func boolToStatus(v bool) metav1.ConditionStatus {
	if v {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func setCondition(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, msg string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

// SetConditionLocal sets a condition in-place without persisting.
func (r *StatusReporter) SetConditionLocal(conditions *[]metav1.Condition, condType string, value bool, reason, msg string) {
	setCondition(conditions, condType, boolToStatus(value), reason, msg)
}

// FlushClusterStatus persists the current cluster status.
func (r *StatusReporter) FlushClusterStatus(ctx context.Context, cluster *v1alpha1.RadiusCluster) error {
	return r.client.Status().Update(ctx, cluster)
}

// RadiusCluster conditions

func (r *StatusReporter) SetProgressing(ctx context.Context, cluster *v1alpha1.RadiusCluster, progressing bool) error {
	reason, msg := "ReconcileComplete", "Reconciliation completed successfully"
	if progressing {
		reason, msg = "Reconciling", "Reconciliation is in progress"
	}
	setCondition(&cluster.Status.Conditions, ConditionProgressing, boolToStatus(progressing), reason, msg)
	return r.client.Status().Update(ctx, cluster)
}

func (r *StatusReporter) SetAvailable(ctx context.Context, cluster *v1alpha1.RadiusCluster, available bool, reason, msg string) error {
	setCondition(&cluster.Status.Conditions, ConditionAvailable, boolToStatus(available), reason, msg)
	return r.client.Status().Update(ctx, cluster)
}

func (r *StatusReporter) SetDegraded(ctx context.Context, cluster *v1alpha1.RadiusCluster, degraded bool, reason, msg string) error {
	setCondition(&cluster.Status.Conditions, ConditionDegraded, boolToStatus(degraded), reason, msg)
	return r.client.Status().Update(ctx, cluster)
}

func (r *StatusReporter) UpdateClusterStatus(ctx context.Context, cluster *v1alpha1.RadiusCluster, readyReplicas int32, currentImage string, podRestarts int32) error {
	cluster.Status.ReadyReplicas = readyReplicas
	cluster.Status.CurrentImage = currentImage
	cluster.Status.PodRestarts = podRestarts
	return r.client.Status().Update(ctx, cluster)
}

// RadiusClient conditions

func (r *StatusReporter) SetClientReady(ctx context.Context, rc *v1alpha1.RadiusClient, ready bool, reason, msg string) error {
	setCondition(&rc.Status.Conditions, ConditionReady, boolToStatus(ready), reason, msg)
	return r.client.Status().Update(ctx, rc)
}

func (r *StatusReporter) SetClientInvalid(ctx context.Context, rc *v1alpha1.RadiusClient, invalid bool, reason, msg string) error {
	setCondition(&rc.Status.Conditions, ConditionInvalid, boolToStatus(invalid), reason, msg)
	return r.client.Status().Update(ctx, rc)
}

// RadiusPolicy conditions

func (r *StatusReporter) SetPolicyReady(ctx context.Context, policy *v1alpha1.RadiusPolicy, ready bool, reason, msg string) error {
	setCondition(&policy.Status.Conditions, ConditionReady, boolToStatus(ready), reason, msg)
	return r.client.Status().Update(ctx, policy)
}

func (r *StatusReporter) SetPolicyInvalid(ctx context.Context, policy *v1alpha1.RadiusPolicy, invalid bool, reason, msg string) error {
	setCondition(&policy.Status.Conditions, ConditionInvalid, boolToStatus(invalid), reason, msg)
	return r.client.Status().Update(ctx, policy)
}
