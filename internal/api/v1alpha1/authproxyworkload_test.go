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

package v1alpha1_test

import (
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuthProxyWorkload_ValidateCreate(t *testing.T) {
	data := []struct {
		desc            string
		spec            cloudsqlapi.AuthProxyWorkloadSpec
		wantCreateValid bool
		wantUpdateValid bool
	}{
		{
			desc: "happy path",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
			},
			wantCreateValid: true,
			wantUpdateValid: true,
		},
		{
			desc: "happy path labels",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": "sample"}}},
			},
			wantCreateValid: true,
			wantUpdateValid: true,
		},
		{
			desc: "invalid labels and name both set",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp", Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": "sample"}}},
			},
			wantCreateValid: false,
			wantUpdateValid: true,
		},
		{
			desc: "invalid labels and name not set",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment"},
			},
			wantCreateValid: false,
			wantUpdateValid: true,
		},
		{
			desc: "happy path update",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db2",
					PortEnvName:      "DB_PORT",
				}},
			},
			wantCreateValid: true,
			wantUpdateValid: true,
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.spec,
			}
			err := p.ValidateCreate()
			switch {
			case tc.wantCreateValid && err != nil:
				t.Errorf("wants create valid, got error %v", err)
				printFieldErrors(t, err)
			case !tc.wantCreateValid && err == nil:
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
		desc            string
		spec            cloudsqlapi.AuthProxyWorkloadSpec
		oldSpec         cloudsqlapi.AuthProxyWorkloadSpec
		wantCreateValid bool
		wantUpdateValid bool
	}{
		{
			desc: "happy path",
			spec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			oldSpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
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
			wantCreateValid: true,
			wantUpdateValid: true,
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.spec,
			}
			oldP := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.oldSpec}

			err := p.ValidateUpdate(&oldP)
			switch {
			case tc.wantUpdateValid && err != nil:
				t.Errorf("wants create valid, got error %v", err)
			case !tc.wantUpdateValid && err == nil:
				t.Errorf("wants an error on create, got no error")
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
