// Package v1alpha1 contains API Schema definitions for the radius.operator.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RadiusClientSpec defines the desired state of RadiusClient.
type RadiusClientSpec struct {
	// ClusterRef is the name of the owning RadiusCluster in the same namespace.
	ClusterRef string `json:"clusterRef"`
	// IP is the IPv4/IPv6 address or CIDR range of the NAS device.
	// +kubebuilder:validation:Pattern=`^((\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?|([0-9a-fA-F:]+)(\/\d{1,3})?)$`
	IP string `json:"ip"`
	// SecretRef references the Kubernetes Secret key containing the RADIUS shared secret.
	SecretRef SecretRef `json:"secretRef"`
	// NASType identifies the NAS device type (e.g. cisco, nokia, other).
	// +optional
	NASType string `json:"nasType,omitempty"`
	// Metadata is an optional map of arbitrary string key/value pairs stored as labels.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RadiusClientStatus defines the observed state of RadiusClient.
type RadiusClientStatus struct {
	// Conditions holds the status conditions for this resource.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RadiusClient is the Schema for the radiusclients API.
type RadiusClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RadiusClientSpec   `json:"spec,omitempty"`
	Status RadiusClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RadiusClientList contains a list of RadiusClient.
type RadiusClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadiusClient `json:"items"`
}
