package main

import (
	"flag"
	"log"
	"os"

	"github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/controllers"
	"github.com/dougbtv/krang/pkg/logging"

	"github.com/go-logr/stdr"
	netdefclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
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
	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&logLevel, "log-level", "debug", "Set log level: debug, verbose, error, panic.")
	flag.Parse()

	// Initialize logger
	logging.SetLogLevel(logLevel)
	logging.SetLogStderr(true)
	logging.Debugf("Starting krangd")
	ctrl.SetLogger(stdr.New(log.New(os.Stderr, "", log.LstdFlags)))

	cfg := ctrl.GetConfigOrDie()
	// ctx := context.Background()

	// Create Net-attach-def client
	netDefClient, err := netdefclient.NewForConfig(cfg)
	if err != nil {
		logging.Panicf("Unable to create NetDef client: %v", err)
		os.Exit(1)
	}

	// --- Leader-only Manager (validation controller) ---
	leaderMgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		LeaderElection:         true,
		LeaderElectionID:       "krangd-leader-election.k8s.cni.cncf.io",
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		logging.Panicf("Unable to start leader manager: %v", err)
		os.Exit(1)
	}

	if err = (&controllers.CNIValidationReconciler{
		Client:       leaderMgr.GetClient(),
		Scheme:       leaderMgr.GetScheme(),
		NetDefClient: netDefClient,
	}).SetupWithManager(leaderMgr); err != nil {
		logging.Panicf("Unable to create validation controller: %v", err)
		os.Exit(1)
	}

	// --- Daemon-style Manager (runs on every node) ---
	daemonMgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // disable metrics endpoint for daemon manager
		},
		LeaderElection: false,
	})
	if err != nil {
		logging.Panicf("Unable to start daemon manager: %v", err)
		os.Exit(1)
	}

	if err = (&controllers.CNIPluginRegistrationReconciler{
		Client: daemonMgr.GetClient(),
		Scheme: daemonMgr.GetScheme(),
	}).SetupWithManager(daemonMgr); err != nil {
		logging.Panicf("Unable to create plugin controller: %v", err)
		os.Exit(1)
	}

	if err = (&controllers.CNIMutationRequestReconciler{
		Client:        daemonMgr.GetClient(),
		Scheme:        daemonMgr.GetScheme(),
		LocalNodeName: os.Getenv("NODE_NAME"),
	}).SetupWithManager(daemonMgr); err != nil {
		logging.Panicf("Unable to create mutation controller: %v", err)
		os.Exit(1)
	}

	signalHandler := ctrl.SetupSignalHandler()

	go func() {
		logging.Verbosef("Starting leader-only controller loop")
		if err := leaderMgr.Start(signalHandler); err != nil {
			logging.Panicf("Problem running leader manager: %v", err)
			os.Exit(1)
		}
	}()

	logging.Verbosef("Starting daemon controller loop")
	if err := daemonMgr.Start(signalHandler); err != nil {
		logging.Panicf("Problem running daemon manager: %v", err)
		os.Exit(1)
	}
}
