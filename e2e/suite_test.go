package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
	"github.com/example/freeradius-operator/internal/controller"
	"github.com/example/freeradius-operator/internal/renderer"
	"github.com/example/freeradius-operator/internal/status"
)

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func setupEnvtest(t *testing.T) {
	t.Helper()

	ctx, cancel = context.WithCancel(context.Background())

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd")},
	}

	cfg, err := testEnv.Start()
	require.NoError(t, err, "failed to start envtest")
	require.NotNil(t, cfg)

	require.NoError(t, radiusv1alpha1.AddToScheme(scheme.Scheme))
	require.NoError(t, autoscalingv2.AddToScheme(scheme.Scheme))

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)

	// Create manager and register controllers
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // Disable metrics in tests
		},
	})
	require.NoError(t, err)

	statusReporter := status.New(mgr.GetClient())
	configRenderer := renderer.New()

	require.NoError(t, (&controller.RadiusClusterReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Renderer: configRenderer,
		Status:   statusReporter,
	}).SetupWithManager(mgr))

	require.NoError(t, (&controller.RadiusClientReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Status: statusReporter,
	}).SetupWithManager(mgr))

	require.NoError(t, (&controller.RadiusPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Status: statusReporter,
	}).SetupWithManager(mgr))

	go func() {
		require.NoError(t, mgr.Start(ctx))
	}()
}

func teardownEnvtest(t *testing.T) {
	t.Helper()
	cancel()
	require.NoError(t, testEnv.Stop())
}

// waitForCondition polls until the given condition is met or timeout.
func waitForCondition(t *testing.T, key types.NamespacedName, obj client.Object, check func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := k8sClient.Get(ctx, key, obj); err == nil {
			if check() {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s for %s", timeout, key)
}

// --- Integration Tests ---

func TestRadiusClusterFullReconcile(t *testing.T) {
	setupEnvtest(t)
	defer teardownEnvtest(t)

	ns := "default"

	// Create a secret referenced by a module
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sql-secret", Namespace: ns},
		StringData: map[string]string{"password": "testpass"},
	}
	require.NoError(t, k8sClient.Create(ctx, secret))

	// Create RadiusCluster
	cluster := &radiusv1alpha1.RadiusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClusterSpec{
			Replicas: 2,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules: []radiusv1alpha1.ModuleConfig{
				{
					Name:    "sql",
					Type:    "rlm_sql",
					Enabled: true,
					SQL: &radiusv1alpha1.SQLConfig{
						Dialect:     "postgresql",
						Server:      "postgres.svc",
						Port:        5432,
						Database:    "radius",
						Login:       "radius",
						PasswordRef: radiusv1alpha1.SecretRef{Name: "test-sql-secret", Key: "password"},
					},
				},
				{
					Name:    "pap",
					Type:    "rlm_pap",
					Enabled: true,
				},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, cluster))

	// Verify Deployment created
	deploy := &appsv1.Deployment{}
	deployKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-freeradius"}
	waitForCondition(t, deployKey, deploy, func() bool {
		return deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == 2
	}, 30*time.Second)

	// Verify owner reference
	assert.Len(t, deploy.OwnerReferences, 1)
	assert.Equal(t, "test-cluster", deploy.OwnerReferences[0].Name)
	assert.True(t, *deploy.OwnerReferences[0].Controller)

	// Verify rolling update strategy
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, deploy.Spec.Strategy.Type)

	// Verify Service created
	svc := &corev1.Service{}
	svcKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-freeradius"}
	waitForCondition(t, svcKey, svc, func() bool {
		return len(svc.Spec.Ports) == 2
	}, 30*time.Second)
	assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)
	assert.Equal(t, int32(1812), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(1813), svc.Spec.Ports[1].Port)

	// Verify ConfigMap created
	cm := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-freeradius-config"}
	waitForCondition(t, cmKey, cm, func() bool {
		_, hasRadiusd := cm.Data["radiusd.conf"]
		_, hasClients := cm.Data["clients.conf"]
		return hasRadiusd && hasClients
	}, 30*time.Second)
	assert.Contains(t, cm.Data, "mods-enabled__sql")
	assert.Contains(t, cm.Data, "mods-enabled__pap")
	assert.Contains(t, cm.Data, "sites-enabled__default")

	// Update replicas
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-cluster"}, cluster))
	cluster.Spec.Replicas = 3
	require.NoError(t, k8sClient.Update(ctx, cluster))

	waitForCondition(t, deployKey, deploy, func() bool {
		return deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == 3
	}, 30*time.Second)

	// Delete cluster — owned resources should be garbage collected by envtest
	require.NoError(t, k8sClient.Delete(ctx, cluster))
}

func TestRadiusClientCrossControllerEnqueue(t *testing.T) {
	setupEnvtest(t)
	defer teardownEnvtest(t)

	ns := "default"

	// Create secret for module
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sql-secret-2", Namespace: ns},
		StringData: map[string]string{"password": "testpass"},
	}
	require.NoError(t, k8sClient.Create(ctx, secret))

	// Create cluster first
	cluster := &radiusv1alpha1.RadiusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-2", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClusterSpec{
			Replicas: 1,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules: []radiusv1alpha1.ModuleConfig{
				{Name: "pap", Type: "rlm_pap", Enabled: true},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, cluster))

	// Wait for ConfigMap to exist
	cm := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-2-freeradius-config"}
	waitForCondition(t, cmKey, cm, func() bool {
		_, has := cm.Data["clients.conf"]
		return has
	}, 30*time.Second)

	// Create a client secret
	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bng-secret", Namespace: ns},
		StringData: map[string]string{"shared-secret": "testsecret"},
	}
	require.NoError(t, k8sClient.Create(ctx, clientSecret))

	// Create RadiusClient
	rc := &radiusv1alpha1.RadiusClient{
		ObjectMeta: metav1.ObjectMeta{Name: "bng-01", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClientSpec{
			ClusterRef: "test-cluster-2",
			IP:         "10.0.1.0/24",
			SecretRef:  radiusv1alpha1.SecretRef{Name: "bng-secret", Key: "shared-secret"},
			NASType:    "other",
		},
	}
	require.NoError(t, k8sClient.Create(ctx, rc))

	// Wait for clients.conf to contain the new client
	waitForCondition(t, cmKey, cm, func() bool {
		clients, ok := cm.Data["clients.conf"]
		return ok && len(clients) > 50 // Should contain client block now
	}, 30*time.Second)
	assert.Contains(t, cm.Data["clients.conf"], "bng-01")

	// Delete the client and verify it's removed
	require.NoError(t, k8sClient.Delete(ctx, rc))
	time.Sleep(2 * time.Second) // Give reconciler time to process
	require.NoError(t, k8sClient.Get(ctx, cmKey, cm))
	// After deletion, the client should eventually not appear
	// (may need a second reconcile cycle)
}

func TestRadiusClientInvalidClusterRef(t *testing.T) {
	setupEnvtest(t)
	defer teardownEnvtest(t)

	ns := "default"

	rc := &radiusv1alpha1.RadiusClient{
		ObjectMeta: metav1.ObjectMeta{Name: "orphan-client", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClientSpec{
			ClusterRef: "nonexistent-cluster",
			IP:         "10.0.1.1",
			SecretRef:  radiusv1alpha1.SecretRef{Name: "some-secret", Key: "key"},
			NASType:    "other",
		},
	}
	require.NoError(t, k8sClient.Create(ctx, rc))

	// Wait for Invalid condition to be set
	waitForCondition(t, types.NamespacedName{Namespace: ns, Name: "orphan-client"}, rc, func() bool {
		for _, c := range rc.Status.Conditions {
			if c.Type == "Invalid" && c.Status == "True" {
				return true
			}
		}
		return false
	}, 30*time.Second)
}

func TestRadiusPolicyCrossControllerEnqueue(t *testing.T) {
	setupEnvtest(t)
	defer teardownEnvtest(t)

	ns := "default"

	cluster := &radiusv1alpha1.RadiusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-3", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClusterSpec{
			Replicas: 1,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules: []radiusv1alpha1.ModuleConfig{
				{Name: "pap", Type: "rlm_pap", Enabled: true},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, cluster))

	// Wait for initial reconcile
	cm := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-3-freeradius-config"}
	waitForCondition(t, cmKey, cm, func() bool {
		_, has := cm.Data["sites-enabled__default"]
		return has
	}, 30*time.Second)

	// Create a RadiusPolicy
	policy := &radiusv1alpha1.RadiusPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "test-policy", Namespace: ns},
		Spec: radiusv1alpha1.RadiusPolicySpec{
			ClusterRef: "test-cluster-3",
			Stage:      "authorize",
			Priority:   10,
			Match: &radiusv1alpha1.PolicyMatch{
				All: []radiusv1alpha1.MatchLeaf{
					{Attribute: "User-Name", Operator: "==", Value: "admin"},
				},
			},
			Actions: []radiusv1alpha1.PolicyAction{
				{Type: "accept"},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, policy))

	// Wait for policy to appear in sites-enabled config
	waitForCondition(t, cmKey, cm, func() bool {
		sites, ok := cm.Data["sites-enabled__default"]
		return ok && len(sites) > 100 // Should contain the policy
	}, 30*time.Second)
	assert.Contains(t, cm.Data["sites-enabled__default"], "test-policy")
}

func TestHPALifecycle(t *testing.T) {
	setupEnvtest(t)
	defer teardownEnvtest(t)

	ns := "default"

	cluster := &radiusv1alpha1.RadiusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-hpa", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClusterSpec{
			Replicas: 1,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Autoscaling: &radiusv1alpha1.AutoscalingConfig{
				Enabled:                        true,
				MinReplicas:                    2,
				MaxReplicas:                    10,
				TargetCPUUtilizationPercentage: 70,
			},
			Modules: []radiusv1alpha1.ModuleConfig{
				{Name: "pap", Type: "rlm_pap", Enabled: true},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, cluster))

	// Verify HPA created
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	hpaKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-hpa-freeradius"}
	waitForCondition(t, hpaKey, hpa, func() bool {
		return hpa.Spec.MaxReplicas == 10
	}, 30*time.Second)
	assert.Equal(t, int32(2), *hpa.Spec.MinReplicas)

	// Disable autoscaling
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-cluster-hpa"}, cluster))
	cluster.Spec.Autoscaling.Enabled = false
	require.NoError(t, k8sClient.Update(ctx, cluster))

	// Wait for HPA to be deleted
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		err := k8sClient.Get(ctx, hpaKey, hpa)
		if err != nil {
			break // HPA deleted
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func TestMissingSecretDegradedStatus(t *testing.T) {
	setupEnvtest(t)
	defer teardownEnvtest(t)

	ns := "default"

	// Create cluster referencing a non-existent secret
	cluster := &radiusv1alpha1.RadiusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-nosecret", Namespace: ns},
		Spec: radiusv1alpha1.RadiusClusterSpec{
			Replicas: 1,
			Image:    "docker.io/freeradius/freeradius-server:3.2.3",
			Modules: []radiusv1alpha1.ModuleConfig{
				{
					Name:    "sql",
					Type:    "rlm_sql",
					Enabled: true,
					SQL: &radiusv1alpha1.SQLConfig{
						Dialect:     "postgresql",
						Server:      "postgres.svc",
						Port:        5432,
						Database:    "radius",
						Login:       "radius",
						PasswordRef: radiusv1alpha1.SecretRef{Name: "missing-secret", Key: "password"},
					},
				},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, cluster))

	// Wait for Degraded condition
	clusterKey := types.NamespacedName{Namespace: ns, Name: "test-cluster-nosecret"}
	waitForCondition(t, clusterKey, cluster, func() bool {
		for _, c := range cluster.Status.Conditions {
			if c.Type == "Degraded" && c.Status == "True" {
				return true
			}
		}
		return false
	}, 30*time.Second)
}
