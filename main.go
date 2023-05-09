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
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/gitlabidentityprovider"
	"github.com/rh-mobb/ocm-operator/controllers/ldapidentityprovider"
	"github.com/rh-mobb/ocm-operator/controllers/machinepool"
	"github.com/rh-mobb/ocm-operator/controllers/rosacluster"
	//+kubebuilder:scaffold:imports
)

const (
	defaultPollerIntervalMinutes = 5
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ocmv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

//nolint:funlen,cyclop
func main() {
	config := controllers.Config{}

	flag.StringVar(&config.MetricsAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&config.ProbeAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&config.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.IntVar(&config.PollerIntervalMinutes, "poller-interval", defaultPollerIntervalMinutes, "Default interval, in minutes, by "+
		"which the controller should reconcile desired state.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     config.MetricsAddress,
		Port:                   9443,
		HealthProbeBindAddress: config.ProbeAddress,
		LeaderElection:         config.EnableLeaderElection,
		LeaderElectionID:       "453df18d.mobb.redhat.com",
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

	// load the token and create the ocm client
	tokenEnvKey := "OCM_TOKEN"
	token, tokenExists := os.LookupEnv(tokenEnvKey)
	if !tokenExists {
		setupLog.Error(err, "unable to load token", "environment variable", tokenEnvKey)
		os.Exit(1)
	}

	// create the connection
	connection, err := sdk.NewConnectionBuilder().
		Tokens(token).
		Build()
	if err != nil {
		setupLog.Error(err, "unable to create ocm client", "environment variable", tokenEnvKey)
		os.Exit(1)
	}

	if err = (&machinepool.Controller{
		Connection: connection,
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("machinepool-controller"),
		Interval:   time.Duration(config.PollerIntervalMinutes) * time.Minute,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MachinePool")
		os.Exit(1)
	}
	if err = (&gitlabidentityprovider.Controller{
		Connection: connection,
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("gitlab-idp-controller"),
		Interval:   time.Duration(config.PollerIntervalMinutes) * time.Minute,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GitLabIdentityProvider")
		os.Exit(1)
	}
	if err = (&ldapidentityprovider.Controller{
		Connection: connection,
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("ldap-idp-controller"),
		Interval:   time.Duration(config.PollerIntervalMinutes) * time.Minute,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LDAPIdentityProvider")
		os.Exit(1)
	}
	if err = (&rosacluster.Controller{
		Connection: connection,
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("rosa-cluster-controller"),
		Interval:   time.Duration(config.PollerIntervalMinutes) * time.Minute,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
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

		if err := connection.Close(); err != nil {
			setupLog.Error(err, "unable to close ocm connection")
		}

		os.Exit(1)
	}
}
