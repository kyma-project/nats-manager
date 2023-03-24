/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	k8szap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natscontroller "github.com/kyma-project/nats-manager/internal/controller/nats"
)

const defaultMetricsPort = 9443

func main() {
	scheme := runtime.NewScheme()
	setupLog := ctrl.Log.WithName("setup")
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(natsv1alpha1.AddToScheme(scheme))

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var leaderElectionID string
	var metricsPort int
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leaderElectionID", "26479083.kyma-project.io",
		"ID for the controller leader election.")
	flag.IntVar(&metricsPort, "metricsPort", defaultMetricsPort, "Port number for metrics endpoint.")

	opts := k8szap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// @TODO: Re-check logger setup and init
	ctrl.SetLogger(k8szap.New(k8szap.UseFlagOptions(&opts)))

	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")

	logger, err := loggerConfig.Build()
	if err != nil {
		setupLog.Error(err, "unable to setup logger")
		os.Exit(1)
	}

	sugaredLogger := logger.Sugar()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   metricsPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// create helmRenderer
	const repoDir = "/charts/nats"
	helmRenderer, err := chart.NewHelmRenderer(repoDir, sugaredLogger)
	if err != nil {
		setupLog.Error(err, "failed to create new helm client")
		os.Exit(1)
	}

	// init custom kube client wrapper
	kubeClient := k8s.NewKubeClient(mgr.GetClient(), "nats-manager")

	natsManager := manager.NewNATSManger(kubeClient, helmRenderer, sugaredLogger)

	// create NATS reconciler instance
	natsReconciler := natscontroller.NewReconciler(
		mgr.GetClient(),
		kubeClient,
		helmRenderer,
		mgr.GetScheme(),
		sugaredLogger,
		mgr.GetEventRecorderFor("nats-manager"),
		natsManager,
	)

	if err = (natsReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Nats")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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
