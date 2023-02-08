// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var authproxyworkloadlog = logf.Log.WithName("authproxyworkload-resource")

func (r *AuthProxyWorkload) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-cloudsql-cloud-google-com-v1alpha1-authproxyworkload,mutating=true,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1alpha1,name=mauthproxyworkload.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &AuthProxyWorkload{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AuthProxyWorkload) Default() {
	authproxyworkloadlog.Info("default", "name", r.Name)
	if r.Spec.AuthProxyContainer != nil &&
		r.Spec.AuthProxyContainer.RolloutStrategy == "" {
		r.Spec.AuthProxyContainer.RolloutStrategy = WorkloadStrategy
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-cloudsql-cloud-google-com-v1alpha1-authproxyworkload,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1alpha1,name=vauthproxyworkload.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &AuthProxyWorkload{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateCreate() error {
	authproxyworkloadlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateUpdate(_ runtime.Object) error {
	authproxyworkloadlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateDelete() error {
	authproxyworkloadlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
