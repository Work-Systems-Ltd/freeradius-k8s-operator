// Package status provides the StatusReporter for writing status conditions
// back to RadiusCluster, RadiusClient, and RadiusPolicy resources.
package status

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/example/freeradius-operator/api/v1alpha1"
)

// Condition type constants for RadiusCluster.
const (
	ConditionAvailable   = "Available"
	ConditionProgressing = "Progressing"
	ConditionDegraded    = "Degraded"
)

// Condition type constants for RadiusClient and RadiusPolicy.
const (
	ConditionReady   = "Ready"
	ConditionInvalid = "Invalid"
)

// StatusReporter writes status conditions back to resources using the status subresource.
type StatusReporter struct {
	client client.Client
}

// New returns a new StatusReporter backed by the given client.
func New(c client.Client) *StatusReporter {
	return &StatusReporter{client: c}
}

// setCondition manipulates a conditions slice in-place using meta.SetStatusCondition.
// This is a pure helper that can be tested without a k8s client.
func setCondition(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, msg string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
}

// SetConditionLocal manipulates a conditions slice in-place without writing to the API server.
// Use FlushClusterStatus to persist after batching multiple condition changes.
func (r *StatusReporter) SetConditionLocal(conditions *[]metav1.Condition, condType string, value bool, reason, msg string) {
	status := metav1.ConditionFalse
	if value {
		status = metav1.ConditionTrue
	}
	setCondition(conditions, condType, status, reason, msg)
}

// FlushClusterStatus writes the current status of a RadiusCluster to the API server.
func (r *StatusReporter) FlushClusterStatus(ctx context.Context, cluster *v1alpha1.RadiusCluster) error {
	return r.client.Status().Update(ctx, cluster)
}

// --- RadiusCluster helpers ---

// SetProgressing sets the Progressing condition on a RadiusCluster.
func (r *StatusReporter) SetProgressing(ctx context.Context, cluster *v1alpha1.RadiusCluster, progressing bool) error {
	status := metav1.ConditionFalse
	reason := "ReconcileComplete"
	msg := "Reconciliation completed successfully"
	if progressing {
		status = metav1.ConditionTrue
		reason = "Reconciling"
		msg = "Reconciliation is in progress"
	}
	setCondition(&cluster.Status.Conditions, ConditionProgressing, status, reason, msg)
	return r.client.Status().Update(ctx, cluster)
}

// SetAvailable sets the Available condition on a RadiusCluster.
func (r *StatusReporter) SetAvailable(ctx context.Context, cluster *v1alpha1.RadiusCluster, available bool, reason, msg string) error {
	status := metav1.ConditionFalse
	if available {
		status = metav1.ConditionTrue
	}
	setCondition(&cluster.Status.Conditions, ConditionAvailable, status, reason, msg)
	return r.client.Status().Update(ctx, cluster)
}

// SetDegraded sets the Degraded condition on a RadiusCluster.
func (r *StatusReporter) SetDegraded(ctx context.Context, cluster *v1alpha1.RadiusCluster, degraded bool, reason, msg string) error {
	status := metav1.ConditionFalse
	if degraded {
		status = metav1.ConditionTrue
	}
	setCondition(&cluster.Status.Conditions, ConditionDegraded, status, reason, msg)
	return r.client.Status().Update(ctx, cluster)
}

// UpdateClusterStatus updates the readyReplicas, currentImage, and podRestarts fields
// on a RadiusCluster and writes the status via the status subresource.
func (r *StatusReporter) UpdateClusterStatus(ctx context.Context, cluster *v1alpha1.RadiusCluster, readyReplicas int32, currentImage string, podRestarts int32) error {
	cluster.Status.ReadyReplicas = readyReplicas
	cluster.Status.CurrentImage = currentImage
	cluster.Status.PodRestarts = podRestarts
	return r.client.Status().Update(ctx, cluster)
}

// --- RadiusClient helpers ---

// SetClientReady sets the Ready condition on a RadiusClient.
func (r *StatusReporter) SetClientReady(ctx context.Context, rc *v1alpha1.RadiusClient, ready bool, reason, msg string) error {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}
	setCondition(&rc.Status.Conditions, ConditionReady, status, reason, msg)
	return r.client.Status().Update(ctx, rc)
}

// SetClientInvalid sets the Invalid condition on a RadiusClient.
func (r *StatusReporter) SetClientInvalid(ctx context.Context, rc *v1alpha1.RadiusClient, invalid bool, reason, msg string) error {
	status := metav1.ConditionFalse
	if invalid {
		status = metav1.ConditionTrue
	}
	setCondition(&rc.Status.Conditions, ConditionInvalid, status, reason, msg)
	return r.client.Status().Update(ctx, rc)
}

// --- RadiusPolicy helpers ---

// SetPolicyReady sets the Ready condition on a RadiusPolicy.
func (r *StatusReporter) SetPolicyReady(ctx context.Context, policy *v1alpha1.RadiusPolicy, ready bool, reason, msg string) error {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}
	setCondition(&policy.Status.Conditions, ConditionReady, status, reason, msg)
	return r.client.Status().Update(ctx, policy)
}

// SetPolicyInvalid sets the Invalid condition on a RadiusPolicy.
func (r *StatusReporter) SetPolicyInvalid(ctx context.Context, policy *v1alpha1.RadiusPolicy, invalid bool, reason, msg string) error {
	status := metav1.ConditionFalse
	if invalid {
		status = metav1.ConditionTrue
	}
	setCondition(&policy.Status.Conditions, ConditionInvalid, status, reason, msg)
	return r.client.Status().Update(ctx, policy)
}
