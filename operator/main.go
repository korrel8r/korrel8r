// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	_ "k8s.io/client-go/plugin/pkg/client/auth" // Import all Kubernetes client auth plugins.

	"github.com/go-logr/stdr"
	"github.com/korrel8r/korrel8r/operator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

func fatalIf(bad bool, format string, args ...any) {
	if bad {
		fmt.Fprintf(os.Stderr, format, args...)
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
}

func main() {
	// Environment variables
	defaultVerbose := 0
	if s := os.Getenv(controllers.VerboseEnv); s != "" {
		n, err := strconv.Atoi(s)
		fatalIf(err != nil, "Invalid environment variable: %v=%v", controllers.VerboseEnv, s)
		defaultVerbose = n
	}
	image := os.Getenv(controllers.ImageEnv)
	fatalIf(image == "", "Missing environment variable: %v", controllers.ImageEnv)

	// Command line flags.
	verbose := flag.Int("verbose", defaultVerbose, "Logging verbosity")
	metricsAddr := flag.String("metrics-bind-address", ":9090", "The address the metric endpoint binds to.")
	probeAddr := flag.String("health-probe-bind-address", ":9091", "The address the probe endpoint binds to.")
	flag.Parse()

	stdr.SetVerbosity(*verbose)
	log := stdr.New(nil).WithName(controllers.ApplicationName)
	ctrl.SetLogger(log)

	check := func(err error, msg string) { fatalIf(err != nil, "%v, %v", msg, err) }

	scheme := runtime.NewScheme()
	check(controllers.AddToScheme(scheme), "Cannot add controller types to scheme")
	mgr, err := manager.New(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: *metricsAddr},
		HealthProbeBindAddress: *probeAddr,
		Logger:                 log,
		Cache:                  controllers.CacheOptions(),
	})
	check(err, "Unable to start manager")

	kr := controllers.NewKorrel8rReconciler(
		image,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor(controllers.ApplicationName))
	check(kr.SetupWithManager(mgr), "Unable to create controller")
	check(mgr.AddHealthzCheck("healthz", healthz.Ping), "Unable to set up health check")
	check(mgr.AddReadyzCheck("readyz", healthz.Ping), "Unable to set up ready check")

	log.Info("Starting controller manager")
	check(mgr.Start(ctrl.SetupSignalHandler()), "Problem starting manager")
}
