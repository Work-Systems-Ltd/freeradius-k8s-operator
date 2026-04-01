// Package controller implements the Kubernetes controllers for the freeradius-operator.
package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
	"github.com/example/freeradius-operator/internal/metrics"
	"github.com/example/freeradius-operator/internal/renderer"
	"github.com/example/freeradius-operator/internal/status"
)

const (
	configMapSuffix  = "-freeradius-config"
	deploymentSuffix = "-freeradius"
	serviceSuffix    = "-freeradius"
	hpaSuffix        = "-freeradius"
)

// RadiusClusterReconciler reconciles a RadiusCluster object.
type RadiusClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Renderer renderer.ConfigRenderer
	Status   *status.StatusReporter
}

// +kubebuilder:rbac:groups=radius.operator.io,resources=radiusclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=radius.operator.io,resources=radiusclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=radius.operator.io,resources=radiusclients,verbs=get;list;watch
// +kubebuilder:rbac:groups=radius.operator.io,resources=radiuspolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

func (r *RadiusClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	start := time.Now()
	result := "success"

	defer func() {
		duration := time.Since(start).Seconds()
		metrics.ReconcileTotal.WithLabelValues(req.Namespace, req.Name, "RadiusCluster", result).Inc()
		metrics.ReconcileDuration.WithLabelValues(req.Namespace, req.Name, "RadiusCluster").Observe(duration)
	}()

	// Fetch the RadiusCluster
	cluster := &radiusv1alpha1.RadiusCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		result = "error"
		return ctrl.Result{}, err
	}

	// Set Progressing
	if err := r.Status.SetProgressing(ctx, cluster, true); err != nil {
		logger.Error(err, "unable to set Progressing status")
	}
	// Re-fetch after status update to avoid conflicts
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		result = "error"
		return ctrl.Result{}, err
	}

	// List RadiusClients for this cluster
	clientList := &radiusv1alpha1.RadiusClientList{}
	if err := r.List(ctx, clientList, client.InNamespace(req.Namespace)); err != nil {
		result = "error"
		return ctrl.Result{}, fmt.Errorf("listing RadiusClients: %w", err)
	}
	var matchingClients []radiusv1alpha1.RadiusClient
	for _, c := range clientList.Items {
		if c.Spec.ClusterRef == cluster.Name {
			matchingClients = append(matchingClients, c)
		}
	}

	// List RadiusPolicies for this cluster
	policyList := &radiusv1alpha1.RadiusPolicyList{}
	if err := r.List(ctx, policyList, client.InNamespace(req.Namespace)); err != nil {
		result = "error"
		return ctrl.Result{}, fmt.Errorf("listing RadiusPolicies: %w", err)
	}
	var matchingPolicies []radiusv1alpha1.RadiusPolicy
	for _, p := range policyList.Items {
		if p.Spec.ClusterRef == cluster.Name {
			matchingPolicies = append(matchingPolicies, p)
		}
	}

	// Resolve secrets — check they exist
	secretRefs := collectSecretRefs(cluster, matchingClients)
	for _, ref := range secretRefs {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: ref.Name}, secret); err != nil {
			if errors.IsNotFound(err) {
				logger.Error(err, "referenced Secret not found", "secret", ref.Name)
				if statusErr := r.Status.SetDegraded(ctx, cluster, true, "MissingSecret",
					fmt.Sprintf("Secret %q not found", ref.Name)); statusErr != nil {
					logger.Error(statusErr, "unable to set Degraded status")
				}
				result = "error"
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			result = "error"
			return ctrl.Result{}, fmt.Errorf("fetching Secret %q: %w", ref.Name, err)
		}
	}

	// Build RenderContext
	renderCtx := buildRenderContext(cluster, matchingClients, matchingPolicies)

	// Render config
	configFiles, err := r.Renderer.Render(renderCtx)
	if err != nil {
		if isInvalidError(err) {
			logger.Error(err, "invalid spec")
			if statusErr := r.Status.SetDegraded(ctx, cluster, true, "InvalidSpec", err.Error()); statusErr != nil {
				logger.Error(statusErr, "unable to set Degraded status")
			}
			result = "error"
			return ctrl.Result{}, nil // Don't requeue for invalid specs
		}
		result = "error"
		return ctrl.Result{}, fmt.Errorf("rendering config: %w", err)
	}

	// CreateOrUpdate ConfigMap
	if err := r.reconcileConfigMap(ctx, cluster, configFiles); err != nil {
		result = "error"
		return ctrl.Result{}, fmt.Errorf("reconciling ConfigMap: %w", err)
	}

	// CreateOrUpdate Deployment
	if err := r.reconcileDeployment(ctx, cluster, secretRefs); err != nil {
		result = "error"
		return ctrl.Result{}, fmt.Errorf("reconciling Deployment: %w", err)
	}

	// CreateOrUpdate Service
	if err := r.reconcileService(ctx, cluster); err != nil {
		result = "error"
		return ctrl.Result{}, fmt.Errorf("reconciling Service: %w", err)
	}

	// Manage HPA
	if err := r.reconcileHPA(ctx, cluster); err != nil {
		result = "error"
		return ctrl.Result{}, fmt.Errorf("reconciling HPA: %w", err)
	}

	// Update status
	deploy := &appsv1.Deployment{}
	deployName := types.NamespacedName{Namespace: req.Namespace, Name: cluster.Name + deploymentSuffix}
	if err := r.Get(ctx, deployName, deploy); err == nil {
		// Re-fetch cluster before status update
		if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
			result = "error"
			return ctrl.Result{}, err
		}
		podRestarts := r.countPodRestarts(ctx, req.Namespace, cluster.Name)
		if err := r.Status.UpdateClusterStatus(ctx, cluster, deploy.Status.ReadyReplicas, cluster.Spec.Image, podRestarts); err != nil {
			logger.Error(err, "unable to update cluster status fields")
		}
	}

	// Batch all final status condition updates into a single write.
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		result = "error"
		return ctrl.Result{}, err
	}
	r.Status.SetConditionLocal(&cluster.Status.Conditions, "Degraded", false, "AllSecretsPresent", "All referenced secrets are available")
	r.Status.SetConditionLocal(&cluster.Status.Conditions, "Available", true, "ReconcileComplete", "All resources reconciled successfully")
	r.Status.SetConditionLocal(&cluster.Status.Conditions, "Progressing", false, "ReconcileComplete", "Reconciliation completed successfully")
	if statusErr := r.Status.FlushClusterStatus(ctx, cluster); statusErr != nil {
		logger.Error(statusErr, "unable to update final status conditions")
	}

	logger.Info("reconciliation complete")
	return ctrl.Result{}, nil
}

func (r *RadiusClusterReconciler) reconcileConfigMap(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, files renderer.ConfigFiles) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + configMapSuffix,
			Namespace: cluster.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if err := ctrl.SetControllerReference(cluster, cm, r.Scheme); err != nil {
			return err
		}
		cm.Data = make(map[string]string, len(files))
		for k, v := range files {
			// Replace / with __ for ConfigMap key compatibility
			key := strings.ReplaceAll(k, "/", "__")
			cm.Data[key] = v
		}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) reconcileDeployment(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, secretRefs []renderer.SecretRef) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + deploymentSuffix,
			Namespace: cluster.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		if err := ctrl.SetControllerReference(cluster, deploy, r.Scheme); err != nil {
			return err
		}

		labels := map[string]string{
			"app.kubernetes.io/name":       "freeradius",
			"app.kubernetes.io/instance":   cluster.Name,
			"app.kubernetes.io/managed-by": "freeradius-operator",
		}

		maxUnavailable := intstr.FromInt(0)
		maxSurge := intstr.FromInt(1)

		deploy.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: r.buildPodSpec(cluster, secretRefs),
			},
		}

		// Only set replicas when autoscaling is not enabled,
		// to avoid fighting with the HPA controller.
		if cluster.Spec.Autoscaling == nil || !cluster.Spec.Autoscaling.Enabled {
			replicas := cluster.Spec.Replicas
			deploy.Spec.Replicas = &replicas
		}

		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) buildPodSpec(cluster *radiusv1alpha1.RadiusCluster, secretRefs []renderer.SecretRef) corev1.PodSpec {
	configMapName := cluster.Name + configMapSuffix

	// Volumes
	volumes := []corev1.Volume{
		{
			Name: "freeradius-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				},
			},
		},
		{
			Name: "freeradius-config-rendered",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	// Add secret volumes
	for _, ref := range secretRefs {
		volName := "secret-" + ref.Name
		mode := int32(0400)
		volumes = append(volumes, corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  ref.Name,
					DefaultMode: &mode,
				},
			},
		})
	}

	// Init container to reconstruct directory structure from __ separated keys
	initVolumeMounts := []corev1.VolumeMount{
		{Name: "freeradius-config", MountPath: "/config-flat", ReadOnly: true},
		{Name: "freeradius-config-rendered", MountPath: "/etc/freeradius"},
	}

	allowPrivEsc := false
	initContainer := corev1.Container{
		Name:  "config-init",
		Image: "docker.io/library/busybox:1.36",
		Command: []string{"sh", "-c", `
cd /config-flat
for f in *; do
  target=$(echo "$f" | sed 's/__/\//g')
  mkdir -p "/etc/freeradius/$(dirname "$target")"
  cp "$f" "/etc/freeradius/$target"
done
`},
		VolumeMounts: initVolumeMounts,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowPrivEsc,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("16Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
		},
	}

	// Main container volume mounts
	mainVolumeMounts := []corev1.VolumeMount{
		{Name: "freeradius-config-rendered", MountPath: "/etc/freeradius", ReadOnly: true},
	}
	for _, ref := range secretRefs {
		mainVolumeMounts = append(mainVolumeMounts, corev1.VolumeMount{
			Name:      "secret-" + ref.Name,
			MountPath: "/etc/freeradius/secrets/" + ref.Name,
			ReadOnly:  true,
		})
	}

	// Probes
	liveness, readiness := defaultProbes()
	if cluster.Spec.Probes != nil {
		if cluster.Spec.Probes.Liveness != nil {
			liveness = cluster.Spec.Probes.Liveness
		}
		if cluster.Spec.Probes.Readiness != nil {
			readiness = cluster.Spec.Probes.Readiness
		}
	}

	mainContainer := corev1.Container{
		Name:            "freeradius",
		Image:           cluster.Spec.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{Name: "auth", ContainerPort: 1812, Protocol: corev1.ProtocolUDP},
			{Name: "acct", ContainerPort: 1813, Protocol: corev1.ProtocolUDP},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowPrivEsc,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		VolumeMounts:   mainVolumeMounts,
		Resources:      cluster.Spec.Resources,
		LivenessProbe:  liveness,
		ReadinessProbe: readiness,
	}

	runAsNonRoot := true
	runAsUser := int64(65534) // nobody
	runAsGroup := int64(65534)

	return corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
			RunAsGroup:   &runAsGroup,
		},
		InitContainers: []corev1.Container{initContainer},
		Containers:     []corev1.Container{mainContainer},
		Volumes:        volumes,
	}
}

func defaultProbes() (*corev1.Probe, *corev1.Probe) {
	liveness := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"radiusd", "-C"},
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
	}
	readiness := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"sh", "-c", "echo 'Message-Authenticator = 0x00' | radclient -x 127.0.0.1:1812 status testing123"},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
	}
	return liveness, readiness
}

func (r *RadiusClusterReconciler) reconcileService(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + serviceSuffix,
			Namespace: cluster.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(cluster, svc, r.Scheme); err != nil {
			return err
		}

		labels := map[string]string{
			"app.kubernetes.io/name":       "freeradius",
			"app.kubernetes.io/instance":   cluster.Name,
			"app.kubernetes.io/managed-by": "freeradius-operator",
		}

		svc.Spec = corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Name: "auth", Port: 1812, TargetPort: intstr.FromInt(1812), Protocol: corev1.ProtocolUDP},
				{Name: "acct", Port: 1813, TargetPort: intstr.FromInt(1813), Protocol: corev1.ProtocolUDP},
			},
		}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) reconcileHPA(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster) error {
	hpaName := types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name + hpaSuffix}

	if cluster.Spec.Autoscaling == nil || !cluster.Spec.Autoscaling.Enabled {
		// Delete HPA if it exists
		existing := &autoscalingv2.HorizontalPodAutoscaler{}
		if err := r.Get(ctx, hpaName, existing); err == nil {
			return r.Delete(ctx, existing)
		}
		return nil
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + hpaSuffix,
			Namespace: cluster.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
		if err := ctrl.SetControllerReference(cluster, hpa, r.Scheme); err != nil {
			return err
		}

		hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       cluster.Name + deploymentSuffix,
			},
			MinReplicas: &cluster.Spec.Autoscaling.MinReplicas,
			MaxReplicas: cluster.Spec.Autoscaling.MaxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cluster.Spec.Autoscaling.TargetCPUUtilizationPercentage,
						},
					},
				},
			},
		}
		return nil
	})
	return err
}

// countPodRestarts sums the restart count for all containers in pods matching the cluster's labels.
func (r *RadiusClusterReconciler) countPodRestarts(ctx context.Context, namespace, clusterName string) int32 {
	podList := &corev1.PodList{}
	labels := client.MatchingLabels{
		"app.kubernetes.io/name":       "freeradius",
		"app.kubernetes.io/instance":   clusterName,
		"app.kubernetes.io/managed-by": "freeradius-operator",
	}
	if err := r.List(ctx, podList, client.InNamespace(namespace), labels); err != nil {
		return 0
	}
	var total int32
	for _, pod := range podList.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			total += cs.RestartCount
		}
	}
	return total
}

// collectSecretRefs gathers all unique SecretRef names from the cluster spec and its clients.
func collectSecretRefs(cluster *radiusv1alpha1.RadiusCluster, clients []radiusv1alpha1.RadiusClient) []renderer.SecretRef {
	seen := make(map[string]bool)
	var refs []renderer.SecretRef

	add := func(name, key string) {
		if name == "" {
			return
		}
		// Deduplicate by secret name. The volume mount exposes
		// the entire secret, so we only need one entry per name.
		if seen[name] {
			return
		}
		seen[name] = true
		refs = append(refs, renderer.SecretRef{Name: name, Key: key})
	}

	// TLS secret
	if cluster.Spec.TLS != nil && cluster.Spec.TLS.Enabled {
		add(cluster.Spec.TLS.SecretRef.Name, cluster.Spec.TLS.SecretRef.Key)
	}

	// Module secrets
	for _, mod := range cluster.Spec.Modules {
		if !mod.Enabled {
			continue
		}
		if mod.SQL != nil {
			add(mod.SQL.PasswordRef.Name, mod.SQL.PasswordRef.Key)
		}
		if mod.LDAP != nil {
			add(mod.LDAP.PasswordRef.Name, mod.LDAP.PasswordRef.Key)
		}
		if mod.REST != nil && mod.REST.PasswordRef != nil {
			add(mod.REST.PasswordRef.Name, mod.REST.PasswordRef.Key)
		}
		if mod.Redis != nil && mod.Redis.PasswordRef != nil {
			add(mod.Redis.PasswordRef.Name, mod.Redis.PasswordRef.Key)
		}
	}

	// Client secrets
	for _, c := range clients {
		add(c.Spec.SecretRef.Name, c.Spec.SecretRef.Key)
	}

	return refs
}

// buildRenderContext converts Kubernetes types to renderer types.
func buildRenderContext(cluster *radiusv1alpha1.RadiusCluster, clients []radiusv1alpha1.RadiusClient, policies []radiusv1alpha1.RadiusPolicy) renderer.RenderContext {
	// Convert modules
	var modules []renderer.ModuleConfig
	for _, m := range cluster.Spec.Modules {
		mod := renderer.ModuleConfig{
			Name:    m.Name,
			Type:    m.Type,
			Enabled: m.Enabled,
		}
		if m.SQL != nil {
			mod.SQL = &renderer.SQLConfig{
				Dialect:     m.SQL.Dialect,
				Server:      m.SQL.Server,
				Port:        m.SQL.Port,
				Database:    m.SQL.Database,
				Login:       m.SQL.Login,
				PasswordRef: renderer.SecretRef{Name: m.SQL.PasswordRef.Name, Key: m.SQL.PasswordRef.Key},
			}
		}
		if m.LDAP != nil {
			mod.LDAP = &renderer.LDAPConfig{
				Server:      m.LDAP.Server,
				Port:        m.LDAP.Port,
				BaseDN:      m.LDAP.BaseDN,
				Identity:    m.LDAP.Identity,
				PasswordRef: renderer.SecretRef{Name: m.LDAP.PasswordRef.Name, Key: m.LDAP.PasswordRef.Key},
			}
		}
		if m.EAP != nil {
			mod.EAP = &renderer.EAPConfig{
				DefaultEAPType: m.EAP.DefaultEAPType,
			}
			if m.EAP.TLS != nil {
				mod.EAP.TLS = &renderer.EAPTLSConfig{
					CertFile: m.EAP.TLS.CertFile,
					KeyFile:  m.EAP.TLS.KeyFile,
				}
			}
			if m.EAP.TTLS != nil {
				mod.EAP.TTLS = &renderer.EAPTTLSConfig{
					DefaultEAPType: m.EAP.TTLS.DefaultEAPType,
					VirtualServer:  m.EAP.TTLS.VirtualServer,
				}
			}
			if m.EAP.PEAP != nil {
				mod.EAP.PEAP = &renderer.EAPPEAPConfig{
					DefaultEAPType: m.EAP.PEAP.DefaultEAPType,
					VirtualServer:  m.EAP.PEAP.VirtualServer,
				}
			}
		}
		if m.REST != nil {
			mod.REST = &renderer.RESTConfig{
				ConnectURI: m.REST.ConnectURI,
				Auth:       m.REST.Auth,
			}
			if m.REST.PasswordRef != nil {
				mod.REST.PasswordRef = &renderer.SecretRef{Name: m.REST.PasswordRef.Name, Key: m.REST.PasswordRef.Key}
			}
			if m.REST.Authorize != nil {
				mod.REST.Authorize = &renderer.RESTStageConfig{URI: m.REST.Authorize.URI, Method: m.REST.Authorize.Method}
			}
			if m.REST.Authenticate != nil {
				mod.REST.Authenticate = &renderer.RESTStageConfig{URI: m.REST.Authenticate.URI, Method: m.REST.Authenticate.Method}
			}
			if m.REST.Preacct != nil {
				mod.REST.Preacct = &renderer.RESTStageConfig{URI: m.REST.Preacct.URI, Method: m.REST.Preacct.Method}
			}
			if m.REST.Accounting != nil {
				mod.REST.Accounting = &renderer.RESTStageConfig{URI: m.REST.Accounting.URI, Method: m.REST.Accounting.Method}
			}
			if m.REST.PostAuth != nil {
				mod.REST.PostAuth = &renderer.RESTStageConfig{URI: m.REST.PostAuth.URI, Method: m.REST.PostAuth.Method}
			}
			if m.REST.PreProxy != nil {
				mod.REST.PreProxy = &renderer.RESTStageConfig{URI: m.REST.PreProxy.URI, Method: m.REST.PreProxy.Method}
			}
			if m.REST.PostProxy != nil {
				mod.REST.PostProxy = &renderer.RESTStageConfig{URI: m.REST.PostProxy.URI, Method: m.REST.PostProxy.Method}
			}
		}
		if m.Redis != nil {
			mod.Redis = &renderer.RedisConfig{
				Server:   m.Redis.Server,
				Port:     m.Redis.Port,
				Database: m.Redis.Database,
			}
			if m.Redis.PasswordRef != nil {
				mod.Redis.PasswordRef = &renderer.SecretRef{Name: m.Redis.PasswordRef.Name, Key: m.Redis.PasswordRef.Key}
			}
		}
		modules = append(modules, mod)
	}

	// Convert clients
	var renderClients []renderer.ClientSpec
	for _, c := range clients {
		renderClients = append(renderClients, renderer.ClientSpec{
			Name:      c.Name,
			IP:        c.Spec.IP,
			SecretRef: renderer.SecretRef{Name: c.Spec.SecretRef.Name, Key: c.Spec.SecretRef.Key},
			NASType:   c.Spec.NASType,
		})
	}

	// Convert policies
	var renderPolicies []renderer.PolicySpec
	for _, p := range policies {
		policy := renderer.PolicySpec{
			Name:     p.Name,
			Stage:    p.Spec.Stage,
			Priority: p.Spec.Priority,
		}
		if p.Spec.Match != nil {
			policy.Match = &renderer.PolicyMatch{}
			for _, leaf := range p.Spec.Match.All {
				policy.Match.All = append(policy.Match.All, renderer.MatchLeaf{
					Attribute: leaf.Attribute, Operator: leaf.Operator, Value: leaf.Value,
				})
			}
			for _, leaf := range p.Spec.Match.Any {
				policy.Match.Any = append(policy.Match.Any, renderer.MatchLeaf{
					Attribute: leaf.Attribute, Operator: leaf.Operator, Value: leaf.Value,
				})
			}
			for _, leaf := range p.Spec.Match.None {
				policy.Match.None = append(policy.Match.None, renderer.MatchLeaf{
					Attribute: leaf.Attribute, Operator: leaf.Operator, Value: leaf.Value,
				})
			}
		}
		for _, a := range p.Spec.Actions {
			policy.Actions = append(policy.Actions, renderer.PolicyAction{
				Type: a.Type, Module: a.Module, Attribute: a.Attribute, Value: a.Value,
			})
		}
		renderPolicies = append(renderPolicies, policy)
	}

	return renderer.RenderContext{
		Cluster: renderer.ClusterSpec{
			Replicas: cluster.Spec.Replicas,
			Image:    cluster.Spec.Image,
			Modules:  modules,
		},
		Clients:  renderClients,
		Policies: renderPolicies,
	}
}

func isInvalidError(err error) bool {
	switch err.(type) {
	case *renderer.InvalidModuleError, *renderer.InvalidStageError, *renderer.InvalidActionError:
		return true
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *RadiusClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&radiusv1alpha1.RadiusCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Watches(&radiusv1alpha1.RadiusClient{}, handler.EnqueueRequestsFromMapFunc(enqueueOwningCluster)).
		Watches(&radiusv1alpha1.RadiusPolicy{}, handler.EnqueueRequestsFromMapFunc(enqueueOwningCluster)).
		Complete(r)
}

// enqueueOwningCluster maps a RadiusClient or RadiusPolicy to its owning RadiusCluster.
func enqueueOwningCluster(ctx context.Context, obj client.Object) []reconcile.Request {
	var clusterRef string
	switch o := obj.(type) {
	case *radiusv1alpha1.RadiusClient:
		clusterRef = o.Spec.ClusterRef
	case *radiusv1alpha1.RadiusPolicy:
		clusterRef = o.Spec.ClusterRef
	default:
		return nil
	}
	if clusterRef == "" {
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      clusterRef,
		},
	}}
}
