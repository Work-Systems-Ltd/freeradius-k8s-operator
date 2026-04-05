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

type RadiusdLogConfig struct {
	// +kubebuilder:validation:Enum=stdout;syslog;files
	Destination  string `json:"destination,omitempty"`
	Auth         *bool  `json:"auth,omitempty"`
	AuthBadpass  *bool  `json:"authBadpass,omitempty"`
	AuthGoodpass *bool  `json:"authGoodpass,omitempty"`
}

type RadiusdSecurityConfig struct {
	MaxAttributes int32 `json:"maxAttributes,omitempty"`
	RejectDelay   int32 `json:"rejectDelay,omitempty"`
}

type RadiusdThreadPool struct {
	StartServers         int32 `json:"startServers,omitempty"`
	MaxServers           int32 `json:"maxServers,omitempty"`
	MinSpareServers      int32 `json:"minSpareServers,omitempty"`
	MaxSpareServers      int32 `json:"maxSpareServers,omitempty"`
	MaxRequestsPerServer int32 `json:"maxRequestsPerServer,omitempty"`
}

type RadiusdConfig struct {
	Log            *RadiusdLogConfig      `json:"log,omitempty"`
	Security       *RadiusdSecurityConfig `json:"security,omitempty"`
	ThreadPool     *RadiusdThreadPool     `json:"threadPool,omitempty"`
	MaxRequestTime int32                  `json:"maxRequestTime,omitempty"`
	MaxRequests    int32                  `json:"maxRequests,omitempty"`
	RawConfig      string                 `json:"rawConfig,omitempty"`
}
