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

	"cloud.google.com/go/cloudsqlconn/instance"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var authproxyworkloadlog = logf.Log.WithName("authproxyworkload-resource")

func (r *AuthProxyWorkload) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(&AuthProxyWorkloadDefaulter{}).
		WithValidator(&AuthProxyWorkloadValidator{Client: mgr.GetClient()}).
		Complete()

}

// +kubebuilder:object:generate=false
// +kubebuilder:webhook:path=/mutate-cloudsql-cloud-google-com-v1-authproxyworkload,mutating=true,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1,name=mauthproxyworkload.kb.io,admissionReviewVersions=v1
type AuthProxyWorkloadDefaulter struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (*AuthProxyWorkloadDefaulter) Default(_ context.Context, r *AuthProxyWorkload) error {
	authproxyworkloadlog.Info("default", "name", r.Name)
	if r.Spec.AuthProxyContainer != nil &&
		r.Spec.AuthProxyContainer.RolloutStrategy == "" {
		r.Spec.AuthProxyContainer.RolloutStrategy = WorkloadStrategy
	}
	return nil
}

// +kubebuilder:object:generate=false
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:webhook:path=/validate-cloudsql-cloud-google-com-v1-authproxyworkload,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=create;update,versions=v1,name=vauthproxyworkload.kb.io,admissionReviewVersions=v1
type AuthProxyWorkloadValidator struct {
	Client client.Client
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *AuthProxyWorkloadValidator) ValidateCreate(ctx context.Context, r *AuthProxyWorkload) (warnings admission.Warnings, err error) {
	allErrs := r.validate()
	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "AuthProxyWorkload"},
			r.Name, allErrs)
	}
	if err := v.validateAuthorization(ctx, r); err != nil {
		return nil, err
	}
	return nil, nil

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *AuthProxyWorkloadValidator) ValidateUpdate(ctx context.Context, old, newObj *AuthProxyWorkload) (warnings admission.Warnings, err error) {
	allErrs := newObj.validate()
	allErrs = append(allErrs, newObj.validateUpdateFrom(old)...)
	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "AuthProxyWorkload"},
			newObj.Name, allErrs)
	}
	if !reflect.DeepEqual(old.Spec, newObj.Spec) {
		if err := v.validateAuthorization(ctx, newObj); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *AuthProxyWorkloadValidator) ValidateDelete(_ context.Context, _ *AuthProxyWorkload) (admission.Warnings, error) {
	return nil, nil
}

func (v *AuthProxyWorkloadValidator) validateAuthorization(ctx context.Context, r *AuthProxyWorkload) error {
	if v.Client == nil {
		return nil
	}
	req, err := admission.RequestFromContext(ctx)
	if err != nil || req.UserInfo.Username == "" {
		return nil
	}

	// 1. Pod delete check: User must be authorized to delete pods in the target namespace.
	sarPod := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   req.UserInfo.Username,
			UID:    req.UserInfo.UID,
			Groups: req.UserInfo.Groups,
			Extra:  convertExtra(req.UserInfo.Extra),
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: r.Namespace,
				Verb:      "delete",
				Group:     "",
				Resource:  "pods",
			},
		},
	}
	if err := v.Client.Create(ctx, sarPod); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("failed to check authorization for deleting pods: %w", err))
	}
	if !sarPod.Status.Allowed {
		return apierrors.NewForbidden(
			schema.GroupResource{Group: GroupVersion.Group, Resource: "authproxyworkloads"},
			r.Name,
			fmt.Errorf("user %q is not authorized to delete pods in namespace %q", req.UserInfo.Username, r.Namespace),
		)
	}

	// 2. Workload update and patch checks: User must be authorized to update and patch the targeted workload.
	group, resource, err := groupResourceForKind(r.Spec.Workload.Kind)
	if err != nil {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "AuthProxyWorkload"},
			r.Name,
			field.ErrorList{field.Invalid(field.NewPath("spec", "workload", "kind"), r.Spec.Workload.Kind, err.Error())},
		)
	}

	if r.Spec.Workload.Name != "" {
		for _, verb := range []string{"update", "patch"} {
			sar := &authorizationv1.SubjectAccessReview{
				Spec: authorizationv1.SubjectAccessReviewSpec{
					User:   req.UserInfo.Username,
					UID:    req.UserInfo.UID,
					Groups: req.UserInfo.Groups,
					Extra:  convertExtra(req.UserInfo.Extra),
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace: r.Namespace,
						Verb:      verb,
						Group:     group,
						Resource:  resource,
						Name:      r.Spec.Workload.Name,
					},
				},
			}
			if err := v.Client.Create(ctx, sar); err != nil {
				return apierrors.NewInternalError(fmt.Errorf("failed to check authorization for %s on %s %s: %w", verb, resource, r.Spec.Workload.Name, err))
			}
			if !sar.Status.Allowed {
				return apierrors.NewForbidden(
					schema.GroupResource{Group: GroupVersion.Group, Resource: "authproxyworkloads"},
					r.Name,
					fmt.Errorf("user %q is not authorized to %s %s %q in namespace %q", req.UserInfo.Username, verb, resource, r.Spec.Workload.Name, r.Namespace),
				)
			}
		}
	} else if r.Spec.Workload.Selector != nil {
		for _, verb := range []string{"update", "patch"} {
			sar := &authorizationv1.SubjectAccessReview{
				Spec: authorizationv1.SubjectAccessReviewSpec{
					User:   req.UserInfo.Username,
					UID:    req.UserInfo.UID,
					Groups: req.UserInfo.Groups,
					Extra:  convertExtra(req.UserInfo.Extra),
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace: r.Namespace,
						Verb:      verb,
						Group:     group,
						Resource:  resource,
					},
				},
			}
			if err := v.Client.Create(ctx, sar); err != nil {
				return apierrors.NewInternalError(fmt.Errorf("failed to check authorization for %s on %s: %w", verb, resource, err))
			}
			if !sar.Status.Allowed {
				return apierrors.NewForbidden(
					schema.GroupResource{Group: GroupVersion.Group, Resource: "authproxyworkloads"},
					r.Name,
					fmt.Errorf("user %q is not authorized to %s %s by label selector in namespace %q (requires namespace-wide update and patch permissions on %s)", req.UserInfo.Username, verb, resource, r.Namespace, resource),
				)
			}
		}
	}

	// 3. Custom container override check
	if r.Spec.AuthProxyContainer != nil && r.Spec.AuthProxyContainer.Container != nil {
		sarOverride := &authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User:   req.UserInfo.Username,
				UID:    req.UserInfo.UID,
				Groups: req.UserInfo.Groups,
				Extra:  convertExtra(req.UserInfo.Extra),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace:   r.Namespace,
					Verb:        "create",
					Group:       GroupVersion.Group,
					Resource:    "authproxyworkloads",
					Subresource: "containeroverride",
					Name:        r.Name,
				},
			},
		}
		if err := v.Client.Create(ctx, sarOverride); err != nil {
			return apierrors.NewInternalError(fmt.Errorf("failed to check authorization for container override: %w", err))
		}
		if !sarOverride.Status.Allowed {
			return apierrors.NewForbidden(
				schema.GroupResource{Group: GroupVersion.Group, Resource: "authproxyworkloads"},
				r.Name,
				fmt.Errorf("user %q is not authorized to specify custom container override (requires containeroverride permission on authproxyworkloads)", req.UserInfo.Username),
			)
		}
	}

	return nil
}

func convertExtra(in map[string]authenticationv1.ExtraValue) map[string]authorizationv1.ExtraValue {
	if in == nil {
		return nil
	}
	out := make(map[string]authorizationv1.ExtraValue, len(in))
	for k, v := range in {
		out[k] = authorizationv1.ExtraValue(v)
	}
	return out
}

func groupResourceForKind(kindArg string) (string, string, error) {
	_, gk := schema.ParseKindArg(kindArg)
	switch gk.Kind {
	case "Deployment":
		return "apps", "deployments", nil
	case "StatefulSet":
		return "apps", "statefulsets", nil
	case "DaemonSet":
		return "apps", "daemonsets", nil
	case "ReplicaSet":
		return "apps", "replicasets", nil
	case "Job":
		return "batch", "jobs", nil
	case "CronJob":
		return "batch", "cronjobs", nil
	case "Pod":
		return "", "pods", nil
	default:
		return "", "", fmt.Errorf("unsupported kind %q", kindArg)
	}
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
		errs = append(errs, validateConnectionString(inst, f)...)
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

func validateConnectionString(inst InstanceSpec, f *field.Path) field.ErrorList {
	if instance.IsValidDomain(inst.ConnectionString) {
		return nil
	}
	if _, err := instance.ParseConnName(inst.ConnectionString); err != nil {
		return []*field.Error{field.Invalid(f, inst.ConnectionString, "is not a valid instance connection name or dns name")}
	}
	return nil
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
