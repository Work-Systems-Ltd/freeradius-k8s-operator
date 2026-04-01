package status

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"
)

func conditionPresent(conditions []metav1.Condition, condType string) bool {
	for _, c := range conditions {
		if c.Type == condType {
			return true
		}
	}
	return false
}

func conditionStatus(conditions []metav1.Condition, condType string) (metav1.ConditionStatus, bool) {
	for _, c := range conditions {
		if c.Type == condType {
			return c.Status, true
		}
	}
	return "", false
}

func TestStatusConditionsPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var clusterConditions []metav1.Condition
		for _, condType := range []string{ConditionAvailable, ConditionProgressing, ConditionDegraded} {
			s := metav1.ConditionFalse
			if rapid.Bool().Draw(t, condType) {
				s = metav1.ConditionTrue
			}
			setCondition(&clusterConditions, condType, s, "TestReason", "test message")
		}
		for _, condType := range []string{ConditionAvailable, ConditionProgressing, ConditionDegraded} {
			if !conditionPresent(clusterConditions, condType) {
				t.Fatalf("missing condition type %q", condType)
			}
		}

		var clientConditions []metav1.Condition
		for _, condType := range []string{ConditionReady, ConditionInvalid} {
			s := metav1.ConditionFalse
			if rapid.Bool().Draw(t, "client_"+condType) {
				s = metav1.ConditionTrue
			}
			setCondition(&clientConditions, condType, s, "TestReason", "test message")
		}
		for _, condType := range []string{ConditionReady, ConditionInvalid} {
			if !conditionPresent(clientConditions, condType) {
				t.Fatalf("missing condition type %q", condType)
			}
		}
	})
}

func TestProgressingLifecycle(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var conditions []metav1.Condition

		setCondition(&conditions, ConditionProgressing, metav1.ConditionTrue, "Reconciling", "in progress")
		if s, _ := conditionStatus(conditions, ConditionProgressing); s != metav1.ConditionTrue {
			t.Fatalf("expected True, got %q", s)
		}

		setCondition(&conditions, ConditionProgressing, metav1.ConditionFalse, "ReconcileComplete", "done")
		if s, _ := conditionStatus(conditions, ConditionProgressing); s != metav1.ConditionFalse {
			t.Fatalf("expected False, got %q", s)
		}

		count := 0
		for _, c := range conditions {
			if c.Type == ConditionProgressing {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("expected 1 Progressing condition, got %d", count)
		}

		cycles := rapid.IntRange(1, 10).Draw(t, "cycles")
		for i := 0; i < cycles; i++ {
			setCondition(&conditions, ConditionProgressing, metav1.ConditionTrue, "Reconciling", "in progress")
			setCondition(&conditions, ConditionProgressing, metav1.ConditionFalse, "ReconcileComplete", "done")
		}
		if s, _ := conditionStatus(conditions, ConditionProgressing); s != metav1.ConditionFalse {
			t.Fatalf("expected False after %d cycles, got %q", cycles, s)
		}
	})
}

func TestStatusFieldsPopulated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		readyReplicas := rapid.Int32Range(0, 20).Draw(t, "readyReplicas")
		currentImage := rapid.StringMatching(`docker\.io/freeradius/freeradius-server:3\.[0-9]+\.[0-9]+`).Draw(t, "currentImage")
		podRestarts := rapid.Int32Range(0, 100).Draw(t, "podRestarts")

		var s struct {
			ReadyReplicas int32
			CurrentImage  string
			PodRestarts   int32
		}
		s.ReadyReplicas = readyReplicas
		s.CurrentImage = currentImage
		s.PodRestarts = podRestarts

		if s.ReadyReplicas != readyReplicas || s.CurrentImage != currentImage || s.PodRestarts != podRestarts {
			t.Fatal("status fields not set correctly")
		}
	})
}
