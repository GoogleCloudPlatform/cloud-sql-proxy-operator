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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-cloudsql-cloud-google-com-v1alpha1-authproxyworkload,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1alpha1,name=vauthproxyworkload.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &AuthProxyWorkload{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateCreate() error {
	authproxyworkloadlog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateUpdate(old runtime.Object) error {
	authproxyworkloadlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateDelete() error {
	authproxyworkloadlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *AuthProxyWorkload) validate() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateSpec(field.NewPath("spec"), r.Spec)...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{
			Group: GroupVersion.Group,
			Kind:  "AuthProxyWorkload"},
		r.Name, allErrs)
}

func validateSpec(f *field.Path, spec *AuthProxyWorkloadSpec) field.ErrorList {
	var allErrs field.ErrorList

	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	allErrs = append(allErrs, validateWorkload(f.Child("workload"), spec.Workload)...)

	// TODO: Validate the other fields in spec
	return allErrs
}

func validateWorkload(f *field.Path, spec WorkloadSelectorSpec) field.ErrorList {
	var errs field.ErrorList
	if spec.Name != "" && spec.Selector != nil {
		errs = append(errs, field.Invalid(f, spec,
			"WorkloadSelectorSpec must specify either name or selector. Both were set."))
	}
	if spec.Name == "" && spec.Selector == nil {
		errs = append(errs, field.Invalid(f, spec,
			"WorkloadSelectorSpec must specify either name or selector. Neither was set."))
	}

	_, gv := schema.ParseKindArg(spec.Kind)
	if gv.Kind != "CronJob" && gv.Kind != "Job" && gv.Kind != "StatefulSet" &&
		gv.Kind != "Deployment" && gv.Kind != "DaemonSet" && gv.Kind != "Pod" {
		errs = append(errs, field.Invalid(f.Child("kind"), spec,
			"Kind must be one of CronJob, Job, StatefulSet, Deployment, DaemonSet or Pod"))

	}

	return errs
}
