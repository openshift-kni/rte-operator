/*
Copyright 2021.

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

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	topologyexporterv1alpha1 "github.com/fromanirh/rte-operator/api/v1alpha1"
	"github.com/fromanirh/rte-operator/controllers"
	"github.com/fromanirh/rte-operator/pkg/images"
	"github.com/go-logr/logr"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform/detect"
	"github.com/k8stopologyawareschedwg/deployer/pkg/tlog"

	apimanifests "github.com/k8stopologyawareschedwg/deployer/pkg/manifests/api"
	rtemanifests "github.com/k8stopologyawareschedwg/deployer/pkg/manifests/rte"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(topologyexporterv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var platformName string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&platformName, "P", "", "platform to deploy on - leave empty to autodetect")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// if it is unknown, it's fine
	userPlatform, _ := platform.FromString(platformName)
	plat := detectPlatform(setupLog, userPlatform)
	clusterPlatform := plat.Discovered
	if clusterPlatform == platform.Unknown {
		err := fmt.Errorf("cannot autodetect the platform, and no platform given")
		setupLog.Error(err, "unable to setup")
		os.Exit(1)
	}
	setupLog.Info("detected cluster", "platform", clusterPlatform)

	apiManifests, err := apimanifests.GetManifests(clusterPlatform)
	if err != nil {
		setupLog.Error(err, "unable to load the API manifests")
		os.Exit(1)
	}
	setupLog.Info("API manifests loaded")

	rteManifests, err := rtemanifests.GetManifests(clusterPlatform)
	if err != nil {
		setupLog.Error(err, "unable to load the RTE manifests")
		os.Exit(1)
	}
	setupLog.Info("RTE manifests loaded")

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0e2a6bd3.openshift-kni.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	imageSpec, err := images.GetCurrentImage(mgr.GetClient(), context.Background())
	if err != nil {
		// intentionally continue
		setupLog.Info("unable to find current image, using hardcoded", "error", err)
	}
	setupLog.Info("using RTE image", "spec", imageSpec)

	if err = (&controllers.ResourceTopologyExporterReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		Log:          ctrl.Log.WithName("controllers").WithName("RTE"),
		APIManifests: apiManifests,
		RTEManifests: rteManifests,
		Platform:     clusterPlatform,
		Helper:       deployer.NewHelperWithClient(mgr.GetClient(), "", tlog.NewNullLogAdapter()),
		ImageSpec:    imageSpec,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ResourceTopologyExporter")
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

type detectionOutput struct {
	AutoDetected platform.Platform `json:"auto_detected"`
	UserSupplied platform.Platform `json:"user_supplied"`
	Discovered   platform.Platform `json:"discovered"`
}

func detectPlatform(debugLog logr.Logger, userSupplied platform.Platform) detectionOutput {
	do := detectionOutput{
		AutoDetected: platform.Unknown,
		UserSupplied: userSupplied,
		Discovered:   platform.Unknown,
	}

	if do.UserSupplied != platform.Unknown {
		debugLog.Info("user-supplied", "platform", do.UserSupplied)
		do.Discovered = do.UserSupplied
		return do
	}

	dp, err := detect.Detect()
	if err != nil {
		debugLog.Error(err, "failed to detect the platform")
		return do
	}

	debugLog.Info("auto-detected", "platform", dp)
	do.AutoDetected = dp
	do.Discovered = do.AutoDetected
	return do
}
