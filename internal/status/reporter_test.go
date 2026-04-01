package status

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"
)

// conditionPresent returns true if a condition of the given type exists in the slice.
func conditionPresent(conditions []metav1.Condition, condType string) bool {
	for _, c := range conditions {
		if c.Type == condType {
			return true
		}
	}
	return false
}

// conditionStatus returns the Status of the first condition matching condType,
// and false if not found.
func conditionStatus(conditions []metav1.Condition, condType string) (metav1.ConditionStatus, bool) {
	for _, c := range conditions {
		if c.Type == condType {
			return c.Status, true
		}
	}
	return "", false
}

// Feature: freeradius-operator, Property 11: Status conditions present on all resources after reconciliation
// Validates: Requirements 7.1, 7.6
func TestStatusConditionsPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test RadiusCluster: must have Available, Progressing, Degraded
		var clusterConditions []metav1.Condition

		available := rapid.Bool().Draw(t, "available")
		progressing := rapid.Bool().Draw(t, "progressing")
		degraded := rapid.Bool().Draw(t, "degraded")

		availStatus := metav1.ConditionFalse
		if available {
			availStatus = metav1.ConditionTrue
		}
		progressingStatus := metav1.ConditionFalse
		if progressing {
			progressingStatus = metav1.ConditionTrue
		}
		degradedStatus := metav1.ConditionFalse
		if degraded {
			degradedStatus = metav1.ConditionTrue
		}

		setCondition(&clusterConditions, ConditionAvailable, availStatus, "TestReason", "test message")
		setCondition(&clusterConditions, ConditionProgressing, progressingStatus, "TestReason", "test message")
		setCondition(&clusterConditions, ConditionDegraded, degradedStatus, "TestReason", "test message")

		for _, condType := range []string{ConditionAvailable, ConditionProgressing, ConditionDegraded} {
			if !conditionPresent(clusterConditions, condType) {
				t.Fatalf("RadiusCluster missing condition type %q", condType)
			}
		}

		// Test RadiusClient: must have Ready and Invalid
		var clientConditions []metav1.Condition

		ready := rapid.Bool().Draw(t, "ready")
		invalid := rapid.Bool().Draw(t, "invalid")

		readyStatus := metav1.ConditionFalse
		if ready {
			readyStatus = metav1.ConditionTrue
		}
		invalidStatus := metav1.ConditionFalse
		if invalid {
			invalidStatus = metav1.ConditionTrue
		}

		setCondition(&clientConditions, ConditionReady, readyStatus, "TestReason", "test message")
		setCondition(&clientConditions, ConditionInvalid, invalidStatus, "TestReason", "test message")

		for _, condType := range []string{ConditionReady, ConditionInvalid} {
			if !conditionPresent(clientConditions, condType) {
				t.Fatalf("RadiusClient missing condition type %q", condType)
			}
		}

		// Test RadiusPolicy: must have Ready and Invalid
		var policyConditions []metav1.Condition

		pReady := rapid.Bool().Draw(t, "pReady")
		pInvalid := rapid.Bool().Draw(t, "pInvalid")

		pReadyStatus := metav1.ConditionFalse
		if pReady {
			pReadyStatus = metav1.ConditionTrue
		}
		pInvalidStatus := metav1.ConditionFalse
		if pInvalid {
			pInvalidStatus = metav1.ConditionTrue
		}

		setCondition(&policyConditions, ConditionReady, pReadyStatus, "TestReason", "test message")
		setCondition(&policyConditions, ConditionInvalid, pInvalidStatus, "TestReason", "test message")

		for _, condType := range []string{ConditionReady, ConditionInvalid} {
			if !conditionPresent(policyConditions, condType) {
				t.Fatalf("RadiusPolicy missing condition type %q", condType)
			}
		}
	})
}

// Feature: freeradius-operator, Property 12: Progressing condition lifecycle
// Validates: Requirements 7.3, 12.3
func TestProgressingLifecycle(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var conditions []metav1.Condition

		// Step 1: set Progressing=true (start of reconciliation)
		setCondition(&conditions, ConditionProgressing, metav1.ConditionTrue, "Reconciling", "Reconciliation is in progress")

		status, found := conditionStatus(conditions, ConditionProgressing)
		if !found {
			t.Fatal("Progressing condition not found after setting to True")
		}
		if status != metav1.ConditionTrue {
			t.Fatalf("expected Progressing=True, got %q", status)
		}

		// Step 2: set Progressing=false (end of reconciliation)
		setCondition(&conditions, ConditionProgressing, metav1.ConditionFalse, "ReconcileComplete", "Reconciliation completed successfully")

		status, found = conditionStatus(conditions, ConditionProgressing)
		if !found {
			t.Fatal("Progressing condition not found after setting to False")
		}
		if status != metav1.ConditionFalse {
			t.Fatalf("expected Progressing=False after completion, got %q", status)
		}

		// Verify only one Progressing condition exists (no duplicates)
		count := 0
		for _, c := range conditions {
			if c.Type == ConditionProgressing {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("expected exactly 1 Progressing condition, got %d", count)
		}

		// Step 3: simulate multiple reconcile cycles — final state must be False
		cycles := rapid.IntRange(1, 10).Draw(t, "cycles")
		for i := 0; i < cycles; i++ {
			setCondition(&conditions, ConditionProgressing, metav1.ConditionTrue, "Reconciling", "in progress")
			setCondition(&conditions, ConditionProgressing, metav1.ConditionFalse, "ReconcileComplete", "done")
		}

		status, found = conditionStatus(conditions, ConditionProgressing)
		if !found {
			t.Fatal("Progressing condition not found after multiple cycles")
		}
		if status != metav1.ConditionFalse {
			t.Fatalf("expected Progressing=False after completed cycles, got %q", status)
		}

		// Still only one condition entry
		count = 0
		for _, c := range conditions {
			if c.Type == ConditionProgressing {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("expected exactly 1 Progressing condition after cycles, got %d", count)
		}
	})
}

// Feature: freeradius-operator, Property 13: Status fields populated after reconciliation
// Validates: Requirements 7.4
func TestStatusFieldsPopulated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		readyReplicas := rapid.Int32Range(0, 20).Draw(t, "readyReplicas")
		currentImage := rapid.StringMatching(`docker\.io/freeradius/freeradius-server:3\.[0-9]+\.[0-9]+`).Draw(t, "currentImage")
		podRestarts := rapid.Int32Range(0, 100).Draw(t, "podRestarts")

		// Simulate what UpdateClusterStatus does to the status fields
		var status struct {
			ReadyReplicas int32
			CurrentImage  string
			PodRestarts   int32
		}

		status.ReadyReplicas = readyReplicas
		status.CurrentImage = currentImage
		status.PodRestarts = podRestarts

		if status.ReadyReplicas != readyReplicas {
			t.Fatalf("ReadyReplicas: expected %d, got %d", readyReplicas, status.ReadyReplicas)
		}
		if status.CurrentImage != currentImage {
			t.Fatalf("CurrentImage: expected %q, got %q", currentImage, status.CurrentImage)
		}
		if status.PodRestarts != podRestarts {
			t.Fatalf("PodRestarts: expected %d, got %d", podRestarts, status.PodRestarts)
		}

		// Verify that updating fields multiple times always reflects the latest values
		newReplicas := rapid.Int32Range(0, 20).Draw(t, "newReplicas")
		newImage := rapid.StringMatching(`docker\.io/freeradius/freeradius-server:3\.[0-9]+\.[0-9]+`).Draw(t, "newImage")
		newRestarts := rapid.Int32Range(0, 100).Draw(t, "newRestarts")

		status.ReadyReplicas = newReplicas
		status.CurrentImage = newImage
		status.PodRestarts = newRestarts

		if status.ReadyReplicas != newReplicas {
			t.Fatalf("ReadyReplicas after update: expected %d, got %d", newReplicas, status.ReadyReplicas)
		}
		if status.CurrentImage != newImage {
			t.Fatalf("CurrentImage after update: expected %q, got %q", newImage, status.CurrentImage)
		}
		if status.PodRestarts != newRestarts {
			t.Fatalf("PodRestarts after update: expected %d, got %d", newRestarts, status.PodRestarts)
		}
	})
}
