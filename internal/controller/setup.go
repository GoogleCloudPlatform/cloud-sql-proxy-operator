// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var setupLog = ctrl.Log.WithName("setup")

// InitScheme was moved out of ../main.go to here so that it can be invoked
// from the testintegration tests AND from the actual operator.
func InitScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(cloudsqlapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// SetupManagers was moved out of ../main.go here so that it can be invoked
// from the testintegration tests AND from the actual operator.
func SetupManagers(mgr manager.Manager, userAgent, defaultProxyImage string) error {
	u := workload.NewUpdater(userAgent, defaultProxyImage)

	setupLog.Info("Configuring reconcilers...")
	var err error

	_, err = NewAuthProxyWorkloadReconciler(mgr, u)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AuthProxyWorkload")
		// Kubebuilder won't properly write the contents of this file. It will want
		// to do os.Exit(1) on error here.. But it's a bad idea because it will
		// disrupt the testintegration tests to exit 1. Instead, this should
		// return the error and let the caller deal with it.
		return err
	}

	wh := &cloudsqlapi.AuthProxyWorkload{}
	err = wh.SetupWebhookWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AuthProxyWorkload")
		return err
	}

	//+kubebuilder:scaffold:builder
	// kubebuilder to scaffold additional controller here.
	// When kubebuilder scaffolds a new controller here, please
	// adjust the code so it follows the pattern above.

	err = RegisterPodWebhook(mgr, u)
	if err != nil {
		setupLog.Error(err, "unable to create workload admission webhook controller")
		return err
	}

	// Register the podDeleteController, which will listen for pod changes and
	// delete misconfigured pods that in a waiting or error state.
	_, err = newPodDeleteController(mgr, u)
	if err != nil {
		setupLog.Error(err, "unable to create pod informer")
		return err
	}

	// Add the runnable task that will upgrade the proxy image on workloads with
	// default container image when  the operator first starts.
	err = mgr.Add(&upgradeDefaultProxyOnStartup{c: mgr.GetClient()})
	if err != nil {
		setupLog.Error(err, "unable to start task to check all AuthProxyWorkloads on startup")
		return err
	}

	setupLog.Info("Configuring reconcilers complete.")
	return nil
}

// RegisterPodWebhook register the webhook to mutate pods
func RegisterPodWebhook(mgr ctrl.Manager, u *workload.Updater) error {

	mgr.GetWebhookServer().Register("/mutate-pods", &webhook.Admission{
		Handler: &PodAdmissionWebhook{
			Client:  mgr.GetClient(),
			updater: u,
			decoder: admission.NewDecoder(mgr.GetScheme()),
		}})

	return nil
}
