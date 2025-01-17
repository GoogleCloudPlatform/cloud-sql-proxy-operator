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

package v1

import (
	"context"
	"fmt"
	"path"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"
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

// +kubebuilder:webhook:path=/mutate-cloudsql-cloud-google-com-v1-authproxyworkload,mutating=true,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1,name=mauthproxyworkload.kb.io,admissionReviewVersions=v1
var _ webhook.CustomDefaulter = &AuthProxyWorkload{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AuthProxyWorkload) Default(_ context.Context, _ runtime.Object) error {
	authproxyworkloadlog.Info("default", "name", r.Name)
	if r.Spec.AuthProxyContainer != nil &&
		r.Spec.AuthProxyContainer.RolloutStrategy == "" {
		r.Spec.AuthProxyContainer.RolloutStrategy = WorkloadStrategy
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-cloudsql-cloud-google-com-v1-authproxyworkload,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1,name=vauthproxyworkload.kb.io,admissionReviewVersions=v1
var _ webhook.CustomValidator = &AuthProxyWorkload{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	allErrs := r.validate()
	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "AuthProxyWorkload"},
			r.Name, allErrs)
	}
	return nil, nil

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateUpdate(_ context.Context, old runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	o, ok := old.(*AuthProxyWorkload)
	if !ok {
		return nil, fmt.Errorf("bad request, expected old to be an AuthProxyWorkload")
	}

	allErrs := r.validate()
	allErrs = append(allErrs, r.validateUpdateFrom(o)...)
	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "AuthProxyWorkload"},
			r.Name, allErrs)
	}
	return nil, nil

}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AuthProxyWorkload) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (r *AuthProxyWorkload) validate() field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateLabelName(r.Name, field.NewPath("metadata", "name"))...)
	allErrs = append(allErrs, validateWorkload(&r.Spec.Workload, field.NewPath("spec", "workload"))...)
	allErrs = append(allErrs, validateInstances(&r.Spec.Instances, field.NewPath("spec", "instances"))...)
	allErrs = append(allErrs, validateContainer(r.Spec.AuthProxyContainer, field.NewPath("spec", "authProxyContainer"))...)

	return allErrs

}

func validateContainer(spec *AuthProxyContainerSpec, f *field.Path) field.ErrorList {
	if spec == nil {
		return nil
	}

	var allErrs field.ErrorList
	if spec.AdminServer != nil {
		if len(spec.AdminServer.EnableAPIs) == 0 {
			allErrs = append(allErrs, field.Invalid(
				f.Child("adminServer", "enableAPIs"), nil,
				"enableAPIs must have at least one valid element: Debug or QuitQuitQuit"))
		}
		for i, v := range spec.AdminServer.EnableAPIs {
			if v != "Debug" && v != "QuitQuitQuit" {
				allErrs = append(allErrs, field.Invalid(
					f.Child("adminServer", "enableAPIs", fmt.Sprintf("%d", i)), v,
					"enableAPIs may contain the values \"Debug\" or \"QuitQuitQuit\""))
			}
		}
	}
	if spec.AdminServer != nil {
		errors := apivalidation.IsValidPortNum(int(spec.AdminServer.Port))
		for _, e := range errors {
			allErrs = append(allErrs, field.Invalid(
				f.Child("adminServer", "port"),
				spec.AdminServer.Port, e))
		}
	}

	return allErrs
}

// validateUpdateFrom checks that an update to an AuthProxyWorkload resource
// adheres to these rules:
// - No changes to the workload selector
// - No changes to the RolloutStrategy
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

// validateWorkload ensures that the WorkloadSelectorSpec follows these rules:
//   - Either Name or Selector is set
//   - Kind is one of the supported kinds: "CronJob", "Job", "StatefulSet",
//     "Deployment", "DaemonSet", "ReplicaSet", "Pod"
//   - Selector is valid according to the k8s validation rules for LabelSelector
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

// validateInstances ensures that InstanceSpec follows these rule:
//   - There is at least 1 InstanceSpec
//   - portEnvName, hostEnvName, and unixSocketPathEnvName have values that adhere
//     to the standard k8s EnvName field validation.
//   - Port has a valid port number according to the standard k8s Port field
//     validation.
//   - UnixSocketPath contains an absolute path.
//   - The configuration clearly specifies either a TCP or a Unix socket but not
//     both.
func validateInstances(spec *[]InstanceSpec, f *field.Path) field.ErrorList {
	var errs field.ErrorList
	if len(*spec) == 0 {
		errs = append(errs, field.Invalid(f,
			nil,
			"at least one database instance must be declared"))
		return errs
	}
	for i, inst := range *spec {
		ff := f.Child(fmt.Sprintf("%d", i))
		if inst.Port != nil {
			for _, s := range apivalidation.IsValidPortNum(int(*inst.Port)) {
				errs = append(errs, field.Invalid(ff.Child("port"), inst.Port, s))
			}
		}
		errs = append(errs, validateEnvName(ff.Child("portEnvName"),
			inst.PortEnvName)...)
		errs = append(errs, validateEnvName(ff.Child("hostEnvName"),
			inst.HostEnvName)...)
		errs = append(errs, validateEnvName(ff.Child("unixSocketPathEnvName"),
			inst.UnixSocketPathEnvName)...)

		if inst.UnixSocketPath != "" && !path.IsAbs(inst.UnixSocketPath) {
			errs = append(errs, field.Invalid(ff.Child("unixSocketPath"),
				inst.UnixSocketPath, "must be an absolute path"))
		}
		if inst.UnixSocketPath != "" && (inst.Port != nil || inst.PortEnvName != "") {
			errs = append(errs, field.Invalid(ff.Child("unixSocketPath"),
				inst.UnixSocketPath,
				"unixSocketPath cannot be set when portEnvName or port are set. Databases can be configured to listen for either TCP or Unix socket connections, not both"))
		}
		if inst.UnixSocketPath == "" && inst.Port == nil && inst.PortEnvName == "" {
			errs = append(errs, field.Invalid(f,
				inst.UnixSocketPath,
				"instance must specify at least one of the following: portEnvName, port, or unixSocketPath"))
		}
	}
	return errs
}

func validateEnvName(f *field.Path, envName string) field.ErrorList {
	var errs field.ErrorList
	if envName != "" {
		for _, s := range apivalidation.IsEnvVarName(envName) {
			errs = append(errs, field.Invalid(f, envName, s))
		}
	}
	return errs
}
