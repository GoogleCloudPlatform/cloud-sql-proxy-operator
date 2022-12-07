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
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	ctx := testContext()
	tcc := newTestCaseClient("create")
	res, err := tcc.CreateResource(ctx)
	if err != nil {
		t.Error(err)
	}
	err = tcc.WaitForFinalizerOnResource(ctx, res)
	if err != nil {
		t.Error(err)
	}
	err = tcc.DeleteResourceAndWait(ctx, res)
	if err != nil {
		t.Error(err)
	}

}

func TestProxyAppliedOnNewWorkload(t *testing.T) {
	// When running tests during development, set the SKIP_CLEANUP=true envvar so that
	// the test namespace remains after the test ends. By default, the test
	// namespace will be deleted when the test exits.
	skipCleanup := loadValue("SKIP_CLEANUP", "", "false") == "true"

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
			ctx := testContext()

			kind := test.o.GetObjectKind().GroupVersionKind().Kind
			tp := newTestCaseClient("new" + strings.ToLower(kind))

			err := tp.CreateOrPatchNamespace(ctx)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if skipCleanup {
					return
				}

				err = tp.DeleteNamespace(ctx)
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
			err = tp.CreateAuthProxyWorkload(ctx, key, appLabel, tp.ConnectionString, kind)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
			if err != nil {
				t.Fatal("unable to create AuthProxyWorkload", err)
			}

			t.Log("Creating ", kind)
			test.o.SetNamespace(tp.Namespace)
			test.o.SetName(test.name)
			err = tp.CreateWorkload(ctx, test.o)
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			}
			t.Log("Checking for container counts", kind)
			err = tp.ExpectPodContainerCount(ctx, selector, 2, "all")
			if err != nil {
				t.Error(err)
			}
			t.Log("Done, OK", kind)
		})
	}
}

func TestProxyAppliedOnExistingWorkload(t *testing.T) {
	// When running tests during development, set the SKIP_CLEANUP=true envvar so that
	// the test namespace remains after the test ends. By default, the test
	// namespace will be deleted when the test exits.
	skipCleanup := loadValue("SKIP_CLEANUP", "", "false") == "true"

	tests := []struct {
		name     string
		o        workload.Workload
		allOrAny string
	}{
		{
			name:     "deployment",
			o:        &workload.DeploymentWorkload{Deployment: testhelpers.BuildDeployment(types.NamespacedName{}, "busybox")},
			allOrAny: "all",
		},
		{
			name:     "statefulset",
			o:        &workload.StatefulSetWorkload{StatefulSet: testhelpers.BuildStatefulSet(types.NamespacedName{}, "busybox")},
			allOrAny: "all",
		},
		{
			name:     "daemonset",
			o:        &workload.DaemonSetWorkload{DaemonSet: testhelpers.BuildDaemonSet(types.NamespacedName{}, "busybox")},
			allOrAny: "all",
		},
		{
			name:     "job",
			o:        &workload.JobWorkload{Job: testhelpers.BuildJob(types.NamespacedName{}, "busybox")},
			allOrAny: "any",
		},
		{
			name:     "cronjob",
			o:        &workload.CronJobWorkload{CronJob: testhelpers.BuildCronJob(types.NamespacedName{}, "busybox")},
			allOrAny: "any",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := testContext()
			kind := test.o.Object().GetObjectKind().GroupVersionKind().Kind

			tp := newTestCaseClient("modify" + strings.ToLower(kind))

			err := tp.CreateOrPatchNamespace(ctx)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if skipCleanup {
					return
				}
				err = tp.DeleteNamespace(ctx)
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
			test.o.Object().SetNamespace(tp.Namespace)
			test.o.Object().SetName(test.name)
			err = tp.CreateWorkload(ctx, test.o.Object())
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			}

			err = tp.ExpectPodContainerCount(ctx, selector, 1, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Creating AuthProxyWorkload")
			err = tp.CreateAuthProxyWorkload(ctx, key, appLabel, tp.ConnectionString, kind)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Pod container count remains unmodified for existing workload")
			err = tp.ExpectPodContainerCount(ctx, selector, 1, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}

			// if this is an apps/v1 resource with a mutable pod template,
			// force a rolling update.

			if wl, ok := test.o.(workload.WithMutablePodTemplate); ok {
				// patch the workload, add an annotation to the podspec
				t.Log("Customer updates the workload triggering a rollout")
				controllerutil.CreateOrPatch(ctx, tp.Client, test.o.Object(), func() error {
					wl.SetPodTemplateAnnotations(map[string]string{"customer": "updated"})
					return nil
				})

				if err != nil {
					t.Fatal(err)
				}
			}

			t.Logf("Wait for %v pods to have 2 containers", test.allOrAny)
			err = tp.ExpectPodContainerCount(ctx, selector, 2, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}

		})
	}
}
