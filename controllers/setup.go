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

package controllers

import (
	cloudsqlv1alpha1 "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const NeedsUpdateAnnotation = "cloudsql.cloud.google.com/authproxyworkload-needs-update"
const WasUpdatedAnnotation = "cloudsql.cloud.google.com/authproxyworkload-was-updated"

var (
	setupLog = ctrl.Log.WithName("setup")
)

// InitScheme was moved out of ../main.go to here so that it can be invoked
// from the integration tests AND from the actual operator.

func InitScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(cloudsqlv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// SetupManagers was moved out of ../main.go to here so that it can be invoked
// from the integration tests AND from the actual operator.

func SetupManagers(mgr manager.Manager) error {
	setupLog.Info("Configuring reconcilers...")
	var err error

	if err = (&AuthProxyWorkloadReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AuthProxyWorkload")
		// Kubebuilder won't properly write the contents of this file. It will want
		// to do os.Exit(1) on error here.. But it's a bad idea because it will
		// disrupt the integration tests to exit 1. Instead, this should
		// return the error and let the caller deal with it.
		return err
	}
	if err = (&cloudsqlv1alpha1.AuthProxyWorkload{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AuthProxyWorkload")
		return err

	}
	//+kubebuilder:scaffold:builder

	setupLog.Info("Configuring reconcilers complete.")
	return nil
}
