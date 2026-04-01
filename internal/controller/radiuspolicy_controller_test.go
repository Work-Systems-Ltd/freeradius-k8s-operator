package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
)

func TestValidStagesSet(t *testing.T) {
	for _, s := range []string{"authorize", "authenticate", "preacct", "accounting", "post-auth", "pre-proxy", "post-proxy", "session"} {
		assert.True(t, validStages[s], "stage %q should be valid", s)
	}
	assert.False(t, validStages["bogus"])
}

func TestValidActionTypesSet(t *testing.T) {
	for _, a := range []string{"set", "call", "reject", "accept"} {
		assert.True(t, validActionTypes[a], "action %q should be valid", a)
	}
	assert.False(t, validActionTypes["bogus"])
}

func TestEnqueueOwningCluster_RadiusPolicy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		clusterRef := rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "clusterRef")
		ns := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, "namespace")

		policy := &radiusv1alpha1.RadiusPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "policyName"),
				Namespace: ns,
			},
			Spec: radiusv1alpha1.RadiusPolicySpec{ClusterRef: clusterRef, Stage: "authorize", Priority: 10},
		}

		requests := enqueueOwningCluster(nil, policy)
		assert.Len(t, requests, 1)
		assert.Equal(t, clusterRef, requests[0].Name)
		assert.Equal(t, ns, requests[0].Namespace)
	})
}
