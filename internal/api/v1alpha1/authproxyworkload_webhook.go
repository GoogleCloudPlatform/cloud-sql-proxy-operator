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
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// +kubebuilder:webhook:path=/mutate-cloudsql-cloud-google-com-v1alpha1-authproxyworkload,mutating=true,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1alpha1,name=mauthproxyworkload.kb.io,admissionReviewVersions=v1
var _ webhook.Defaulter = &AuthProxyWorkload{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AuthProxyWorkload) Default() {
	authproxyworkloadlog.Info("default", "name", r.Name)
	if r.Spec.AuthProxyContainer != nil &&
		r.Spec.AuthProxyContainer.RolloutStrategy == "" {
		r.Spec.AuthProxyContainer.RolloutStrategy = WorkloadStrategy
	}
}

// +kubebuilder:webhook:path=/validate-cloudsql-cloud-google-com-v1alpha1-authproxyworkload,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1alpha1,name=vauthproxyworkload.kb.io,admissionReviewVersions=v1
var _ webhook.Validator = &AuthProxyWorkload{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateCreate() error {
	allErrs := r.validate()
	if len(allErrs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "AuthProxyWorkload"},
			r.Name, allErrs)
	}
	return nil

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateUpdate(old runtime.Object) error {
	o, ok := old.(*AuthProxyWorkload)
	if !ok {
		return fmt.Errorf("bad request, expected old to be an AuthProxyWorkload")
	}

	allErrs := r.validate()
	allErrs = append(allErrs, r.validateUpdateFrom(o)...)
	if len(allErrs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "AuthProxyWorkload"},
			r.Name, allErrs)
	}
	return nil

}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateDelete() error {
	return nil
}

func (r *AuthProxyWorkload) validate() field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateLabelName(r.Name, field.NewPath("metadata", "name"))...)
	allErrs = append(allErrs, validateWorkload(&r.Spec.Workload, field.NewPath("spec", "workload"))...)

	return allErrs

}

func (r *AuthProxyWorkload) validateUpdateFrom(op *AuthProxyWorkload) field.ErrorList {
	var allErrs field.ErrorList

	if r.Spec.Workload.Kind != op.Spec.Workload.Kind {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "workload", "kind"), r.Spec.Workload.Kind,
			"kind cannot be changed on update"))
	}
	if r.Spec.Workload.Name != op.Spec.Workload.Name {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "workload", "name"), r.Spec.Workload.Name,
			"kind cannot be changed on update"))
	}
	if selectorNotEqual(r.Spec.Workload.Selector, op.Spec.Workload.Selector) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "workload", "selector"), r.Spec.Workload.Selector,
			"selector cannot be changed on update"))
	}

	allErrs = append(allErrs, validateRolloutStrategyChange(r.Spec.AuthProxyContainer, op.Spec.AuthProxyContainer)...)

	return allErrs
}

// validateRolloutStrategyChange ensures that the rollout strategy does not
// change on update, taking default values into account.
func validateRolloutStrategyChange(c *AuthProxyContainerSpec, oc *AuthProxyContainerSpec) []*field.Error {
	var allErrs field.ErrorList
	var (
		s  = WorkloadStrategy
		os = WorkloadStrategy
	)
	if c != nil && c.RolloutStrategy != "" {
		s = c.RolloutStrategy
	}
	if oc != nil && oc.RolloutStrategy != "" {
		os = oc.RolloutStrategy
	}
	if s != os {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "authProxyContainer", "rolloutStrategy"), s,
			fmt.Sprintf("rolloutStrategy cannot be changed on update from %s", os)))
	}

	return allErrs

}

func selectorNotEqual(s *metav1.LabelSelector, os *metav1.LabelSelector) bool {
	if s == nil && os == nil {
		return false
	}

	if s != nil && os != nil {
		return !reflect.DeepEqual(s, os)
	}

	return true
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
