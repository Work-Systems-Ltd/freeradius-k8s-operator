package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type MatchLeaf struct {
	Attribute string `json:"attribute"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
}

type PolicyMatch struct {
	All  []MatchLeaf `json:"all,omitempty"`
	Any  []MatchLeaf `json:"any,omitempty"`
	None []MatchLeaf `json:"none,omitempty"`
}

type PolicyAction struct {
	// +kubebuilder:validation:Enum=set;call;reject;accept;redundant;load-balance
	Type      string   `json:"type"`
	Module    string   `json:"module,omitempty"`
	Modules   []string `json:"modules,omitempty"`
	Attribute string   `json:"attribute,omitempty"`
	Value     string   `json:"value,omitempty"`
}

type RadiusPolicySpec struct {
	ClusterRef string `json:"clusterRef"`
	// +kubebuilder:validation:Enum=authorize;authenticate;preacct;accounting;post-auth;pre-proxy;post-proxy;session
	Stage     string         `json:"stage"`
	Priority  int32          `json:"priority"`
	Match     *PolicyMatch   `json:"match,omitempty"`
	Actions   []PolicyAction `json:"actions,omitempty"`
	RawConfig string         `json:"rawConfig,omitempty"`
}

type RadiusPolicyStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type RadiusPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RadiusPolicySpec   `json:"spec,omitempty"`
	Status            RadiusPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type RadiusPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadiusPolicy `json:"items"`
}
