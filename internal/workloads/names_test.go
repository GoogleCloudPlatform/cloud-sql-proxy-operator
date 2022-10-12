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

package workloads_test

import (
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workloads"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
)

func TestSafePrefixedName(t *testing.T) {
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
			want: "csql-twas-brillig-and-the-sliyre-and-gimble-in-the-wa-e398b76e",
		},
		{
			desc: "just barely too long name truncates to safe length",
			name: "twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-th",
			want: "csql-twas-brillig-and-the-sliid-gyre-and-gimble-in-th-78bfbd48",
		},
		{
			desc: "acceptable length long name is left whole",
			name: "twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-t",
			want: "csql-twas-brillig-and-the-slithy-toves-did-gyre-and-gimble-in-t",
		},
		{
			desc: "truncated difference in middle preserved in mustHash 1",
			name: "twas-brillig-and-the-slithy-toves-1111-did-gyre-and-gimble-in",
			want: "csql-twas-brillig-and-the-slit11-did-gyre-and-gimble-in-d0b9860",
		},
		{
			desc: "truncated difference in middle preserved in mustHash 2",
			name: "twas-brillig-and-the-slithy-toves-2222-did-gyre-and-gimble-in",
			want: "csql-twas-brillig-and-the-sli2-did-gyre-and-gimble-in-34c209d4",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := workloads.SafePrefixedName("csql-", tc.name)
			if got != tc.want {
				t.Errorf("want %v. got %v", tc.want, got)
			}
			if len(got) > 63 {
				t.Errorf("want len(containerName) <= 63. got %v", len(got))
			}
		})
	}
}

func TestContainerName(t *testing.T) {
	csql := authProxyWorkload("hello-world", []v1alpha1.InstanceSpec{{ConnectionString: "proj:inst:db"}})
	got := workloads.ContainerName(csql)
	want := "csql-default-hello-world"
	if want != got {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestVolumeName(t *testing.T) {
	csql := authProxyWorkload("hello-world", []v1alpha1.InstanceSpec{{ConnectionString: "proj:inst:db"}})
	got := workloads.VolumeName(csql, &csql.Spec.Instances[0], "temp")
	want := "csql-default-hello-world-temp-proj-inst-db"
	if want != got {
		t.Errorf("got %v, want %v", got, want)
	}
}
