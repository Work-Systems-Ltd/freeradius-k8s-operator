// +kubebuilder:object:generate=true
package v1alpha1

import corev1 "k8s.io/api/core/v1"

type ServiceConfig struct {
	// +kubebuilder:validation:Enum=ClusterIP;LoadBalancer;NodePort
	// +kubebuilder:default=ClusterIP
	Type                  corev1.ServiceType                      `json:"type,omitempty"`
	LoadBalancerIP        string                                  `json:"loadBalancerIP,omitempty"`
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`
	Annotations           map[string]string                       `json:"annotations,omitempty"`
}

type CoAConfig struct {
	Enabled bool  `json:"enabled"`
	Port    int32 `json:"port,omitempty"`
}

type ServiceEndpointConfig struct {
	ServiceConfig `json:",inline"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas    int32              `json:"replicas,omitempty"`
	Autoscaling *AutoscalingConfig `json:"autoscaling,omitempty"`
}

type ServicesConfig struct {
	Auth       *ServiceEndpointConfig `json:"auth,omitempty"`
	Accounting *ServiceEndpointConfig `json:"accounting,omitempty"`
	CoA        *ServiceEndpointConfig `json:"coa,omitempty"`
}
