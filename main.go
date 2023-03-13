/*
Copyright 2022.

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
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/ocm-agent-operator/controllers/fleetnotification"
	"github.com/openshift/ocm-agent-operator/pkg/localmetrics"
	"github.com/openshift/ocm-agent-operator/pkg/ocmagenthandler"
	"github.com/openshift/ocm-agent-operator/pkg/util/namespace"
	"github.com/openshift/ocm-agent-operator/pkg/version"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	zaplogfmt "github.com/sykesm/zap-logfmt"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	oconfigv1 "github.com/openshift/api/config/v1"
	monitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	// OSD metrics
	osdmetrics "github.com/openshift/operator-custom-metrics/pkg/metrics"

	ocmagentmanagedopenshiftiov1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	"github.com/openshift/ocm-agent-operator/controllers/ocmagent"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = apiruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var (
	osdMetricsPort = "8686"
	osdMetricsPath = "/metrics"
)

var log = logf.Log.WithName("cmd")

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ocmagentmanagedopenshiftiov1alpha1.AddToScheme(scheme))
	utilruntime.Must(oconfigv1.Install(scheme))
	utilruntime.Must(monitorv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: v%v", version.SDKVersion))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Add a custom logger to log in RFC3339 format instead of unix epoch time format
	configLog := uzap.NewProductionEncoderConfig()
	configLog.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339Nano))
	}
	logfmtEncoder := zaplogfmt.NewEncoder(configLog)
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout), zap.Encoder(logfmtEncoder))
	logf.SetLogger(logger)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	printVersion()

	operatorNS, err := namespace.GetOperatorNamespace()
	if err != nil {
		setupLog.Error(err, "unable to determine operator namespace, please define OPERATOR_NAMESPACE")
		os.Exit(1)
	}

	if err := monitorv1.AddToScheme(clientgoscheme.Scheme); err != nil {
		setupLog.Error(err, "unable to add monitoringv1 scheme")
		os.Exit(1)
	}

	metricsServer := osdmetrics.NewBuilder(operatorNS, "ocm-agent-operator").
		WithPort(osdMetricsPort).
		WithPath(osdMetricsPath).
		WithCollectors(localmetrics.MetricsList).
		WithServiceMonitor().
		GetConfig()

	if err := osdmetrics.ConfigureMetrics(context.TODO(), *metricsServer); err != nil {
		setupLog.Error(err, "Failed to configure OSD metrics")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace:              operatorNS,
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8716512f.managed.openshift.io",
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create a separate client for the OAH Builder
	kubeConfig := ctrl.GetConfigOrDie()
	handlerClient, err := client.New(kubeConfig, client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		os.Exit(1)
	}

	if err = (&ocmagent.OcmAgentReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		OCMAgentHandlerBuilder: ocmagenthandler.NewBuilder(handlerClient),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OcmAgent")
		os.Exit(1)
	}
	if err = (&fleetnotification.ManagedFleetNotificationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ManagedFleetNotification")
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
