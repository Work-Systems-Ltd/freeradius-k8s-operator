package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

type RadiusClusterReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Renderer      renderer.ConfigRenderer
	Status        *status.StatusReporter
	OperatorImage string
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
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *RadiusClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	start := time.Now()
	result := "success"

	defer func() {
		metrics.ReconcileTotal.WithLabelValues(req.Namespace, req.Name, "RadiusCluster", result).Inc()
		metrics.ReconcileDuration.WithLabelValues(req.Namespace, req.Name, "RadiusCluster").Observe(time.Since(start).Seconds())
	}()

	cluster := &radiusv1alpha1.RadiusCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		result = "error"
		return ctrl.Result{}, err
	}

	if err := r.Status.SetProgressing(ctx, cluster, true); err != nil {
		logger.Error(err, "unable to set Progressing status")
	}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		result = "error"
		return ctrl.Result{}, err
	}

	matchingClients, err := r.listMatchingClients(ctx, req.Namespace, cluster.Name)
	if err != nil {
		result = "error"
		return ctrl.Result{}, err
	}

	matchingPolicies, err := r.listMatchingPolicies(ctx, req.Namespace, cluster.Name)
	if err != nil {
		result = "error"
		return ctrl.Result{}, err
	}

	secretRefs := collectSecretRefs(cluster, matchingClients)
	for _, ref := range secretRefs {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: ref.Name}, secret); err != nil {
			if errors.IsNotFound(err) {
				logger.Error(err, "referenced Secret not found", "secret", ref.Name)
				_ = r.Status.SetDegraded(ctx, cluster, true, "MissingSecret", fmt.Sprintf("Secret %q not found", ref.Name))
				result = "error"
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			result = "error"
			return ctrl.Result{}, fmt.Errorf("fetching Secret %q: %w", ref.Name, err)
		}
	}

	renderCtx := buildRenderContext(cluster, matchingClients, matchingPolicies)
	configFiles, err := r.Renderer.Render(renderCtx)
	if err != nil {
		if isInvalidError(err) {
			logger.Error(err, "invalid spec")
			_ = r.Status.SetDegraded(ctx, cluster, true, "InvalidSpec", err.Error())
			result = "error"
			return ctrl.Result{}, nil
		}
		result = "error"
		return ctrl.Result{}, fmt.Errorf("rendering config: %w", err)
	}

	configHash := computeConfigHash(configFiles, matchingClients)

	for _, reconcileFn := range []func() error{
		func() error { return r.reconcileRBAC(ctx, cluster) },
		func() error { return r.reconcileConfigMap(ctx, cluster, configFiles) },
		func() error { return r.reconcileDeploymentsAndServices(ctx, cluster, secretRefs, configHash) },
		func() error { return r.reconcileHPAs(ctx, cluster) },
		func() error { return r.reconcilePDB(ctx, cluster) },
	} {
		if err := reconcileFn(); err != nil {
			result = "error"
			return ctrl.Result{}, err
		}
	}

	deploy := &appsv1.Deployment{}
	deployName := types.NamespacedName{Namespace: req.Namespace, Name: cluster.Name + deploymentSuffix}
	if err := r.Get(ctx, deployName, deploy); err == nil {
		if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
			result = "error"
			return ctrl.Result{}, err
		}
		podRestarts := r.countPodRestarts(ctx, req.Namespace, cluster.Name)
		if err := r.Status.UpdateClusterStatus(ctx, cluster, deploy.Status.ReadyReplicas, cluster.Spec.Image, podRestarts); err != nil {
			logger.Error(err, "unable to update cluster status fields")
		}
	}

	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		result = "error"
		return ctrl.Result{}, err
	}
	r.Status.SetConditionLocal(&cluster.Status.Conditions, "Degraded", false, "AllSecretsPresent", "All referenced secrets are available")
	r.Status.SetConditionLocal(&cluster.Status.Conditions, "Available", true, "ReconcileComplete", "All resources reconciled successfully")
	r.Status.SetConditionLocal(&cluster.Status.Conditions, "Progressing", false, "ReconcileComplete", "Reconciliation completed successfully")
	if err := r.Status.FlushClusterStatus(ctx, cluster); err != nil {
		logger.Error(err, "unable to update final status conditions")
	}

	logger.Info("reconciliation complete")
	return ctrl.Result{}, nil
}

func (r *RadiusClusterReconciler) listMatchingClients(ctx context.Context, ns, clusterName string) ([]radiusv1alpha1.RadiusClient, error) {
	list := &radiusv1alpha1.RadiusClientList{}
	if err := r.List(ctx, list, client.InNamespace(ns)); err != nil {
		return nil, fmt.Errorf("listing RadiusClients: %w", err)
	}
	var out []radiusv1alpha1.RadiusClient
	for _, c := range list.Items {
		if c.Spec.ClusterRef == clusterName {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *RadiusClusterReconciler) listMatchingPolicies(ctx context.Context, ns, clusterName string) ([]radiusv1alpha1.RadiusPolicy, error) {
	list := &radiusv1alpha1.RadiusPolicyList{}
	if err := r.List(ctx, list, client.InNamespace(ns)); err != nil {
		return nil, fmt.Errorf("listing RadiusPolicies: %w", err)
	}
	var out []radiusv1alpha1.RadiusPolicy
	for _, p := range list.Items {
		if p.Spec.ClusterRef == clusterName {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *RadiusClusterReconciler) reconcileConfigMap(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, files renderer.ConfigFiles) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + configMapSuffix, Namespace: cluster.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if err := ctrl.SetControllerReference(cluster, cm, r.Scheme); err != nil {
			return err
		}
		cm.Data = make(map[string]string, len(files))
		for k, v := range files {
			cm.Data[strings.ReplaceAll(k, "/", "__")] = v
		}
		return nil
	})
	return err
}

type endpointDef struct {
	role      string
	suffix    string
	ports     []corev1.ServicePort
	cPorts    []corev1.ContainerPort
	replicas  int32
	autoscale *radiusv1alpha1.AutoscalingConfig
	svcCfg    *radiusv1alpha1.ServiceConfig
}

func (r *RadiusClusterReconciler) reconcileDeploymentsAndServices(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, secretRefs []renderer.SecretRef, configHash string) error {
	endpoints := r.buildEndpoints(cluster)
	for _, ep := range endpoints {
		if err := r.reconcileOneDeployment(ctx, cluster, secretRefs, configHash, ep); err != nil {
			return err
		}
		if err := r.reconcileOneService(ctx, cluster, ep); err != nil {
			return err
		}
	}
	return r.cleanupStaleResources(ctx, cluster)
}

func (r *RadiusClusterReconciler) buildEndpoints(cluster *radiusv1alpha1.RadiusCluster) []endpointDef {
	coaPort := int32(3799)
	if cluster.Spec.CoA != nil && cluster.Spec.CoA.Port > 0 {
		coaPort = cluster.Spec.CoA.Port
	}

	if cluster.Spec.Services == nil {
		ports := []corev1.ServicePort{
			{Name: "auth", Port: 1812, TargetPort: intstr.FromInt(1812), Protocol: corev1.ProtocolUDP},
			{Name: "acct", Port: 1813, TargetPort: intstr.FromInt(1813), Protocol: corev1.ProtocolUDP},
		}
		cPorts := []corev1.ContainerPort{
			{Name: "auth", ContainerPort: 1812, Protocol: corev1.ProtocolUDP},
			{Name: "acct", ContainerPort: 1813, Protocol: corev1.ProtocolUDP},
		}
		if cluster.Spec.CoA != nil && cluster.Spec.CoA.Enabled {
			ports = append(ports, corev1.ServicePort{Name: "coa", Port: coaPort, TargetPort: intstr.FromInt(int(coaPort)), Protocol: corev1.ProtocolUDP})
			cPorts = append(cPorts, corev1.ContainerPort{Name: "coa", ContainerPort: coaPort, Protocol: corev1.ProtocolUDP})
		}
		return []endpointDef{{
			suffix: deploymentSuffix, ports: ports, cPorts: cPorts,
			replicas: cluster.Spec.Replicas, autoscale: cluster.Spec.Autoscaling,
			svcCfg: cluster.Spec.Service,
		}}
	}

	var eps []endpointDef
	svcs := cluster.Spec.Services

	if svcs.Auth != nil {
		eps = append(eps, endpointDef{
			role: "auth", suffix: "-freeradius-auth",
			ports:    []corev1.ServicePort{{Name: "auth", Port: 1812, TargetPort: intstr.FromInt(1812), Protocol: corev1.ProtocolUDP}},
			cPorts:   []corev1.ContainerPort{{Name: "auth", ContainerPort: 1812, Protocol: corev1.ProtocolUDP}},
			replicas: svcs.Auth.Replicas, autoscale: svcs.Auth.Autoscaling,
			svcCfg: &svcs.Auth.ServiceConfig,
		})
	}
	if svcs.Accounting != nil {
		eps = append(eps, endpointDef{
			role: "acct", suffix: "-freeradius-acct",
			ports:    []corev1.ServicePort{{Name: "acct", Port: 1813, TargetPort: intstr.FromInt(1813), Protocol: corev1.ProtocolUDP}},
			cPorts:   []corev1.ContainerPort{{Name: "acct", ContainerPort: 1813, Protocol: corev1.ProtocolUDP}},
			replicas: svcs.Accounting.Replicas, autoscale: svcs.Accounting.Autoscaling,
			svcCfg: &svcs.Accounting.ServiceConfig,
		})
	}
	if svcs.CoA != nil {
		eps = append(eps, endpointDef{
			role: "coa", suffix: "-freeradius-coa",
			ports:    []corev1.ServicePort{{Name: "coa", Port: coaPort, TargetPort: intstr.FromInt(int(coaPort)), Protocol: corev1.ProtocolUDP}},
			cPorts:   []corev1.ContainerPort{{Name: "coa", ContainerPort: coaPort, Protocol: corev1.ProtocolUDP}},
			replicas: svcs.CoA.Replicas, autoscale: svcs.CoA.Autoscaling,
			svcCfg: &svcs.CoA.ServiceConfig,
		})
	}
	return eps
}

func (r *RadiusClusterReconciler) cleanupStaleResources(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster) error {
	var staleSuffixes []string
	if cluster.Spec.Services != nil {
		staleSuffixes = []string{deploymentSuffix}
	} else {
		staleSuffixes = []string{"-freeradius-auth", "-freeradius-acct", "-freeradius-coa"}
	}

	for _, suffix := range staleSuffixes {
		name := types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name + suffix}
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, name, deploy); err == nil {
			if err := r.Delete(ctx, deploy); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
		svc := &corev1.Service{}
		if err := r.Get(ctx, name, svc); err == nil {
			if err := r.Delete(ctx, svc); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (r *RadiusClusterReconciler) reconcileOneDeployment(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, secretRefs []renderer.SecretRef, configHash string, ep endpointDef) error {
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + ep.suffix, Namespace: cluster.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		if err := ctrl.SetControllerReference(cluster, deploy, r.Scheme); err != nil {
			return err
		}
		labels := endpointLabels(cluster.Name, ep.role)
		maxUnavailable := intstr.FromInt(0)
		maxSurge := intstr.FromInt(1)

		annotations := map[string]string{
			"radius.operator.io/config-hash": configHash,
		}

		deploy.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
				Spec:       r.buildPodSpec(cluster, secretRefs, ep.cPorts),
			},
		}

		if ep.autoscale == nil || !ep.autoscale.Enabled {
			replicas := ep.replicas
			if replicas < 1 {
				replicas = 1
			}
			deploy.Spec.Replicas = &replicas
		}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) buildPodSpec(cluster *radiusv1alpha1.RadiusCluster, secretRefs []renderer.SecretRef, containerPorts []corev1.ContainerPort) corev1.PodSpec {
	configMapName := cluster.Name + configMapSuffix
	falseVal := false
	trueVal := true
	freerad := int64(101)

	volumes := []corev1.Volume{
		{Name: "freeradius-config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configMapName}},
		}},
		{Name: "freeradius-config-rendered", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}

	for _, ref := range secretRefs {
		mode := int32(0400)
		volumes = append(volumes, corev1.Volume{
			Name: "secret-" + ref.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: ref.Name, DefaultMode: &mode},
			},
		})
	}

	restrictedSC := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &falseVal,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}

	stockCopyContainer := corev1.Container{
		Name:    "stock-config",
		Image:   cluster.Spec.Image,
		Command: []string{"sh", "-c", `cp -rL /etc/freeradius/. /config-rendered/ 2>/dev/null; true`},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "freeradius-config-rendered", MountPath: "/config-rendered"},
		},
		SecurityContext: restrictedSC,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("16Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("32Mi")},
		},
	}

	initContainer := corev1.Container{
		Name:  "config-init",
		Image: "docker.io/library/busybox:1.36",
		Command: []string{"sh", "-c", `cd /config-flat
for f in *; do
  target=$(echo "$f" | sed 's/__/\//g')
  mkdir -p "/etc/freeradius/$(dirname "$target")"
  cp "$f" "/etc/freeradius/$target"
done`},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "freeradius-config", MountPath: "/config-flat", ReadOnly: true},
			{Name: "freeradius-config-rendered", MountPath: "/etc/freeradius"},
		},
		SecurityContext: restrictedSC,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("16Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("32Mi")},
		},
	}

	clientInitContainer := corev1.Container{
		Name:            "render-clients",
		Image:           r.OperatorImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{"/operator",
			"--mode=render-clients",
			"--cluster-name=" + cluster.Name,
			"--namespace=" + cluster.Namespace,
			"--output=/etc/freeradius/clients.conf",
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "freeradius-config-rendered", MountPath: "/etc/freeradius"},
		},
		SecurityContext: restrictedSC,
		Resources:       initContainerResources(cluster),
	}

	mainMounts := []corev1.VolumeMount{
		{Name: "freeradius-config-rendered", MountPath: "/etc/freeradius"},
	}
	for _, ref := range secretRefs {
		mainMounts = append(mainMounts, corev1.VolumeMount{
			Name: "secret-" + ref.Name, MountPath: "/etc/freeradius/secrets/" + ref.Name, ReadOnly: true,
		})
	}

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
		Ports:           containerPorts,
		SecurityContext: restrictedSC,
		VolumeMounts:    mainMounts,
		Resources:       cluster.Spec.Resources,
		LivenessProbe:   liveness,
		ReadinessProbe:  readiness,
	}

	var affinity *corev1.Affinity
	if cluster.Spec.Affinity != nil {
		affinity = cluster.Spec.Affinity
	} else if cluster.Spec.Replicas > 1 {
		weight := int32(100)
		affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
					Weight: weight,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{MatchLabels: podLabels(cluster.Name)},
						TopologyKey:   "kubernetes.io/hostname",
					},
				}},
			},
		}
	}

	podSpec := corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &trueVal,
			RunAsUser:    &freerad,
			RunAsGroup:   &freerad,
		},
		Affinity:                  affinity,
		TopologySpreadConstraints: cluster.Spec.TopologySpreadConstraints,
		ServiceAccountName:        cluster.Name + serviceAccountSuffix,
		InitContainers:            []corev1.Container{stockCopyContainer, initContainer, clientInitContainer},
		Containers:                []corev1.Container{mainContainer},
		Volumes:                   volumes,
	}

	return podSpec
}

func defaultProbes() (*corev1.Probe, *corev1.Probe) {
	return &corev1.Probe{
			ProbeHandler:        corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: []string{"freeradius", "-C"}}},
			InitialDelaySeconds: 10, PeriodSeconds: 30,
		}, &corev1.Probe{
			ProbeHandler:        corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "echo 'Message-Authenticator = 0x00' | radclient -x 127.0.0.1:1812 status testing123"}}},
			InitialDelaySeconds: 5, PeriodSeconds: 10,
		}
}

func (r *RadiusClusterReconciler) reconcileOneService(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, ep endpointDef) error {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + ep.suffix, Namespace: cluster.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(cluster, svc, r.Scheme); err != nil {
			return err
		}

		svcType := corev1.ServiceTypeClusterIP
		if ep.svcCfg != nil && ep.svcCfg.Type != "" {
			svcType = ep.svcCfg.Type
		}

		svc.Spec = corev1.ServiceSpec{
			Type:     svcType,
			Selector: endpointLabels(cluster.Name, ep.role),
			Ports:    ep.ports,
		}

		if ep.svcCfg != nil {
			if ep.svcCfg.LoadBalancerIP != "" {
				svc.Spec.LoadBalancerIP = ep.svcCfg.LoadBalancerIP
			}
			if ep.svcCfg.ExternalTrafficPolicy != "" {
				svc.Spec.ExternalTrafficPolicy = ep.svcCfg.ExternalTrafficPolicy
			}
			if len(ep.svcCfg.Annotations) > 0 {
				svc.Annotations = ep.svcCfg.Annotations
			}
		}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) reconcileHPAs(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster) error {
	endpoints := r.buildEndpoints(cluster)
	for _, ep := range endpoints {
		if err := r.reconcileOneHPA(ctx, cluster, ep); err != nil {
			return err
		}
	}
	return nil
}

func (r *RadiusClusterReconciler) reconcileOneHPA(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster, ep endpointDef) error {
	hpaName := types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name + ep.suffix}

	if ep.autoscale == nil || !ep.autoscale.Enabled {
		existing := &autoscalingv2.HorizontalPodAutoscaler{}
		if err := r.Get(ctx, hpaName, existing); err == nil {
			return r.Delete(ctx, existing)
		}
		return nil
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + ep.suffix, Namespace: cluster.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
		if err := ctrl.SetControllerReference(cluster, hpa, r.Scheme); err != nil {
			return err
		}
		hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1", Kind: "Deployment", Name: cluster.Name + ep.suffix,
			},
			MinReplicas: &ep.autoscale.MinReplicas,
			MaxReplicas: ep.autoscale.MaxReplicas,
			Metrics: []autoscalingv2.MetricSpec{{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name:   corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: &ep.autoscale.TargetCPUUtilizationPercentage},
				},
			}},
		}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) reconcilePDB(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster) error {
	pdbName := types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name + pdbSuffix}

	if cluster.Spec.PDB == nil && cluster.Spec.Replicas <= 1 {
		existing := &policyv1.PodDisruptionBudget{}
		if err := r.Get(ctx, pdbName, existing); err == nil {
			return r.Delete(ctx, existing)
		}
		return nil
	}

	pdb := &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + pdbSuffix, Namespace: cluster.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pdb, func() error {
		if err := ctrl.SetControllerReference(cluster, pdb, r.Scheme); err != nil {
			return err
		}
		pdb.Spec = policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podLabels(cluster.Name)},
		}
		if cluster.Spec.PDB != nil {
			pdb.Spec.MinAvailable = cluster.Spec.PDB.MinAvailable
			pdb.Spec.MaxUnavailable = cluster.Spec.PDB.MaxUnavailable
		} else {
			minAvail := intstr.FromInt(1)
			pdb.Spec.MinAvailable = &minAvail
		}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) reconcileRBAC(ctx context.Context, cluster *radiusv1alpha1.RadiusCluster) error {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + serviceAccountSuffix, Namespace: cluster.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		return ctrl.SetControllerReference(cluster, sa, r.Scheme)
	}); err != nil {
		return err
	}

	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + roleSuffix, Namespace: cluster.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if err := ctrl.SetControllerReference(cluster, role, r.Scheme); err != nil {
			return err
		}
		role.Rules = []rbacv1.PolicyRule{{
			APIGroups: []string{"radius.operator.io"},
			Resources: []string{"radiusclients"},
			Verbs:     []string{"get", "list", "watch"},
		}}
		return nil
	}); err != nil {
		return err
	}

	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: cluster.Name + roleBindingSuffix, Namespace: cluster.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
		if err := ctrl.SetControllerReference(cluster, rb, r.Scheme); err != nil {
			return err
		}
		rb.RoleRef = rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: role.Name}
		rb.Subjects = []rbacv1.Subject{{Kind: "ServiceAccount", Name: sa.Name, Namespace: cluster.Namespace}}
		return nil
	})
	return err
}

func (r *RadiusClusterReconciler) countPodRestarts(ctx context.Context, namespace, clusterName string) int32 {
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabels(podLabels(clusterName))); err != nil {
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

func computeConfigHash(files renderer.ConfigFiles, clients []radiusv1alpha1.RadiusClient) string {
	h := sha256.New()

	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(files[k]))
	}

	for _, c := range clients {
		h.Write([]byte(c.Name))
		h.Write([]byte(c.Spec.IP))
		h.Write([]byte(c.Spec.SecretRef.Name))
		h.Write([]byte(c.Spec.SecretRef.Key))
		h.Write([]byte(c.Spec.NASType))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func initContainerResources(cluster *radiusv1alpha1.RadiusCluster) corev1.ResourceRequirements {
	if cluster.Spec.InitResources != nil {
		return *cluster.Spec.InitResources
	}
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("16Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("32Mi")},
	}
}

func podLabels(clusterName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "freeradius",
		"app.kubernetes.io/instance":   clusterName,
		"app.kubernetes.io/managed-by": "freeradius-operator",
	}
}

func endpointLabels(clusterName, role string) map[string]string {
	labels := podLabels(clusterName)
	if role != "" {
		labels["radius.operator.io/role"] = role
	}
	return labels
}

func collectSecretRefs(cluster *radiusv1alpha1.RadiusCluster, clients []radiusv1alpha1.RadiusClient) []renderer.SecretRef {
	seen := make(map[string]bool)
	var refs []renderer.SecretRef

	add := func(name, key string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		refs = append(refs, renderer.SecretRef{Name: name, Key: key})
	}

	if cluster.Spec.TLS != nil && cluster.Spec.TLS.Enabled {
		add(cluster.Spec.TLS.SecretRef.Name, cluster.Spec.TLS.SecretRef.Key)
	}
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
	for _, c := range clients {
		add(c.Spec.SecretRef.Name, c.Spec.SecretRef.Key)
	}
	return refs
}

func buildRenderContext(cluster *radiusv1alpha1.RadiusCluster, clients []radiusv1alpha1.RadiusClient, policies []radiusv1alpha1.RadiusPolicy) renderer.RenderContext {
	modules := make([]renderer.ModuleConfig, 0, len(cluster.Spec.Modules))
	for _, m := range cluster.Spec.Modules {
		mod := renderer.ModuleConfig{Name: m.Name, Type: m.Type, Enabled: m.Enabled}
		if m.SQL != nil {
			mod.SQL = &renderer.SQLConfig{
				Dialect: m.SQL.Dialect, Server: m.SQL.Server, Port: m.SQL.Port,
				Database: m.SQL.Database, Login: m.SQL.Login,
				PasswordRef: renderer.SecretRef{Name: m.SQL.PasswordRef.Name, Key: m.SQL.PasswordRef.Key},
			}
		}
		if m.LDAP != nil {
			mod.LDAP = &renderer.LDAPConfig{
				Server: m.LDAP.Server, Port: m.LDAP.Port, BaseDN: m.LDAP.BaseDN, Identity: m.LDAP.Identity,
				PasswordRef: renderer.SecretRef{Name: m.LDAP.PasswordRef.Name, Key: m.LDAP.PasswordRef.Key},
			}
		}
		if m.EAP != nil {
			mod.EAP = &renderer.EAPConfig{DefaultEAPType: m.EAP.DefaultEAPType}
			if m.EAP.TLS != nil {
				mod.EAP.TLS = &renderer.EAPTLSConfig{CertFile: m.EAP.TLS.CertFile, KeyFile: m.EAP.TLS.KeyFile}
			}
			if m.EAP.TTLS != nil {
				mod.EAP.TTLS = &renderer.EAPTTLSConfig{DefaultEAPType: m.EAP.TTLS.DefaultEAPType, VirtualServer: m.EAP.TTLS.VirtualServer}
			}
			if m.EAP.PEAP != nil {
				mod.EAP.PEAP = &renderer.EAPPEAPConfig{DefaultEAPType: m.EAP.PEAP.DefaultEAPType, VirtualServer: m.EAP.PEAP.VirtualServer}
			}
		}
		if m.REST != nil {
			mod.REST = &renderer.RESTConfig{ConnectURI: m.REST.ConnectURI, Auth: m.REST.Auth}
			if m.REST.PasswordRef != nil {
				mod.REST.PasswordRef = &renderer.SecretRef{Name: m.REST.PasswordRef.Name, Key: m.REST.PasswordRef.Key}
			}
			for _, pair := range []struct {
				src  *radiusv1alpha1.RESTStageConfig
				dest **renderer.RESTStageConfig
			}{
				{m.REST.Authorize, &mod.REST.Authorize}, {m.REST.Authenticate, &mod.REST.Authenticate},
				{m.REST.Preacct, &mod.REST.Preacct}, {m.REST.Accounting, &mod.REST.Accounting},
				{m.REST.PostAuth, &mod.REST.PostAuth}, {m.REST.PreProxy, &mod.REST.PreProxy},
				{m.REST.PostProxy, &mod.REST.PostProxy},
			} {
				if pair.src != nil {
					*pair.dest = &renderer.RESTStageConfig{URI: pair.src.URI, Method: pair.src.Method}
				}
			}
		}
		if m.Redis != nil {
			mod.Redis = &renderer.RedisConfig{Server: m.Redis.Server, Port: m.Redis.Port, Database: m.Redis.Database}
			if m.Redis.PasswordRef != nil {
				mod.Redis.PasswordRef = &renderer.SecretRef{Name: m.Redis.PasswordRef.Name, Key: m.Redis.PasswordRef.Key}
			}
		}
		modules = append(modules, mod)
	}

	renderClients := make([]renderer.ClientSpec, 0, len(clients))
	for _, c := range clients {
		renderClients = append(renderClients, renderer.ClientSpec{
			Name: c.Name, IP: c.Spec.IP,
			SecretRef: renderer.SecretRef{Name: c.Spec.SecretRef.Name, Key: c.Spec.SecretRef.Key},
			NASType:   c.Spec.NASType,
		})
	}

	renderPolicies := make([]renderer.PolicySpec, 0, len(policies))
	for _, p := range policies {
		policy := renderer.PolicySpec{Name: p.Name, Stage: p.Spec.Stage, Priority: p.Spec.Priority}
		if p.Spec.Match != nil {
			policy.Match = &renderer.PolicyMatch{}
			for _, leaf := range p.Spec.Match.All {
				policy.Match.All = append(policy.Match.All, renderer.MatchLeaf{Attribute: leaf.Attribute, Operator: leaf.Operator, Value: leaf.Value})
			}
			for _, leaf := range p.Spec.Match.Any {
				policy.Match.Any = append(policy.Match.Any, renderer.MatchLeaf{Attribute: leaf.Attribute, Operator: leaf.Operator, Value: leaf.Value})
			}
			for _, leaf := range p.Spec.Match.None {
				policy.Match.None = append(policy.Match.None, renderer.MatchLeaf{Attribute: leaf.Attribute, Operator: leaf.Operator, Value: leaf.Value})
			}
		}
		for _, a := range p.Spec.Actions {
			policy.Actions = append(policy.Actions, renderer.PolicyAction{Type: a.Type, Module: a.Module, Attribute: a.Attribute, Value: a.Value})
		}
		renderPolicies = append(renderPolicies, policy)
	}

	clusterSpec := renderer.ClusterSpec{Replicas: cluster.Spec.Replicas, Image: cluster.Spec.Image, Modules: modules}
	if cluster.Spec.CoA != nil && cluster.Spec.CoA.Enabled {
		clusterSpec.CoAEnabled = true
		clusterSpec.CoAPort = cluster.Spec.CoA.Port
		if clusterSpec.CoAPort == 0 {
			clusterSpec.CoAPort = 3799
		}
	}

	return renderer.RenderContext{
		Cluster:  clusterSpec,
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

func (r *RadiusClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&radiusv1alpha1.RadiusCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Watches(&radiusv1alpha1.RadiusClient{}, handler.EnqueueRequestsFromMapFunc(enqueueOwningCluster)).
		Watches(&radiusv1alpha1.RadiusPolicy{}, handler.EnqueueRequestsFromMapFunc(enqueueOwningCluster)).
		Complete(r)
}

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
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: clusterRef}}}
}
