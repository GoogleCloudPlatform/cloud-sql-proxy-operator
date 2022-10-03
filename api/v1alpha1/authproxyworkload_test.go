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
package v1alpha1_test

import (
	"os"
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

func TestUnmarshalAuthProxyWorkloadSample(t *testing.T) {
	wl := &cloudsqlapi.AuthProxyWorkload{}
	data, err := os.ReadFile("../../config/samples/cloudsql_v1alpha1_authproxyworkload_full.yaml")
	if err != nil {
		t.Error(err)
	}
	err = yaml.Unmarshal(data, wl)
	if err != nil {
		t.Errorf("Can't unmarshal, %v", err)
	}
}

func TestAuthProxyWorkload_ValidateCreate(t *testing.T) {
	type testcase struct {
		desc            string
		spec            *cloudsqlapi.AuthProxyWorkloadSpec
		oldSpec         *cloudsqlapi.AuthProxyWorkloadSpec
		wantCreateValid bool
		wantUpdateValid bool
	}

	data := []*testcase{
		{
			desc: "happy path",
			spec: &cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
			},
			wantCreateValid: true,
			wantUpdateValid: true,
		},
		{
			desc: "happy path update",
			spec: &cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db2",
					PortEnvName:      "DB_PORT",
				}},
			},
			oldSpec: &cloudsqlapi.AuthProxyWorkloadSpec{
				Workload: cloudsqlapi.WorkloadSelectorSpec{Kind: "Deployment", Name: "webapp"},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "proj:region:db1",
					PortEnvName:      "DB_PORT",
				}},
			},
			wantCreateValid: true,
			wantUpdateValid: true,
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
			p := cloudsqlapi.AuthProxyWorkload{Spec: tc.spec}
			err := p.ValidateCreate()
			switch {
			case tc.wantCreateValid && err != nil:
				t.Errorf("wants create valid, got error %v", err)
			case !tc.wantCreateValid && err == nil:
				t.Errorf("wants an error on create, got no error")
			default:
				t.Logf("create passed %s", tc.desc)
				// test passes, do nothing.
			}

			if tc.oldSpec == nil {
				return
			}

			oldP := cloudsqlapi.AuthProxyWorkload{Spec: tc.oldSpec}

			err = p.ValidateUpdate(&oldP)
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
