package main

import (
	"flag"
	"log"
	"os"

	"github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/controllers"
	"github.com/dougbtv/krang/pkg/logging"

	"github.com/go-logr/stdr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&logLevel, "log-level", "debug", "Set log level: debug, verbose, error, panic.")
	flag.Parse()

	// Initialize logger
	logging.SetLogLevel(logLevel)
	logging.SetLogStderr(true)
	logging.Debugf("Starting krangd node-local daemon")

	ctrl.SetLogger(stdr.New(log.New(os.Stderr, "", log.LstdFlags)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
			// Port:        9443,
		},
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "krangd-leader-election.k8s.cni.cncf.io",
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		logging.Panicf("Unable to start manager: %v", err)
		os.Exit(1)
	}

	if err = (&controllers.CNIPluginRegistrationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logging.Panicf("Unable to create controller: %v", err)
		os.Exit(1)
	}

	if err = (&controllers.CNIMutationRequestReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		LocalNodeName: os.Getenv("NODE_NAME"),
	}).SetupWithManager(mgr); err != nil {
		logging.Panicf("Unable to create mutation controller: %v", err)
		os.Exit(1)
	}

	logging.Verbosef("Controller setup complete, starting manager loop")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logging.Panicf("Problem running manager: %v", err)
		os.Exit(1)
	}
}
