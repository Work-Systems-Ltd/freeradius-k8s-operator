package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type SecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type SQLConfig struct {
	// +kubebuilder:validation:Enum=mysql;postgresql;sqlite;mssql;oracle;mongo
	Dialect     string    `json:"dialect"`
	Server      string    `json:"server"`
	Port        int32     `json:"port"`
	Database    string    `json:"database"`
	Login       string    `json:"login"`
	PasswordRef SecretRef `json:"passwordRef"`
}

type LDAPConfig struct {
	Server      string    `json:"server"`
	Port        int32     `json:"port"`
	BaseDN      string    `json:"baseDN"`
	Identity    string    `json:"identity"`
	PasswordRef SecretRef `json:"passwordRef"`
}

type EAPTLSConfig struct {
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
}

type EAPTTLSConfig struct {
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	VirtualServer  string `json:"virtualServer,omitempty"`
}

type EAPPEAPConfig struct {
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	VirtualServer  string `json:"virtualServer,omitempty"`
}

type EAPConfig struct {
	DefaultEAPType string         `json:"defaultEAPType,omitempty"`
	TLS            *EAPTLSConfig  `json:"tls,omitempty"`
	TTLS           *EAPTTLSConfig `json:"ttls,omitempty"`
	PEAP           *EAPPEAPConfig `json:"peap,omitempty"`
}

type RESTStageConfig struct {
	URI    string `json:"uri,omitempty"`
	Method string `json:"method,omitempty"`
}

type RESTConfig struct {
	ConnectURI   string           `json:"connectURI"`
	Auth         string           `json:"auth,omitempty"`
	PasswordRef  *SecretRef       `json:"passwordRef,omitempty"`
	Authorize    *RESTStageConfig `json:"authorize,omitempty"`
	Authenticate *RESTStageConfig `json:"authenticate,omitempty"`
	Preacct      *RESTStageConfig `json:"preacct,omitempty"`
	Accounting   *RESTStageConfig `json:"accounting,omitempty"`
	PostAuth     *RESTStageConfig `json:"postAuth,omitempty"`
	PreProxy     *RESTStageConfig `json:"preProxy,omitempty"`
	PostProxy    *RESTStageConfig `json:"postProxy,omitempty"`
}

type RedisConfig struct {
	Server      string     `json:"server"`
	Port        int32      `json:"port"`
	Database    int32      `json:"database,omitempty"`
	PasswordRef *SecretRef `json:"passwordRef,omitempty"`
}

type ModuleConfig struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Enabled bool         `json:"enabled"`
	SQL     *SQLConfig   `json:"sql,omitempty"`
	LDAP    *LDAPConfig  `json:"ldap,omitempty"`
	EAP     *EAPConfig   `json:"eap,omitempty"`
	REST    *RESTConfig  `json:"rest,omitempty"`
	Redis   *RedisConfig `json:"redis,omitempty"`
}

type PDBConfig struct {
	MinAvailable   *intstr.IntOrString `json:"minAvailable,omitempty"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

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
