// Copyright 2022 Google LLC
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

package names_test

import (
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/names"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSafeDnsLabel(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		desc string
		want string
		name string
	}{
		{
			desc: "short name",
			name: "instance1",
			want: "csql-instance1",
		},
		{
			desc: "max length name truncates to safe length",
			name: "twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-the-wa",
			want: "csql-twas-brillig-and-the-sli-yre-and-gimble-in-the-wa-e398b76e",
		},
		{
			desc: "just barely too long name truncates to safe length",
			name: "twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-th",
			want: "csql-twas-brillig-and-the-sli-id-gyre-and-gimble-in-th-78bfbd48",
		},
		{
			desc: "acceptable length long name is left whole",
			name: "twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-t",
			want: "csql-twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-t",
		},
		{
			desc: "truncated difference in middle preserved in hash 1",
			name: "twas-brillig-and-the-slithy-toves-1111-did-gyre-and-gimble-in",
			want: "csql-twas-brillig-and-the-sli-1-did-gyre-and-gimble-in-d0b9860",
		},
		{
			desc: "truncated difference in middle preserved in hash 2",
			name: "twas-brillig-and-the-slithy-toves-2222-did-gyre-and-gimble-in",
			want: "csql-twas-brillig-and-the-sli-2-did-gyre-and-gimble-in-34c209d4",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := names.SafePrefixedName("csql-", tc.name)
			if got != tc.want {
				t.Errorf("want %v. got %v", tc.want, got)
			}
			if len(got) > 63 {
				t.Errorf("want len(containerName) <= 63. got %v", len(got))
			}
		})
	}

}

// TestContainerName container names are a public
func TestContainerName(t *testing.T) {
	csql := mustMakeCsql("hello-world", "default")
	got := names.ContainerName(csql)
	want := "csql-hello-world"
	if want != got {
		t.Errorf("got %v, want %v", got, want)
	}

}

func mustMakeCsql(name string, namespace string) *cloudsqlapi.AuthProxyWorkload {
	// Create a CloudSqlInstance that matches the deployment
	return &cloudsqlapi.AuthProxyWorkload{
		TypeMeta:   metav1.TypeMeta{Kind: "AuthProxyWorkload", APIVersion: cloudsqlapi.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: cloudsqlapi.AuthProxyWorkloadSpec{
			Workload: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "",
				Name: "",
				Selector: &metav1.LabelSelector{
					MatchLabels:      map[string]string{"app": "hello"},
					MatchExpressions: nil,
				},
			},
			Instances: []cloudsqlapi.InstanceSpec{{ConnectionString: "proj:inst:db"}},
		},
		Status: cloudsqlapi.AuthProxyWorkloadStatus{},
	}

}
