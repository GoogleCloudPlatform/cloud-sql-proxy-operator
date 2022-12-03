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

package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMain(m *testing.M) {
	teardown, err := setupTests()
	defer teardown()

	if err != nil {
		fmt.Println("errors while initializing e2e test", err)
		os.Exit(1)
	}

	code := m.Run()

	os.Exit(code)
}

func TestCreateAndDeleteResource(t *testing.T) {
	tcc := newTestCaseClient("create")
	res, err := tcc.CreateResource(tcc.Ctx)
	if err != nil {
		t.Error(err)
	}
	err = tcc.WaitForFinalizerOnResource(tcc.Ctx, res)
	if err != nil {
		t.Error(err)
	}
	err = tcc.DeleteResourceAndWait(tcc.Ctx, res)
	if err != nil {
		t.Error(err)
	}

}

func TestProxyAppliedOnNewWorkload(t *testing.T) {
	tests := []struct {
		name string
		o    client.Object
	}{
		{
			name: "deployment",
			o:    testhelpers.BuildDeployment(types.NamespacedName{}, "busybox"),
		},
		{
			name: "statefulset",
			o:    testhelpers.BuildStatefulSet(types.NamespacedName{}, "busybox"),
		},
		{
			name: "daemonset",
			o:    testhelpers.BuildDaemonSet(types.NamespacedName{}, "busybox"),
		},
		{
			name: "job",
			o:    testhelpers.BuildJob(types.NamespacedName{}, "busybox"),
		},
		{
			name: "cronjob",
			o:    testhelpers.BuildCronJob(types.NamespacedName{}, "busybox"),
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			kind := test.o.GetObjectKind().GroupVersionKind().Kind
			tp := newTestCaseClient("new" + strings.ToLower(kind))

			err := tp.CreateOrPatchNamespace()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				err = tp.DeleteNamespace()
				if err != nil {
					t.Fatal(err)
				}
			})

			const (
				pwlName  = "newss"
				appLabel = "busybox"
			)
			key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

			t.Log("Creating AuthProxyWorkload")
			err = tp.CreateAuthProxyWorkload(key, appLabel, tp.ConnectionString, kind)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(key)
			if err != nil {
				t.Fatal("unable to create AuthProxyWorkload", err)
			}

			t.Log("Creating ", kind)
			test.o.SetNamespace(tp.Namespace)
			test.o.SetName(test.name)
			err = tp.CreateWorkload(test.o)
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			}
			err = tp.ExpectPodContainerCount(selector, 2, "all")
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestProxyAppliedOnExistingWorkload(t *testing.T) {
	tests := []struct {
		name     string
		o        client.Object
		allOrAny string
	}{
		{
			name:     "deployment",
			o:        testhelpers.BuildDeployment(types.NamespacedName{}, "busybox"),
			allOrAny: "all",
		},
		{
			name:     "statefulset",
			o:        testhelpers.BuildStatefulSet(types.NamespacedName{}, "busybox"),
			allOrAny: "all",
		},
		{
			name:     "daemonset",
			o:        testhelpers.BuildDaemonSet(types.NamespacedName{}, "busybox"),
			allOrAny: "all",
		},
		{
			name:     "job",
			o:        testhelpers.BuildJob(types.NamespacedName{}, "busybox"),
			allOrAny: "any",
		},
		{
			name:     "cronjob",
			o:        testhelpers.BuildCronJob(types.NamespacedName{}, "busybox"),
			allOrAny: "any",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			kind := test.o.GetObjectKind().GroupVersionKind().Kind

			tp := newTestCaseClient("modify" + strings.ToLower(kind))

			err := tp.CreateOrPatchNamespace()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				err = tp.DeleteNamespace()
				if err != nil {
					t.Fatal(err)
				}
			})

			const (
				pwlName  = "newss"
				appLabel = "busybox"
			)
			key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

			t.Log("Creating ", kind)
			test.o.SetNamespace(tp.Namespace)
			test.o.SetName(test.name)
			err = tp.CreateWorkload(test.o)
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			}

			err = tp.ExpectPodContainerCount(selector, 1, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Creating AuthProxyWorkload")
			err = tp.CreateAuthProxyWorkload(key, appLabel, tp.ConnectionString, kind)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(key)
			if err != nil {
				t.Fatal(err)
			}

			err = tp.ExpectPodContainerCount(selector, 2, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
