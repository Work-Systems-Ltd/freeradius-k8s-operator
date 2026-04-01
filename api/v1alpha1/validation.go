package v1alpha1

import "fmt"

func validateReplicas(r int32) error {
	if r < 1 {
		return fmt.Errorf("replicas must be >= 1, got %d", r)
	}
	return nil
}
