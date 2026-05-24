package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	secretsv1alpha1 "github.com/txodds/rancher-secrets-manager/pkg/api/v1alpha1"
	"github.com/txodds/rancher-secrets-manager/pkg/controller/managedsecret"
	"github.com/txodds/rancher-secrets-manager/pkg/rancher"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(secretsv1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		rancherURL           string
		insecureTLS          bool
		caBundle             string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint bind address.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe endpoint bind address.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&rancherURL, "rancher-url", "", "Base URL of the Rancher API (e.g. https://rancher.cattle-system.svc).")
	flag.BoolVar(&insecureTLS, "insecure-tls", false, "Skip TLS verification when connecting to Rancher. Use only in development.")
	flag.StringVar(&caBundle, "ca-bundle", "", "Path to a PEM CA bundle to trust for the Rancher endpoint.")

	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)
	setupLog := ctrl.Log.WithName("setup")

	if rancherURL == "" {
		setupLog.Error(nil, "--rancher-url is required")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "rancher-secrets-manager.cattle.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	rancherClient, err := rancher.NewClient(rancher.Config{
		RancherURL:   rancherURL,
		InsecureTLS:  insecureTLS,
		CABundlePath: caBundle,
	}, mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create Rancher client")
		os.Exit(1)
	}

	if err := (&managedsecret.Reconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		RancherClient: rancherClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller")
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

	setupLog.Info("starting manager", "rancherURL", rancherURL)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
