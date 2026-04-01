package main

import (
	"flag"
	"os"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	crtzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
	"github.com/example/freeradius-operator/internal/controller"
	"github.com/example/freeradius-operator/internal/renderer"
	"github.com/example/freeradius-operator/internal/status"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(radiusv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr, probeAddr, watchNamespace string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint bind address.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe bind address.")
	flag.StringVar(&watchNamespace, "watch-namespace", "", "Namespace to watch (empty = all).")
	flag.Parse()

	ctrl.SetLogger(crtzap.New(
		crtzap.JSONEncoder(func(ec *zapcore.EncoderConfig) {
			ec.TimeKey = "timestamp"
			ec.EncodeTime = zapcore.ISO8601TimeEncoder
		}),
	))

	mgrOpts := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
	}
	if watchNamespace != "" {
		mgrOpts.Cache = cache.Options{DefaultNamespaces: map[string]cache.Config{watchNamespace: {}}}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	statusReporter := status.New(mgr.GetClient())
	configRenderer := renderer.New()

	if err := (&controller.RadiusClusterReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Renderer: configRenderer, Status: statusReporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RadiusCluster")
		os.Exit(1)
	}

	if err := (&controller.RadiusClientReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Status: statusReporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RadiusClient")
		os.Exit(1)
	}

	if err := (&controller.RadiusPolicyReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Status: statusReporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RadiusPolicy")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
