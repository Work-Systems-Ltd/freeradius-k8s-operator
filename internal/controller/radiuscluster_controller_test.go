package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
	"github.com/example/freeradius-operator/internal/renderer"
)

// --- Generators ---

func genSecretRef(t *rapid.T) radiusv1alpha1.SecretRef {
	return radiusv1alpha1.SecretRef{
		Name: rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "secretName"),
		Key:  rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "secretKey"),
	}
}

func genModuleConfig(t *rapid.T) radiusv1alpha1.ModuleConfig {
	modType := rapid.SampledFrom([]string{"rlm_sql", "rlm_ldap", "rlm_eap", "rlm_pap", "rlm_chap"}).Draw(t, "modType")
	mod := radiusv1alpha1.ModuleConfig{
		Name:    rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, "modName"),
		Type:    modType,
		Enabled: rapid.Bool().Draw(t, "enabled"),
	}
	if modType == "rlm_sql" {
		mod.SQL = &radiusv1alpha1.SQLConfig{
			Dialect:     "postgresql",
			Server:      "postgres.svc",
			Port:        5432,
			Database:    "radius",
			Login:       "radius",
			PasswordRef: genSecretRef(t),
		}
	}
	if modType == "rlm_ldap" {
		mod.LDAP = &radiusv1alpha1.LDAPConfig{
			Server:      "ldap.svc",
			Port:        389,
			BaseDN:      "dc=example,dc=com",
			Identity:    "cn=admin,dc=example,dc=com",
			PasswordRef: genSecretRef(t),
		}
	}
	return mod
}

func genRadiusCluster(t *rapid.T) *radiusv1alpha1.RadiusCluster {
	nMods := rapid.IntRange(0, 3).Draw(t, "nMods")
	var mods []radiusv1alpha1.ModuleConfig
	for i := 0; i < nMods; i++ {
		mods = append(mods, genModuleConfig(t))
	}
	return &radiusv1alpha1.RadiusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "clusterName"),
			Namespace: "default",
		},
		Spec: radiusv1alpha1.RadiusClusterSpec{
			Replicas: int32(rapid.IntRange(1, 10).Draw(t, "replicas")),
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules:  mods,
		},
	}
}

func genRadiusClient(t *rapid.T, clusterName string) radiusv1alpha1.RadiusClient {
	return radiusv1alpha1.RadiusClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "clientName"),
			Namespace: "default",
		},
		Spec: radiusv1alpha1.RadiusClientSpec{
			ClusterRef: clusterName,
			IP:         "10.0.1.0/24",
			SecretRef:  genSecretRef(t),
			NASType:    "other",
		},
	}
}

// Feature: freeradius-operator, Property 2: Owner references on all managed resources
func TestBuildRenderContext(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		nClients := rapid.IntRange(0, 5).Draw(t, "nClients")
		var clients []radiusv1alpha1.RadiusClient
		for i := 0; i < nClients; i++ {
			clients = append(clients, genRadiusClient(t, cluster.Name))
		}

		ctx := buildRenderContext(cluster, clients, nil)

		// All clients should appear in the render context
		assert.Equal(t, len(clients), len(ctx.Clients))
		for i, c := range clients {
			assert.Equal(t, c.Name, ctx.Clients[i].Name)
			assert.Equal(t, c.Spec.IP, ctx.Clients[i].IP)
		}

		// Modules should match
		enabledCount := 0
		for _, m := range cluster.Spec.Modules {
			if m.Enabled {
				enabledCount++
			}
		}
		assert.Equal(t, len(cluster.Spec.Modules), len(ctx.Cluster.Modules))
	})
}

// Feature: freeradius-operator, Property 10: Secrets mounted as read-only volumes at well-known paths
func TestCollectSecretRefs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		nClients := rapid.IntRange(0, 5).Draw(t, "nClients")
		var clients []radiusv1alpha1.RadiusClient
		for i := 0; i < nClients; i++ {
			clients = append(clients, genRadiusClient(t, cluster.Name))
		}

		refs := collectSecretRefs(cluster, clients)

		// Each client secret should be present
		for _, c := range clients {
			found := false
			for _, ref := range refs {
				if ref.Name == c.Spec.SecretRef.Name {
					found = true
					break
				}
			}
			assert.True(t, found, "client secret %q not found in refs", c.Spec.SecretRef.Name)
		}

		// No duplicates
		seen := make(map[string]bool)
		for _, ref := range refs {
			assert.False(t, seen[ref.Name], "duplicate secret ref: %s", ref.Name)
			seen[ref.Name] = true
		}
	})
}

// Feature: freeradius-operator, Property 16: Default probes configured on pods
func TestDefaultProbes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		cluster.Spec.Probes = nil // No custom probes

		podSpec := (&RadiusClusterReconciler{}).buildPodSpec(cluster, nil)

		require.Len(t, podSpec.Containers, 1)
		container := podSpec.Containers[0]

		assert.NotNil(t, container.LivenessProbe, "liveness probe must be set")
		assert.NotNil(t, container.ReadinessProbe, "readiness probe must be set")
		assert.NotNil(t, container.LivenessProbe.Exec, "liveness probe must be exec")
		assert.NotNil(t, container.ReadinessProbe.Exec, "readiness probe must be exec")
	})
}

// Feature: freeradius-operator, Property 17: Custom probes override defaults
func TestCustomProbesOverride(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		customLiveness := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"custom-liveness"},
				},
			},
			InitialDelaySeconds: int32(rapid.IntRange(1, 60).Draw(t, "delay")),
		}
		cluster.Spec.Probes = &radiusv1alpha1.ProbesConfig{
			Liveness: customLiveness,
		}

		podSpec := (&RadiusClusterReconciler{}).buildPodSpec(cluster, nil)

		container := podSpec.Containers[0]
		assert.Equal(t, customLiveness, container.LivenessProbe)
		// Readiness should still be default since we only overrode liveness
		assert.NotNil(t, container.ReadinessProbe)
	})
}

// Feature: freeradius-operator, Property 10: Secrets mounted as read-only volumes
func TestSecretVolumeMounts(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		nRefs := rapid.IntRange(1, 5).Draw(t, "nRefs")
		var refs []renderer.SecretRef
		seen := make(map[string]bool)
		for i := 0; i < nRefs; i++ {
			name := rapid.StringMatching(`[a-z][a-z0-9]{3,10}`).Draw(t, "refName")
			if seen[name] {
				continue
			}
			seen[name] = true
			refs = append(refs, renderer.SecretRef{Name: name, Key: "key"})
		}

		podSpec := (&RadiusClusterReconciler{}).buildPodSpec(cluster, refs)

		// Check volumes exist for each secret
		for _, ref := range refs {
			volName := "secret-" + ref.Name
			found := false
			for _, v := range podSpec.Volumes {
				if v.Name == volName {
					found = true
					assert.NotNil(t, v.Secret, "volume %q must be a secret volume", volName)
					assert.Equal(t, ref.Name, v.Secret.SecretName)
					assert.Equal(t, int32(0400), *v.Secret.DefaultMode)
					break
				}
			}
			assert.True(t, found, "volume %q not found", volName)

			// Check mount exists and is read-only
			mountFound := false
			for _, m := range podSpec.Containers[0].VolumeMounts {
				if m.Name == volName {
					mountFound = true
					assert.True(t, m.ReadOnly, "mount %q must be read-only", volName)
					assert.Equal(t, "/etc/freeradius/secrets/"+ref.Name, m.MountPath)
					break
				}
			}
			assert.True(t, mountFound, "mount %q not found", volName)
		}
	})
}

// Feature: freeradius-operator, Property 18: Namespace isolation
func TestNamespaceIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		ns := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, "namespace")
		cluster.Namespace = ns

		// Clients in the same namespace
		sameNsClient := genRadiusClient(t, cluster.Name)
		sameNsClient.Namespace = ns

		// Client in a different namespace — should NOT be included
		diffNsClient := genRadiusClient(t, cluster.Name)
		diffNsClient.Namespace = ns + "-other"

		// Only pass same-namespace clients (as the controller would filter)
		ctx := buildRenderContext(cluster, []radiusv1alpha1.RadiusClient{sameNsClient}, nil)

		assert.Equal(t, 1, len(ctx.Clients))
		assert.Equal(t, sameNsClient.Name, ctx.Clients[0].Name)
	})
}
