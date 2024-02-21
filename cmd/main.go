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

package main //nolint:cyclop // main function needs to initialize many objects

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kapiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kutilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	kcontrollerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	klogzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	controllercache "github.com/kyma-project/nats-manager/internal/controller/cache"
	nmcontroller "github.com/kyma-project/nats-manager/internal/controller/nats"
	"github.com/kyma-project/nats-manager/pkg/env"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
)

const defaultMetricsPort = 9443

func main() { //nolint:funlen // main function needs to initialize many objects
	scheme := runtime.NewScheme()
	setupLog := kcontrollerruntime.Log.WithName("setup")
	kutilruntime.Must(kscheme.AddToScheme(scheme))
	kutilruntime.Must(nmapiv1alpha1.AddToScheme(scheme))

	// get configs from ENV
	envConfigs, err := env.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to get configs from env")
		os.Exit(1)
	}

	// get configs from command-line args
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

	// setup k8s ctrl logger
	logLevel, err := zapcore.ParseLevel(envConfigs.LogLevel)
	if err != nil {
		setupLog.Error(err, "unable to parse log level")
		os.Exit(1)
	}
	opts := klogzap.Options{
		Development: false,
		Level:       logLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	kcontrollerruntime.SetLogger(klogzap.New(klogzap.UseFlagOptions(&opts)))

	// setup logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")
	loggerConfig.Level = zap.NewAtomicLevelAt(logLevel)

	logger, err := loggerConfig.Build()
	if err != nil {
		setupLog.Error(err, "unable to setup logger")
		os.Exit(1)
	}
	sugaredLogger := logger.Sugar()

	// setup ctrl manager
	mgr, err := kcontrollerruntime.NewManager(kcontrollerruntime.GetConfigOrDie(), kcontrollerruntime.Options{
		Scheme:                 scheme,
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
		Metrics:       server.Options{BindAddress: metricsAddr},
		WebhookServer: webhook.NewServer(webhook.Options{Port: metricsPort}),
		NewCache:      controllercache.New,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// create helmRenderer
	helmRenderer, err := chart.NewHelmRenderer(envConfigs.NATSChartDir, sugaredLogger)
	if err != nil {
		setupLog.Error(err, "failed to create new helm client")
		os.Exit(1)
	}

	// init custom kube client wrapper
	apiClientSet, err := kapiextclientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to create new k8s clientset")
		os.Exit(1)
	}

	kubeClient := k8s.NewKubeClient(mgr.GetClient(), apiClientSet, "nats-manager")

	natsManager := manager.NewNATSManger(kubeClient, helmRenderer, sugaredLogger)

	// create NATS reconciler instance
	natsReconciler := nmcontroller.NewReconciler(
		mgr.GetClient(),
		kubeClient,
		helmRenderer,
		mgr.GetScheme(),
		sugaredLogger,
		mgr.GetEventRecorderFor("nats-manager"),
		natsManager,
		&nmapiv1alpha1.NATS{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      envConfigs.NATSCRName,
				Namespace: envConfigs.NATSCRNamespace,
			},
		},
	)

	if err = (natsReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NATS")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(kcontrollerruntime.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
