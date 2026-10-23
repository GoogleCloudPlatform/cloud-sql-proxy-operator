// Copyright 2023 Google LLC
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

package v1_test

import (
	"context"
	"fmt"
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func ptr[T int | int32 | int64 | string | bool](i T) *T {
	return &i
}

func TestAuthProxyWorkload_ValidateCreate_InstanceSpec(t *testing.T) {

	data := []struct {
		desc      string
		spec      []cloudsqlapi.InstanceSpec
		wantValid bool
	}{
		{
			desc:      "Invalid, empty instances",
			wantValid: false,
		},
		{
			desc: "Invalid, Instance configured without PortEnvName, Port, or UnixSocketPath",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
			}},
			wantValid: false,
		},
		{
			desc: "Valid, Instance configured with UnixSocketPath",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				UnixSocketPath:   "/db/socket",
			}},
			wantValid: true,
		},
		{
			desc: "Invalid, Instance configured with UnixSocketPath and Port",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				UnixSocketPath:   "/db/socket",
				Port:             ptr(int32(2443)),
			}},
			wantValid: false,
		},
		{
			desc: "Valid, Instance configured with valid port",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				Port:             ptr(int32(2443)),
			}},
			wantValid: true,
		},
		{
			desc: "Invalid, Instance configured with bad port",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				Port:             ptr(int32(-22)),
			}},
			wantValid: false,
		},
		{
			desc: "Invalid, Instance configured with bad portEnvName",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				PortEnvName:      "22423!",
			}},
			wantValid: false,
		},
		{
			desc: "Invalid, Instance configured with bad hostEnvName",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				HostEnvName:      "22423!",
			}},
			wantValid: false,
		},
		{
			desc: "Invalid, Instance configured with bad UnixSocketPathEnvName",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString:      "proj:region:db2",
				UnixSocketPathEnvName: "22423!",
				UnixSocketPath:        "/db/socket",
			}},
			wantValid: false,
		},
		{
			desc: "Invalid, Instance configured with bad relative UnixSocketPath",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				UnixSocketPath:   "db/socket",
			}},
			wantValid: false,
		},
		{
			desc: "Valid, Instance configured with valid instance name",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				Port:             ptr(int32(5000)),
			}},
			wantValid: true,
		},
		{
			desc: "Valid, Instance configured with domain name name",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				Port:             ptr(int32(5000)),
			}},
			wantValid: true,
		},
		{
			desc: "Invalid, Instance configured with malformed name",
			spec: []cloudsqlapi.InstanceSpec{{
				ConnectionString: "bad name!",
				Port:             ptr(int32(5000)),
			}},
			wantValid: false,
		},
	}
	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec: cloudsqlapi.AuthProxyWorkloadSpec{
					Workload: cloudsqlapi.WorkloadSelectorSpec{
						Kind: "Deployment",
						Name: "webapp",
					},
					Instances: tc.spec,
				},
			}
			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &p)
			_, err := (&cloudsqlapi.AuthProxyWorkloadValidator{}).ValidateCreate(ctx, &p)
			gotValid := err == nil
			switch {
			case tc.wantValid && !gotValid:
				t.Errorf("wants create valid, got error %v", err)
				printFieldErrors(t, err)
			case !tc.wantValid && gotValid:
				t.Errorf("wants an error on create, got no error")
			default:
				t.Logf("create passed %s", tc.desc)
				// test passes, do nothing.
			}
		})
	}

}
func TestAuthProxyWorkload_ValidateCreate_WorkloadSpec(t *testing.T) {
	data := []struct {
		desc      string
		spec      cloudsqlapi.WorkloadSelectorSpec
		wantValid bool
	}{
		{
			desc: "Valid WorkloadSelectorSpec with Name",
			spec: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Deployment",
				Name: "webapp",
			},
			wantValid: true,
		},
		{
			desc: "Valid WorkloadSelectorSpec with Selector",
			spec: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Deployment",
				Selector: &v1.LabelSelector{
					MatchLabels: map[string]string{"app": "sample"},
				},
			},
			wantValid: true,
		},
		{
			desc: "Invalid, both workload selector and name both set",
			spec: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Deployment",
				Name: "webapp",
				Selector: &v1.LabelSelector{
					MatchLabels: map[string]string{"app": "sample"},
				},
			},
			wantValid: false,
		},
		{
			desc:      "Invalid, WorkloadSelector missing name and selector",
			spec:      cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment"},
			wantValid: false,
		},
		{
			desc: "Valid, Instance configured with PortEnvName",
			spec: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Deployment",
				Name: "webapp",
			},
			wantValid: true,
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec: cloudsqlapi.AuthProxyWorkloadSpec{
					Workload: tc.spec,
					Instances: []cloudsqlapi.InstanceSpec{{
						ConnectionString: "proj:region:db2",
						Port:             ptr(int32(2443)),
					}},
				},
			}
			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &p)
			_, err := (&cloudsqlapi.AuthProxyWorkloadValidator{}).ValidateCreate(ctx, &p)
			gotValid := err == nil
			switch {
			case tc.wantValid && !gotValid:
				t.Errorf("wants create valid, got error %v", err)
				printFieldErrors(t, err)
			case !tc.wantValid && gotValid:
				t.Errorf("wants an error on create, got no error")
			default:
				t.Logf("create passed %s", tc.desc)
				// test passes, do nothing.
			}
		})
	}
}
func TestAuthProxyWorkload_ValidateCreate_AuthProxyContainerSpec(t *testing.T) {
	wantPort := int32(9393)

	data := []struct {
		desc      string
		spec      cloudsqlapi.AuthProxyContainerSpec
		wantValid bool
	}{

		{
			desc: "Valid, Debug and AdminPort set",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				AdminServer: &cloudsqlapi.AdminServerSpec{
					EnableAPIs: []string{"Debug"},
					Port:       wantPort,
				},
			},
			wantValid: true,
		},
		{
			desc: "Valid, ImpersonationChain set",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				Authentication: &cloudsqlapi.AuthenticationSpec{
					ImpersonationChain: []string{"sv1@developer.gserviceaccount.com", "sv2@developer.gserviceaccount.com"},
				},
			},
			wantValid: true,
		},
		{
			desc: "Invalid, Debug set without AdminPort",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				AdminServer: &cloudsqlapi.AdminServerSpec{
					EnableAPIs: []string{"Debug"},
				},
			},
			wantValid: false,
		},
		{
			desc: "Invalid, EnableAPIs is empty",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				AdminServer: &cloudsqlapi.AdminServerSpec{
					Port: wantPort,
				},
			},
			wantValid: false,
		},
		{
			desc: "Invalid, EnableAPIs is has a bad value",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				AdminServer: &cloudsqlapi.AdminServerSpec{
					EnableAPIs: []string{"Nope", "Debug"},
					Port:       wantPort,
				},
			},
			wantValid: false,
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec: cloudsqlapi.AuthProxyWorkloadSpec{
					Workload: cloudsqlapi.WorkloadSelectorSpec{
						Kind: "Deployment",
						Name: "webapp",
					},
					AuthProxyContainer: &tc.spec,
					Instances: []cloudsqlapi.InstanceSpec{{
						ConnectionString: "proj:region:db2",
						Port:             ptr(int32(2443)),
					}},
				},
			}
			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &p)
			_, err := (&cloudsqlapi.AuthProxyWorkloadValidator{}).ValidateCreate(ctx, &p)
			gotValid := err == nil
			switch {
			case tc.wantValid && !gotValid:
				t.Errorf("wants create valid, got error %v", err)
				printFieldErrors(t, err)
			case !tc.wantValid && gotValid:
				t.Errorf("wants an error on create, got no error")
			default:
				t.Logf("create passed %s", tc.desc)
				// test passes, do nothing.
			}
		})
	}
}

func TestAuthProxyWorkload_ValidateUpdate(t *testing.T) {
	data := []struct {
		desc      string
		spec      cloudsqlapi.AuthProxyWorkloadSpec
		oldSpec   cloudsqlapi.AuthProxyWorkloadSpec
		wantValid bool
	}{
		{
			desc: "Valid, update adds another instance",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "webapp",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			oldSpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "webapp",
				},
				Instances: []cloudsqlapi.InstanceSpec{
					{
						ConnectionString: "proj:region:db1",
						PortEnvName:      "DB_PORT",
					},
					{
						ConnectionString: "proj:region:db2",
						PortEnvName:      "DB_PORT2",
					},
				},
			},
			wantValid: true,
		},
		{
			desc: "Invalid, WorkloadSelectorSpec.Kind changed",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "webapp",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db2",
					PortEnvName:      "DB_PORT",
				}},
			},
			oldSpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "StatefulSet",
					Name: "webapp",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			wantValid: false,
		},
		{
			desc: "Invalid, WorkloadSelectorSpec.Name changed",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "things",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db2",
					PortEnvName:      "DB_PORT",
				}},
			},
			oldSpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "webapp",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			wantValid: false,
		},
		{
			desc: "Invalid, WorkloadSelectorSpec.Selector changed",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Selector: &v1.LabelSelector{
						MatchLabels: map[string]string{"app": "sample"},
					},
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db2",
					PortEnvName:      "DB_PORT",
				}},
			},
			oldSpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Selector: &v1.LabelSelector{
						MatchLabels: map[string]string{"app": "other"},
					},
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.spec,
			}
			oldP := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.oldSpec,
			}
			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &p)
			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &oldP)
			_, err := (&cloudsqlapi.AuthProxyWorkloadValidator{}).ValidateUpdate(ctx, &oldP, &p)
			gotValid := err == nil

			switch {
			case tc.wantValid && !gotValid:
				t.Errorf("wants update valid, got error %v", err)
			case !tc.wantValid && gotValid:
				t.Errorf("wants an error on update, got no error")
			default:
				t.Logf("update passed %s", tc.desc)
				// test passes, do nothing.
			}
		})
	}
}

func TestAuthProxyWorkload_ValidateUpdate_AuthProxyContainerSpec(t *testing.T) {
	data := []struct {
		desc      string
		spec      *cloudsqlapi.AuthProxyContainerSpec
		oldSpec   *cloudsqlapi.AuthProxyContainerSpec
		wantValid bool
	}{
		{
			desc: "Invalid when AuthProxyContainerSpec.RolloutStrategy changes from explict to different default value",
			spec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "None",
			},
			oldSpec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "Workload",
			},
		},
		{
			desc: "Valid when AuthProxyContainerSpec.RolloutStrategy goes from default to same explicit value",
			spec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "Workload",
			},
			wantValid: true,
		},
		{
			desc: "Invalid when AuthProxyContainerSpec.RolloutStrategy changes from default to different explicit value",
			spec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "None",
			},
			wantValid: false,
		},
		{
			desc: "Invalid when AuthProxyContainerSpec.RolloutStrategy changes to different explicit value",
			spec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "None",
			},
			oldSpec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "Workload",
			},
			wantValid: false,
		},
		{
			desc: "Invalid when AuthProxyContainerSpec.RolloutStrategy changes from explict to different default value",
			spec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "None",
			},
			oldSpec: &cloudsqlapi.AuthProxyContainerSpec{
				RolloutStrategy: "Workload",
			},
			wantValid: false,
		},
	}
	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec: cloudsqlapi.AuthProxyWorkloadSpec{
					Workload: cloudsqlapi.WorkloadSelectorSpec{
						Kind: "Deployment",
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{"app": "sample"},
						},
					},
					AuthProxyContainer: tc.spec,
					Instances: []cloudsqlapi.InstanceSpec{{
						ConnectionString: "proj:region:db2",
						PortEnvName:      "DB_PORT",
					}},
				},
			}
			oldP := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec: cloudsqlapi.AuthProxyWorkloadSpec{
					Workload: cloudsqlapi.WorkloadSelectorSpec{
						Kind: "Deployment",
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{"app": "sample"},
						},
					},
					AuthProxyContainer: tc.oldSpec,
					Instances: []cloudsqlapi.InstanceSpec{{
						ConnectionString: "proj:region:db2",
						PortEnvName:      "DB_PORT",
					}},
				},
			}

			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &p)
			(&cloudsqlapi.AuthProxyWorkloadDefaulter{}).Default(ctx, &oldP)
			_, err := (&cloudsqlapi.AuthProxyWorkloadValidator{}).ValidateUpdate(ctx, &oldP, &p)
			gotValid := err == nil

			switch {
			case tc.wantValid && !gotValid:
				t.Errorf("wants update valid, got error %v", err)
			case !tc.wantValid && gotValid:
				t.Errorf("wants an error on update, got no error")
			default:
				t.Logf("update passed %s", tc.desc)
				// test passes, do nothing.
			}
		})
	}

}

func printFieldErrors(t *testing.T, err error) {
	t.Helper()
	statusErr, ok := err.(*apierrors.StatusError)
	if ok {
		t.Errorf("Field status errors: ")
		for _, v := range statusErr.Status().Details.Causes {
			t.Errorf("   %v %v: %v ", v.Field, v.Type, v.Message)
		}
	}
}

type fakeSARClient struct {
	client.Client
	allowedRules func(spec authorizationv1.SubjectAccessReviewSpec) bool
}

func (f *fakeSARClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	sar, ok := obj.(*authorizationv1.SubjectAccessReview)
	if !ok {
		return fmt.Errorf("unexpected object type %T", obj)
	}
	if f.allowedRules != nil {
		sar.Status.Allowed = f.allowedRules(sar.Spec)
	} else {
		sar.Status.Allowed = true
	}
	return nil
}

func TestAuthProxyWorkload_ValidateAuthorization(t *testing.T) {
	tests := []struct {
		desc         string
		spec         cloudsqlapi.AuthProxyWorkloadSpec
		allowedRules func(spec authorizationv1.SubjectAccessReviewSpec) bool
		wantAllowed  bool
	}{
		{
			desc: "Allowed: User has pod delete and deployment update/patch",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "my-deployment",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(_ authorizationv1.SubjectAccessReviewSpec) bool {
				return true
			},
			wantAllowed: true,
		},
		{
			desc: "Denied: User lacks pod delete permission",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "my-deployment",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(spec authorizationv1.SubjectAccessReviewSpec) bool {
				if spec.ResourceAttributes != nil && spec.ResourceAttributes.Resource == "pods" && spec.ResourceAttributes.Verb == "delete" {
					return false
				}
				return true
			},
			wantAllowed: false,
		},
		{
			desc: "Denied: User lacks deployment update permission",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "my-deployment",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(spec authorizationv1.SubjectAccessReviewSpec) bool {
				if spec.ResourceAttributes != nil && spec.ResourceAttributes.Resource == "deployments" {
					return false
				}
				return true
			},
			wantAllowed: false,
		},
		{
			desc: "Allowed: User has wildcard permission for label selector",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "StatefulSet",
					Selector: &v1.LabelSelector{
						MatchLabels: map[string]string{"app": "db"},
					},
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(_ authorizationv1.SubjectAccessReviewSpec) bool {
				return true
			},
			wantAllowed: true,
		},
		{
			desc: "Denied: User lacks wildcard permission for label selector",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "StatefulSet",
					Selector: &v1.LabelSelector{
						MatchLabels: map[string]string{"app": "db"},
					},
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(spec authorizationv1.SubjectAccessReviewSpec) bool {
				if spec.ResourceAttributes != nil && spec.ResourceAttributes.Resource == "statefulsets" {
					return false
				}
				return true
			},
			wantAllowed: false,
		},
		{
			desc: "Allowed: User has container override permission when specifying custom container",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "my-deployment",
				},
				AuthProxyContainer: &cloudsqlapi.AuthProxyContainerSpec{
					Container: &corev1.Container{
						Name:  "custom-proxy",
						Image: "custom-image:v1",
					},
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(_ authorizationv1.SubjectAccessReviewSpec) bool {
				return true
			},
			wantAllowed: true,
		},
		{
			desc: "Denied: User lacks container override permission when specifying custom container",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{
					Kind: "Deployment",
					Name: "my-deployment",
				},
				AuthProxyContainer: &cloudsqlapi.AuthProxyContainerSpec{
					Container: &corev1.Container{
						Name:  "custom-proxy",
						Image: "custom-image:v1",
					},
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			allowedRules: func(spec authorizationv1.SubjectAccessReviewSpec) bool {
				if spec.ResourceAttributes != nil && spec.ResourceAttributes.Subresource == "containeroverride" {
					return false
				}
				return true
			},
			wantAllowed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			p := &cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-authproxy",
					Namespace: "test-ns",
				},
				Spec: tc.spec,
			}

			req := admissionRequest("alice")
			ctx := admission.NewContextWithRequest(context.Background(), req)

			validator := &cloudsqlapi.AuthProxyWorkloadValidator{
				Client: &fakeSARClient{
					allowedRules: tc.allowedRules,
				},
			}

			_, err := validator.ValidateCreate(ctx, p)
			if tc.wantAllowed && err != nil {
				t.Fatalf("expected allowed, got error: %v", err)
			}
			if !tc.wantAllowed && err == nil {
				t.Fatalf("expected forbidden error, got nil")
			}

			// Also verify ValidateUpdate with modified spec
			oldObj := p.DeepCopy()
			oldObj.Spec.Instances = []cloudsqlapi.InstanceSpec{{
				ConnectionString: "proj:region:db2",
				PortEnvName:      "DB_PORT",
			}}
			_, err = validator.ValidateUpdate(ctx, oldObj, p)
			if tc.wantAllowed && err != nil {
				t.Fatalf("expected update allowed, got error: %v", err)
			}
			if !tc.wantAllowed && err == nil {
				t.Fatalf("expected update forbidden error, got nil")
			}

			// Verify ValidateUpdate with unchanged spec (e.g. controller adding finalizers) is always allowed
			_, err = validator.ValidateUpdate(ctx, p, p)
			if err != nil {
				t.Fatalf("expected update with unchanged spec to be allowed, got: %v", err)
			}
		})
	}
}

func admissionRequest(user string) admission.Request {
	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			UserInfo: authenticationv1.UserInfo{
				Username: user,
				UID:      "uid-" + user,
				Groups:   []string{"system:authenticated"},
			},
		},
	}
}
