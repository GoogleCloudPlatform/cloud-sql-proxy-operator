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

func ptr[T int | int32 | int64 | string](i T) *T {
	return &i
}

func TestAuthProxyWorkload_ValidateCreate_Instances(t *testing.T) {

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
	}
	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
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
			p.Default()
			err := p.ValidateCreate()
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
			p.Default()
			err := p.ValidateCreate()
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
	wantTrue := true
	wantPort := int32(9393)

	data := []struct {
		desc      string
		spec      cloudsqlapi.AuthProxyContainerSpec
		wantValid bool
	}{

		{
			desc: "Valid, Debug and AdminPort set",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				Telemetry: &cloudsqlapi.TelemetrySpec{
					Debug:     &wantTrue,
					AdminPort: &wantPort,
				},
			},
			wantValid: true,
		},
		{
			desc: "Invalid, Debug set without AdminPort",
			spec: cloudsqlapi.AuthProxyContainerSpec{
				Telemetry: &cloudsqlapi.TelemetrySpec{
					Debug: &wantTrue,
				},
			},
			wantValid: false,
		},
	}

	for _, tc := range data {
		t.Run(tc.desc, func(t *testing.T) {
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
			p.Default()
			err := p.ValidateCreate()
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
			p := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.spec,
			}
			oldP := cloudsqlapi.AuthProxyWorkload{
				ObjectMeta: v1.ObjectMeta{Name: "sample"},
				Spec:       tc.oldSpec,
			}
			p.Default()
			oldP.Default()

			err := p.ValidateUpdate(&oldP)
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
			p.Default()
			oldP.Default()

			err := p.ValidateUpdate(&oldP)
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
