// +kubebuilder:object:generate=true
package v1alpha1

type SecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}
