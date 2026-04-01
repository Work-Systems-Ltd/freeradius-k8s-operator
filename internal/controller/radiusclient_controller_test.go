package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestEnqueueOwningCluster_RadiusClient(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		clusterRef := rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "clusterRef")
		ns := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, "namespace")

		client := genRadiusClient(t, clusterRef)
		client.Namespace = ns

		requests := enqueueOwningCluster(context.TODO(), &client)

		assert.Len(t, requests, 1)
		assert.Equal(t, clusterRef, requests[0].Name)
		assert.Equal(t, ns, requests[0].Namespace)
	})
}
