package v1alpha1

import "fmt"

// validateReplicas returns an error if r is less than 1.
func validateReplicas(r int32) error {
	if r < 1 {
		return fmt.Errorf("replicas must be >= 1, got %d", r)
	}
	return nil
}
