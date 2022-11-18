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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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

func TestCreateResource(t *testing.T) {
	tctx := params(t, "create")
	testhelpers.TestCreateResource(tctx)
}

func TestDeleteResource(t *testing.T) {
	tctx := params(t, "delete")
	testhelpers.TestDeleteResource(tctx)
}

func TestProxyAppliedOnNewWorkload(t *testing.T) {
	tests := []struct {
		name string
		o    client.Object
		kind string
		yaml string
	}{
		{
			name: "deployment",
			o:    &appsv1.Deployment{},
			kind: "Deployment",
			yaml: testhelpers.BusyboxDeployYaml,
		},
		{
			name: "statefulset",
			o:    &appsv1.StatefulSet{},
			kind: "StatefulSet",
			yaml: testhelpers.BusyboxStatefulSetYaml,
		},
		{
			name: "daemonset",
			o:    &appsv1.DaemonSet{},
			kind: "DaemonSet",
			yaml: testhelpers.BusyboxDaemonSetYaml,
		},
		{
			name: "job",
			o:    &batchv1.Job{},
			kind: "Job",
			yaml: testhelpers.BusyboxJob,
		},
		{
			name: "cronjob",
			o:    &batchv1.CronJob{},
			kind: "CronJob",
			yaml: testhelpers.BusyboxCronJob,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tp := params(t, "new"+strings.ToLower(test.kind))

			testhelpers.CreateOrPatchNamespace(tp.Ctx, tp)

			const (
				pwlName  = "newss"
				appLabel = "busybox"
			)
			ctx := tp.Ctx
			key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

			t.Log("Creating AuthProxyWorkload")
			err := testhelpers.CreateAuthProxyWorkload(ctx, tp, key, appLabel, tp.ConnectionString, test.kind)
			if err != nil {
				t.Error(err)
				testhelpers.DeleteNamespace(tp, false)
				return
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = testhelpers.GetAuthProxyWorkloadAfterReconcile(ctx, tp, key)
			if err != nil {
				t.Error("unable to create AuthProxyWorkload", err)
				testhelpers.DeleteNamespace(tp, false)
				return
			}

			t.Log("Creating ", test.kind)
			err = testhelpers.CreateWorkload(ctx, tp, key, appLabel, test.yaml, test.o)
			if err != nil {
				t.Error("unable to create ", test.kind, err)
				testhelpers.DeleteNamespace(tp, false)
				return
			}
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			}
			testhelpers.ExpectContainerCount(tp, test.o.GetNamespace(), selector, 2, "all")
			testhelpers.DeleteNamespace(tp, false)
		})
	}
}

func TestProxyAppliedOnExistingWorkload(t *testing.T) {
	tests := []struct {
		name     string
		o        client.Object
		kind     string
		yaml     string
		allOrAny string
	}{
		{
			name:     "deployment",
			o:        &appsv1.Deployment{},
			kind:     "Deployment",
			yaml:     testhelpers.BusyboxDeployYaml,
			allOrAny: "all",
		},
		{
			name:     "statefulset",
			o:        &appsv1.StatefulSet{},
			kind:     "StatefulSet",
			yaml:     testhelpers.BusyboxStatefulSetYaml,
			allOrAny: "all",
		},
		{
			name:     "daemonset",
			o:        &appsv1.DaemonSet{},
			kind:     "DaemonSet",
			yaml:     testhelpers.BusyboxDaemonSetYaml,
			allOrAny: "all",
		},
		{
			name:     "job",
			o:        &batchv1.Job{},
			kind:     "Job",
			yaml:     testhelpers.BusyboxJob,
			allOrAny: "any",
		},
		{
			name:     "cronjob",
			o:        &batchv1.CronJob{},
			kind:     "CronJob",
			yaml:     testhelpers.BusyboxCronJob,
			allOrAny: "any",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tp := params(t, "modify"+strings.ToLower(test.kind))

			testhelpers.CreateOrPatchNamespace(tp.Ctx, tp)

			const (
				pwlName  = "newss"
				appLabel = "busybox"
			)
			ctx := tp.Ctx
			key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

			t.Log("Creating ", test.kind)
			err := testhelpers.CreateWorkload(ctx, tp, key, appLabel, test.yaml, test.o)
			if err != nil {
				t.Error("unable to create ", test.kind, err)
				testhelpers.DeleteNamespace(tp, false)
				return
			}
			selector := &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			}
			err = testhelpers.ExpectContainerCount(tp, test.o.GetNamespace(), selector, 1, test.allOrAny)
			if err != nil {
				testhelpers.DeleteNamespace(tp, false)
				return
			}

			t.Log("Creating AuthProxyWorkload")
			err = testhelpers.CreateAuthProxyWorkload(ctx, tp, key, appLabel, tp.ConnectionString, test.kind)
			if err != nil {
				t.Error(err)
				testhelpers.DeleteNamespace(tp, false)
				return
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = testhelpers.GetAuthProxyWorkloadAfterReconcile(ctx, tp, key)
			if err != nil {
				t.Error("unable to create AuthProxyWorkload", err)
				return
			}

			testhelpers.ExpectContainerCount(tp, test.o.GetNamespace(), selector, 2, test.allOrAny)

			testhelpers.DeleteNamespace(tp, false)

		})
	}
}
