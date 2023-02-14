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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
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
	return r.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateUpdate(_ runtime.Object) error {
	authproxyworkloadlog.Info("validate update", "name", r.Name)
	return r.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateDelete() error {
	authproxyworkloadlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *AuthProxyWorkload) Validate() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateLabelName(r.Name, field.NewPath("metadata", "name"))...)
	allErrs = append(allErrs, validateSpec(&r.Spec, field.NewPath("spec"))...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{
			Group: GroupVersion.Group,
			Kind:  "AuthProxyWorkload"},
		r.Name, allErrs)
}

func validateSpec(spec *AuthProxyWorkloadSpec, f *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateWorkload(&spec.Workload, f.Child("workload"))...)

	// TODO: Validate the other fields in spec
	// allErrs = append(allErrs, validateAuthProxyContainer(spec.AuthProxyContainer, f.Child("authProxyContainer"))...)
	// allErrs = append(allErrs, validateAuthentication(spec.Authentication, f.Child("authentication"))...)
	// allErrs = append(allErrs, validateAuthentication(spec.Instances, f.Child("instances"))...)

	return allErrs
}

var supportedKinds = []string{"CronJob", "Job", "StatefulSet", "Deployment", "DaemonSet", "ReplicaSet", "Pod"}

func validateWorkload(spec *WorkloadSelectorSpec, f *field.Path) field.ErrorList {
	var errs field.ErrorList
	if spec.Selector != nil {
		verr := validation.ValidateLabelSelector(spec.Selector, validation.LabelSelectorValidationOptions{}, f.Child("selector"))
		errs = append(errs, verr...)
	}

	if spec.Name != "" && spec.Selector != nil {
		errs = append(errs, field.Invalid(f.Child("name"), spec,
			"WorkloadSelectorSpec must specify either name or selector. Both were set."))
	}
	if spec.Name == "" && spec.Selector == nil {
		errs = append(errs, field.Invalid(f.Child("name"), spec,
			"WorkloadSelectorSpec must specify either name or selector. Neither was set."))
	}

	_, gk := schema.ParseKindArg(spec.Kind)
	var found bool
	for _, kind := range supportedKinds {
		if kind == gk.Kind {
			found = true
			break
		}
	}
	if !found {
		errs = append(errs, field.Invalid(f.Child("kind"), spec.Kind,
			fmt.Sprintf("Kind was %q, must be one of CronJob, Job, StatefulSet, Deployment, DaemonSet or Pod", gk.Kind)))

	}

	return errs
}
