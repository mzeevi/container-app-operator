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
	"os"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	dnsrecordv1alpha1 "github.com/dana-team/provider-dns/apis/record/v1alpha1"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	cappcontroller "github.com/dana-team/container-app-operator/internal/kinds/capp/controllers"
	"github.com/dana-team/container-app-operator/internal/kinds/capp/utils"
	crcontroller "github.com/dana-team/container-app-operator/internal/kinds/capprevision/controllers"
	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	"github.com/go-logr/zapr"
	loggingv1beta1 "github.com/kube-logging/logging-operator/pkg/sdk/logging/api/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	knativev1beta1 "knative.dev/serving/pkg/apis/serving/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	runtimezap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(knativev1.AddToScheme(scheme))
	utilruntime.Must(loggingv1beta1.AddToScheme(scheme))
	utilruntime.Must(knativev1beta1.AddToScheme(scheme))
	utilruntime.Must(cappv1alpha1.AddToScheme(scheme))
	utilruntime.Must(nfspvcv1alpha1.AddToScheme(scheme))
	utilruntime.Must(cmapi.AddToScheme(scheme))
	utilruntime.Must(dnsrecordv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func initOpenshiftSchemes() {
	utilruntime.Must(routev1.Install(scheme))
}

func initEcsLogger() {
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.DebugLevel)
	logger := zap.New(core, zap.AddCaller())
	logf.SetLogger(zapr.NewLogger(logger))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var ecsLogging bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&ecsLogging, "ecs-logging", true, "Display controller logs in ecs format.")

	flag.Parse()

	if ecsLogging {
		initEcsLogger()
	} else {
		ctrl.SetLogger(runtimezap.New())
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c1382367.dana.io",
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	onOpenshift, err := utils.IsOnOpenshift(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to check if we run on Openshift")
		os.Exit(1)
	}

	if onOpenshift {
		initOpenshiftSchemes()
	}

	if err = (&cappcontroller.CappReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		OnOpenshift:   onOpenshift,
		EventRecorder: mgr.GetEventRecorderFor("container-app-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Capp")
		os.Exit(1)
	}

	if err = (&crcontroller.CappRevisionReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("capprevision-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CappRevision")
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
