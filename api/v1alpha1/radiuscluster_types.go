// Package v1alpha1 contains API Schema definitions for the radius.operator.io v1alpha1 API group.
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretRef references a key within a Kubernetes Secret.
type SecretRef struct {
	// Name is the name of the Kubernetes Secret.
	Name string `json:"name"`
	// Key is the key within the Secret.
	Key string `json:"key"`
}

// SQLConfig holds configuration for the rlm_sql module.
type SQLConfig struct {
	// Dialect is the SQL dialect. One of: mysql, postgresql, sqlite, mssql, oracle, mongo.
	// +kubebuilder:validation:Enum=mysql;postgresql;sqlite;mssql;oracle;mongo
	Dialect string `json:"dialect"`
	// Server is the SQL server hostname or IP.
	Server string `json:"server"`
	// Port is the SQL server port.
	Port int32 `json:"port"`
	// Database is the database name.
	Database string `json:"database"`
	// Login is the database username.
	Login string `json:"login"`
	// PasswordRef references the Kubernetes Secret containing the database password.
	PasswordRef SecretRef `json:"passwordRef"`
}

// LDAPConfig holds configuration for the rlm_ldap module.
type LDAPConfig struct {
	// Server is the LDAP server hostname or IP.
	Server string `json:"server"`
	// Port is the LDAP server port.
	Port int32 `json:"port"`
	// BaseDN is the LDAP base distinguished name.
	BaseDN string `json:"baseDN"`
	// Identity is the full LDAP bind DN (e.g. cn=admin,dc=example,dc=org).
	Identity string `json:"identity"`
	// PasswordRef references the Kubernetes Secret containing the LDAP bind password.
	PasswordRef SecretRef `json:"passwordRef"`
}

// EAPTLSConfig holds TLS sub-type configuration for EAP.
type EAPTLSConfig struct {
	// CertFile is the path to the certificate file.
	CertFile string `json:"certFile,omitempty"`
	// KeyFile is the path to the private key file.
	KeyFile string `json:"keyFile,omitempty"`
}

// EAPTTLSConfig holds TTLS sub-type configuration for EAP.
type EAPTTLSConfig struct {
	// DefaultEAPType is the default inner EAP type for TTLS.
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	// VirtualServer is the virtual server to use for TTLS inner tunnel.
	VirtualServer string `json:"virtualServer,omitempty"`
}

// EAPPEAPConfig holds PEAP sub-type configuration for EAP.
type EAPPEAPConfig struct {
	// DefaultEAPType is the default inner EAP type for PEAP.
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	// VirtualServer is the virtual server to use for PEAP inner tunnel.
	VirtualServer string `json:"virtualServer,omitempty"`
}

// EAPConfig holds configuration for the rlm_eap module.
type EAPConfig struct {
	// DefaultEAPType is the default EAP type.
	DefaultEAPType string `json:"defaultEAPType,omitempty"`
	// TLS holds EAP-TLS sub-type configuration.
	TLS *EAPTLSConfig `json:"tls,omitempty"`
	// TTLS holds EAP-TTLS sub-type configuration.
	TTLS *EAPTTLSConfig `json:"ttls,omitempty"`
	// PEAP holds EAP-PEAP sub-type configuration.
	PEAP *EAPPEAPConfig `json:"peap,omitempty"`
}

// RESTStageConfig holds per-stage URI/method overrides for the rlm_rest module.
type RESTStageConfig struct {
	// URI is the URI for this processing stage.
	URI string `json:"uri,omitempty"`
	// Method is the HTTP method for this processing stage.
	Method string `json:"method,omitempty"`
}

// RESTConfig holds configuration for the rlm_rest module.
type RESTConfig struct {
	// ConnectURI is the base URI for the REST server.
	ConnectURI string `json:"connectURI"`
	// Auth is the authentication type (e.g. bearer, basic).
	Auth string `json:"auth,omitempty"`
	// PasswordRef references the Kubernetes Secret containing the REST auth credential.
	PasswordRef *SecretRef `json:"passwordRef,omitempty"`
	// Authorize holds per-stage config for the authorize phase.
	Authorize *RESTStageConfig `json:"authorize,omitempty"`
	// Authenticate holds per-stage config for the authenticate phase.
	Authenticate *RESTStageConfig `json:"authenticate,omitempty"`
	// Preacct holds per-stage config for the preacct phase.
	Preacct *RESTStageConfig `json:"preacct,omitempty"`
	// Accounting holds per-stage config for the accounting phase.
	Accounting *RESTStageConfig `json:"accounting,omitempty"`
	// PostAuth holds per-stage config for the post-auth phase.
	PostAuth *RESTStageConfig `json:"postAuth,omitempty"`
	// PreProxy holds per-stage config for the pre-proxy phase.
	PreProxy *RESTStageConfig `json:"preProxy,omitempty"`
	// PostProxy holds per-stage config for the post-proxy phase.
	PostProxy *RESTStageConfig `json:"postProxy,omitempty"`
}

// RedisConfig holds configuration for the rlm_redis module.
type RedisConfig struct {
	// Server is the Redis server hostname or IP.
	Server string `json:"server"`
	// Port is the Redis server port.
	Port int32 `json:"port"`
	// Database is the Redis database index.
	Database int32 `json:"database,omitempty"`
	// PasswordRef references the Kubernetes Secret containing the Redis password.
	PasswordRef *SecretRef `json:"passwordRef,omitempty"`
}

// ModuleConfig declares a single RLM backend instance.
type ModuleConfig struct {
	// Name is the module instance identifier.
	Name string `json:"name"`
	// Type is the RLM backend type (e.g. rlm_sql, rlm_ldap).
	Type string `json:"type"`
	// Enabled controls whether this module is active.
	Enabled bool `json:"enabled"`
	// SQL holds configuration for rlm_sql modules.
	SQL *SQLConfig `json:"sql,omitempty"`
	// LDAP holds configuration for rlm_ldap modules.
	LDAP *LDAPConfig `json:"ldap,omitempty"`
	// EAP holds configuration for rlm_eap modules.
	EAP *EAPConfig `json:"eap,omitempty"`
	// REST holds configuration for rlm_rest modules.
	REST *RESTConfig `json:"rest,omitempty"`
	// Redis holds configuration for rlm_redis modules.
	Redis *RedisConfig `json:"redis,omitempty"`
}

// AutoscalingConfig holds HPA configuration for a RadiusCluster.
type AutoscalingConfig struct {
	// Enabled controls whether HPA is active.
	Enabled bool `json:"enabled"`
	// MinReplicas is the minimum number of replicas.
	MinReplicas int32 `json:"minReplicas,omitempty"`
	// MaxReplicas is the maximum number of replicas.
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
	// TargetCPUUtilizationPercentage is the target CPU utilization for HPA scaling.
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage,omitempty"`
}

// TLSConfig holds TLS configuration for a RadiusCluster.
type TLSConfig struct {
	// Enabled controls whether TLS is active.
	Enabled bool `json:"enabled"`
	// SecretRef references the Kubernetes Secret containing the TLS certificate and key.
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

// ProbesConfig holds liveness and readiness probe configuration.
type ProbesConfig struct {
	// Liveness is the liveness probe configuration.
	Liveness *corev1.Probe `json:"liveness,omitempty"`
	// Readiness is the readiness probe configuration.
	Readiness *corev1.Probe `json:"readiness,omitempty"`
}

// RadiusClusterSpec defines the desired state of RadiusCluster.
type RadiusClusterSpec struct {
	// Replicas is the desired number of FreeRADIUS pod replicas.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`
	// Image is the FreeRADIUS container image and tag.
	Image string `json:"image"`
	// Resources defines CPU and memory requests/limits for the FreeRADIUS container.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Autoscaling holds HPA configuration.
	Autoscaling *AutoscalingConfig `json:"autoscaling,omitempty"`
	// TLS holds TLS configuration.
	TLS *TLSConfig `json:"tls,omitempty"`
	// Probes holds liveness and readiness probe configuration.
	Probes *ProbesConfig `json:"probes,omitempty"`
	// Modules is the list of RLM backend module configurations.
	Modules []ModuleConfig `json:"modules,omitempty"`
}

// RadiusClusterStatus defines the observed state of RadiusCluster.
type RadiusClusterStatus struct {
	// ReadyReplicas is the number of ready FreeRADIUS pods.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// CurrentImage is the container image currently running in the managed Deployment.
	CurrentImage string `json:"currentImage,omitempty"`
	// PodRestarts is the total number of pod restarts observed.
	PodRestarts int32 `json:"podRestarts,omitempty"`
	// Conditions holds the status conditions for this resource.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RadiusCluster is the Schema for the radiusclusters API.
type RadiusCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RadiusClusterSpec   `json:"spec,omitempty"`
	Status RadiusClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RadiusClusterList contains a list of RadiusCluster.
type RadiusClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadiusCluster `json:"items"`
}
