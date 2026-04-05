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

func defaultContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: "auth", ContainerPort: 1812, Protocol: corev1.ProtocolUDP},
		{Name: "acct", ContainerPort: 1813, Protocol: corev1.ProtocolUDP},
	}
}

func genSecretRef(t *rapid.T) radiusv1alpha1.SecretRef {
	return radiusv1alpha1.SecretRef{
		Name: rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "secretName"),
		Key:  rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "secretKey"),
	}
}

func genModuleConfig(t *rapid.T) radiusv1alpha1.ModuleConfig {
	modType := rapid.SampledFrom([]string{"rlm_sql", "rlm_ldap", "rlm_eap", "rlm_pap", "rlm_chap"}).Draw(t, "modType")
	mod := radiusv1alpha1.ModuleConfig{
		Name: rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, "modName"),
		Type: modType, Enabled: rapid.Bool().Draw(t, "enabled"),
	}
	if modType == "rlm_sql" {
		mod.SQL = &radiusv1alpha1.SQLConfig{
			Dialect: "postgresql", Server: "postgres.svc", Port: 5432,
			Database: "radius", Login: "radius", PasswordRef: genSecretRef(t),
		}
	}
	if modType == "rlm_ldap" {
		mod.LDAP = &radiusv1alpha1.LDAPConfig{
			Server: "ldap.svc", Port: 389, BaseDN: "dc=example,dc=com",
			Identity: "cn=admin,dc=example,dc=com", PasswordRef: genSecretRef(t),
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
			Name: rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "clusterName"), Namespace: "default",
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
			Name: rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "clientName"), Namespace: "default",
		},
		Spec: radiusv1alpha1.RadiusClientSpec{
			ClusterRef: clusterName, IP: "10.0.1.0/24", SecretRef: genSecretRef(t), NASType: "other",
		},
	}
}

func TestBuildRenderContext(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		nClients := rapid.IntRange(0, 5).Draw(t, "nClients")
		var clients []radiusv1alpha1.RadiusClient
		for i := 0; i < nClients; i++ {
			clients = append(clients, genRadiusClient(t, cluster.Name))
		}
		ctx := buildRenderContext(cluster, clients, nil)
		assert.Equal(t, len(clients), len(ctx.Clients))
		assert.Equal(t, len(cluster.Spec.Modules), len(ctx.Cluster.Modules))
	})
}

func TestCollectSecretRefs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		nClients := rapid.IntRange(0, 5).Draw(t, "nClients")
		var clients []radiusv1alpha1.RadiusClient
		for i := 0; i < nClients; i++ {
			clients = append(clients, genRadiusClient(t, cluster.Name))
		}
		refs := collectSecretRefs(cluster, clients)
		for _, c := range clients {
			found := false
			for _, ref := range refs {
				if ref.Name == c.Spec.SecretRef.Name {
					found = true
					break
				}
			}
			assert.True(t, found, "client secret %q not found", c.Spec.SecretRef.Name)
		}
		seen := make(map[string]bool)
		for _, ref := range refs {
			assert.False(t, seen[ref.Name], "duplicate: %s", ref.Name)
			seen[ref.Name] = true
		}
	})
}

func TestDefaultProbes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		cluster.Spec.Probes = nil
		podSpec := (&RadiusClusterReconciler{}).buildPodSpec(cluster, nil, defaultContainerPorts())
		require.Len(t, podSpec.Containers, 1)
		c := podSpec.Containers[0]
		assert.NotNil(t, c.LivenessProbe)
		assert.NotNil(t, c.ReadinessProbe)
	})
}

func TestCustomProbesOverride(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		custom := &corev1.Probe{
			ProbeHandler:        corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: []string{"custom-liveness"}}},
			InitialDelaySeconds: int32(rapid.IntRange(1, 60).Draw(t, "delay")),
		}
		cluster.Spec.Probes = &radiusv1alpha1.ProbesConfig{Liveness: custom}
		podSpec := (&RadiusClusterReconciler{}).buildPodSpec(cluster, nil, defaultContainerPorts())
		assert.Equal(t, custom, podSpec.Containers[0].LivenessProbe)
		assert.NotNil(t, podSpec.Containers[0].ReadinessProbe)
	})
}

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
		podSpec := (&RadiusClusterReconciler{}).buildPodSpec(cluster, refs, defaultContainerPorts())
		for _, ref := range refs {
			volName := "secret-" + ref.Name
			var volFound, mountFound bool
			for _, v := range podSpec.Volumes {
				if v.Name == volName {
					volFound = true
					assert.Equal(t, ref.Name, v.Secret.SecretName)
					assert.Equal(t, int32(0440), *v.Secret.DefaultMode)
				}
			}
			for _, m := range podSpec.Containers[0].VolumeMounts {
				if m.Name == volName {
					mountFound = true
					assert.True(t, m.ReadOnly)
					assert.Equal(t, "/etc/freeradius/secrets/"+ref.Name, m.MountPath)
				}
			}
			assert.True(t, volFound, "volume %q not found", volName)
			assert.True(t, mountFound, "mount %q not found", volName)
		}
	})
}

func TestNamespaceIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cluster := genRadiusCluster(t)
		ns := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, "namespace")
		cluster.Namespace = ns
		sameNs := genRadiusClient(t, cluster.Name)
		sameNs.Namespace = ns
		ctx := buildRenderContext(cluster, []radiusv1alpha1.RadiusClient{sameNs}, nil)
		assert.Equal(t, 1, len(ctx.Clients))
		assert.Equal(t, sameNs.Name, ctx.Clients[0].Name)
	})
}
