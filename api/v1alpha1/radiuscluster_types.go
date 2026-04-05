package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RadiusClusterSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas    int32                       `json:"replicas,omitempty"`
	Image       string                      `json:"image"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Autoscaling *AutoscalingConfig          `json:"autoscaling,omitempty"`
	TLS         *TLSConfig                  `json:"tls,omitempty"`
	Probes      *ProbesConfig               `json:"probes,omitempty"`
	Modules     []ModuleConfig              `json:"modules,omitempty"`

	// ISP-scale features
	PDB                       *PDBConfig                        `json:"pdb,omitempty"`
	Affinity                  *corev1.Affinity                  `json:"affinity,omitempty"`
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	Service                   *ServiceConfig                    `json:"service,omitempty"`
	Services                  *ServicesConfig                   `json:"services,omitempty"`
	CoA                       *CoAConfig                        `json:"coa,omitempty"`
	InitResources             *corev1.ResourceRequirements      `json:"initResources,omitempty"`
}

type RadiusClusterStatus struct {
	ReadyReplicas int32              `json:"readyReplicas,omitempty"`
	CurrentImage  string             `json:"currentImage,omitempty"`
	PodRestarts   int32              `json:"podRestarts,omitempty"`
	ServiceIP     string             `json:"serviceIP,omitempty"`
	ExternalIPs   []string           `json:"externalIPs,omitempty"`
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type RadiusCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RadiusClusterSpec   `json:"spec,omitempty"`
	Status            RadiusClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RadiusClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadiusCluster `json:"items"`
}
