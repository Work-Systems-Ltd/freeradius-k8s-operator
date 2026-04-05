package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type RadiusClientSpec struct {
	ClusterRef string `json:"clusterRef"`
	// +kubebuilder:validation:Pattern=`^((\d{1,3}\.){3}\d{1,3}(/\d{1,2})?|[0-9a-fA-F:.]*:[0-9a-fA-F:.]*(/\d{1,3})?)$`
	IP        string            `json:"ip"`
	SecretRef SecretRef         `json:"secretRef"`
	NASType   string            `json:"nasType,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	RawConfig string            `json:"rawConfig,omitempty"`
}

type RadiusClientStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type RadiusClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RadiusClientSpec   `json:"spec,omitempty"`
	Status            RadiusClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RadiusClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadiusClient `json:"items"`
}
