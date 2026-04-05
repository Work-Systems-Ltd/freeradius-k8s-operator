// +kubebuilder:object:generate=true
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type AutoscalingConfig struct {
	Enabled                        bool  `json:"enabled"`
	MinReplicas                    int32 `json:"minReplicas,omitempty"`
	MaxReplicas                    int32 `json:"maxReplicas,omitempty"`
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage,omitempty"`
}

type TLSConfig struct {
	Enabled   bool      `json:"enabled"`
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

type ProbesConfig struct {
	Liveness  *corev1.Probe `json:"liveness,omitempty"`
	Readiness *corev1.Probe `json:"readiness,omitempty"`
}

type PDBConfig struct {
	MinAvailable   *intstr.IntOrString `json:"minAvailable,omitempty"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}
