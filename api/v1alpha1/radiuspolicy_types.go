// Package v1alpha1 contains API Schema definitions for the radius.operator.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MatchLeaf is a single RADIUS attribute condition.
type MatchLeaf struct {
	// Attribute is the RADIUS attribute name.
	Attribute string `json:"attribute"`
	// Operator is the comparison operator (e.g. ==, !=, >=).
	Operator string `json:"operator"`
	// Value is the value to compare against.
	Value string `json:"value"`
}

// PolicyMatch is a condition tree node supporting all/any/none combinators.
type PolicyMatch struct {
	// All requires all leaf conditions to match.
	All []MatchLeaf `json:"all,omitempty"`
	// Any requires at least one leaf condition to match.
	Any []MatchLeaf `json:"any,omitempty"`
	// None requires no leaf conditions to match.
	None []MatchLeaf `json:"none,omitempty"`
}

// PolicyAction is a single unlang action.
type PolicyAction struct {
	// Type is the action type. One of: set, call, reject, accept.
	// +kubebuilder:validation:Enum=set;call;reject;accept
	Type string `json:"type"`
	// Module is the module name to call (used with type=call).
	// +optional
	Module string `json:"module,omitempty"`
	// Attribute is the RADIUS attribute to set (used with type=set).
	// +optional
	Attribute string `json:"attribute,omitempty"`
	// Value is the value to assign to the attribute (used with type=set).
	// +optional
	Value string `json:"value,omitempty"`
}

// RadiusPolicySpec defines the desired state of RadiusPolicy.
type RadiusPolicySpec struct {
	// ClusterRef is the name of the owning RadiusCluster in the same namespace.
	ClusterRef string `json:"clusterRef"`
	// Stage is the FreeRADIUS processing stage this policy applies to.
	// +kubebuilder:validation:Enum=authorize;authenticate;preacct;accounting;post-auth;pre-proxy;post-proxy;session
	Stage string `json:"stage"`
	// Priority controls the order of policy evaluation within a stage; lower values are evaluated first.
	Priority int32 `json:"priority"`
	// Match is the condition tree for this policy.
	// +optional
	Match *PolicyMatch `json:"match,omitempty"`
	// Actions is the list of policy actions to execute when the match conditions are met.
	Actions []PolicyAction `json:"actions,omitempty"`
}

// RadiusPolicyStatus defines the observed state of RadiusPolicy.
type RadiusPolicyStatus struct {
	// Conditions holds the status conditions for this resource.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RadiusPolicy is the Schema for the radiuspolicies API.
type RadiusPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RadiusPolicySpec   `json:"spec,omitempty"`
	Status RadiusPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RadiusPolicyList contains a list of RadiusPolicy.
type RadiusPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RadiusPolicy `json:"items"`
}
