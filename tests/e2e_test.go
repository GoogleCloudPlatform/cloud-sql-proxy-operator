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
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	tcc := newPublicPostgresClient("create")
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
			tp := newPublicPostgresClient("new" + strings.ToLower(kind))

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
			_, err = tp.CreateAuthProxyWorkload(ctx, key, appLabel, tp.ConnectionString, kind)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
			if err != nil {
				t.Fatal("unable to create AuthProxyWorkload", err)
			}

			t.Log("Creating ", kind)
			err = createWorkload(ctx, tp, test.o, test.name)
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := appSelector(appLabel)
			t.Log("Checking for container counts", kind)
			err = tp.ExpectPodContainerCount(ctx, selector, 2, test.allOrAny)
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

			tp := newPublicPostgresClient("modify" + strings.ToLower(kind))

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
			err = createWorkload(ctx, tp, test.o, test.name)
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := appSelector(appLabel)

			err = tp.ExpectPodContainerCount(ctx, selector, 1, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Creating AuthProxyWorkload")
			_, err = tp.CreateAuthProxyWorkload(ctx, key, appLabel, tp.ConnectionString, kind)
			if err != nil {
				t.Fatal(err)
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Wait for %v pods to have 2 containers", test.allOrAny)
			err = tp.ExpectPodContainerCount(ctx, selector, 2, test.allOrAny)
			if err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestPublicDBConnections(t *testing.T) {
	// When running tests during development, set the SKIP_CLEANUP=true envvar so that
	// the test namespace remains after the test ends. By default, the test
	// namespace will be deleted when the test exits.
	skipCleanup := loadValue("SKIP_CLEANUP", "", "false") == "true"
	const (
		pwlName  = "newss"
		appLabel = "client"
		kind     = "Deployment"
	)

	tests := []struct {
		name         string
		c            *testhelpers.TestCaseClient
		podTemplate  corev1.PodTemplateSpec
		allOrAny     string
		isUnixSocket bool
	}{
		{
			name:        "postgres",
			c:           newPublicPostgresClient("postgresconn"),
			podTemplate: testhelpers.BuildPgPodSpec(600, appLabel, "db-secret"),
			allOrAny:    "all",
		},
		{
			name:         "postgres-unix",
			c:            newPublicPostgresClient("pgconnunix"),
			podTemplate:  testhelpers.BuildPgUnixPodSpec(600, appLabel, "db-secret"),
			allOrAny:     "all",
			isUnixSocket: true,
		},
		{
			name:        "mysql",
			c:           newPublicMySQLClient("mysqlconn"),
			podTemplate: testhelpers.BuildMySQLPodSpec(600, appLabel, "db-secret"),
			allOrAny:    "all",
		},
		{
			name:         "mysql-unix",
			c:            newPublicMySQLClient("mysqlconnunix"),
			podTemplate:  testhelpers.BuildMySQLUnixPodSpec(600, appLabel, "db-secret"),
			allOrAny:     "all",
			isUnixSocket: true,
		},
		{
			name:        "mssql",
			c:           newPublicMSSQLClient("mssqlconn"),
			podTemplate: testhelpers.BuildMSSQLPodSpec(600, appLabel, "db-secret"),
			allOrAny:    "all",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := testContext()
			tp := test.c

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

			key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

			s := testhelpers.BuildSecret("db-secret", tp.DBRootUsername, tp.DBRootPassword, tp.DBName)
			s.SetNamespace(tp.Namespace)
			err = tp.Client.Create(ctx, &s)
			if err != nil {
				t.Fatal(err)
			}

			wl := &workload.DeploymentWorkload{Deployment: testhelpers.BuildDeployment(types.NamespacedName{}, appLabel)}
			wl.Deployment.Spec.Template = test.podTemplate
			t.Log("Creating AuthProxyWorkload")

			if test.isUnixSocket {
				p := testhelpers.NewAuthProxyWorkload(key)
				testhelpers.AddUnixInstance(p, tp.ConnectionString, "/var/tests/dbsocket")
				tp.ConfigureSelector(p, appLabel, kind)
				tp.ConfigureResources(p)
				err = tp.Create(ctx, p)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				_, err = tp.CreateAuthProxyWorkload(ctx, key, appLabel, tp.ConnectionString, kind)
				if err != nil {
					t.Fatal(err)
				}
			}

			t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
			_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
			if err != nil {
				t.Fatal("unable to create AuthProxyWorkload", err)
			}

			t.Log("Creating ", kind)
			err = createWorkload(ctx, tp, wl, pwlName)
			if err != nil {
				t.Fatal("unable to create ", kind, err)
			}
			selector := appSelector(appLabel)
			t.Log("Checking for container counts", kind)
			err = tp.ExpectPodContainerCount(ctx, selector, 2, "all")
			if err != nil {
				t.Error(err)
			}

			// The pods are configured to only be ready when the real database client
			// successfully executes a simple query on the database.
			t.Log("Checking for ready", kind)
			err = tp.ExpectPodReady(ctx, selector, "all")
			if err != nil {
				t.Error(err)
			}

			t.Log("Done, OK", kind)

		})
	}

}

func TestUpdateWorkloadOnDelete(t *testing.T) {

	const (
		pwlName  = "newss"
		appLabel = "busybox"
		name     = "app"
		allOrAny = "all"
	)
	// Use a deployment workload
	wl := &workload.DeploymentWorkload{Deployment: testhelpers.BuildDeployment(types.NamespacedName{}, "busybox")}
	o := wl.Object()
	kind := o.GetObjectKind().GroupVersionKind().Kind

	// Set up the e2e test namespace
	skipCleanup := loadValue("SKIP_CLEANUP", "", "false") == "true"
	ctx := testContext()
	tp := newPublicPostgresClient("new" + strings.ToLower(kind))

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

	// Create AuthProxyWorkload
	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	t.Log("Creating AuthProxyWorkload")
	proxy, err := tp.CreateAuthProxyWorkload(ctx, key, appLabel, tp.ConnectionString, kind)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
	if err != nil {
		t.Fatal("unable to create AuthProxyWorkload", err)
	}

	// Create deployment
	t.Log("Creating ", kind)
	err = createWorkload(ctx, tp, wl, name)
	if err != nil {
		t.Fatal("unable to create ", kind, err)
	}
	selector := appSelector(appLabel)

	// Check that the deployment pods are configured with the proxy: pods
	// have 2 containers.
	t.Log("Checking for container counts", kind)
	err = tp.ExpectPodContainerCount(ctx, selector, 2, allOrAny)
	if err != nil {
		t.Error(err)
	}
	t.Log("Workload Created. Removing AuthProxyWorkload", kind)

	// Delete the AuthProxyWorkload
	err = tp.Client.Delete(ctx, proxy)
	if err != nil {
		t.Fatal(err)
	}

	// Check that deployment pods are configured without the proxy: pods have
	// 1 container.
	t.Log("Checking for container counts after delete", kind)
	err = tp.ExpectPodContainerCount(ctx, selector, 1, allOrAny)
	if err != nil {
		t.Error(err)
	}
}

func TestPrivateDBConnections(t *testing.T) {
	// When running tests during development, set the SKIP_CLEANUP=true envvar so that
	// the test namespace remains after the test ends. By default, the test
	// namespace will be deleted when the test exits.
	skipCleanup := loadValue("SKIP_CLEANUP", "", "false") == "true"
	const (
		pwlName  = "newss"
		appLabel = "client"
		kind     = "Deployment"
		allOrAny = "all"
	)

	ctx := testContext()
	tp := newPrivatePostgresClient("postgresconn")

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

	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	s := testhelpers.BuildSecret("db-secret", tp.DBRootUsername, tp.DBRootPassword, tp.DBName)
	s.SetNamespace(tp.Namespace)
	err = tp.Client.Create(ctx, &s)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Creating AuthProxyWorkload")
	b := true
	p := testhelpers.NewAuthProxyWorkload(key)
	testhelpers.AddUnixInstance(p, tp.ConnectionString, "/var/tests/dbsocket")
	tp.ConfigureSelector(p, appLabel, kind)
	tp.ConfigureResources(p)
	p.Spec.Instances[0].PrivateIP = &b

	err = tp.Create(ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = tp.GetAuthProxyWorkloadAfterReconcile(ctx, key)
	if err != nil {
		t.Fatal("unable to create AuthProxyWorkload", err)
	}

	t.Log("Creating ", kind)
	wl := &workload.DeploymentWorkload{Deployment: testhelpers.BuildDeployment(types.NamespacedName{}, appLabel)}
	wl.Deployment.Spec.Template = testhelpers.BuildPgUnixPodSpec(
		600, appLabel, "db-secret")
	err = createWorkload(ctx, tp, wl, pwlName)
	if err != nil {
		t.Fatal("unable to create ", kind, err)
	}

	selector := appSelector(appLabel)
	t.Log("Checking for container counts", kind)
	err = tp.ExpectPodContainerCount(ctx, selector, 2, "all")
	if err != nil {
		t.Error(err)
	}

	// The pods are configured to only be ready when the real database client
	// successfully executes a simple query on the database.
	t.Log("Checking for ready", kind)
	err = tp.ExpectPodReady(ctx, selector, "all")
	if err != nil {
		t.Error(err)
	}

	t.Log("Done, OK", kind)

}

// createWorkload will set name and namespace appropriately, then use the client
// to create the workload.
func createWorkload(ctx context.Context, tp *testhelpers.TestCaseClient, wl workload.Workload, name string) error {
	wl.Object().SetNamespace(tp.Namespace)
	wl.Object().SetName(name)
	return tp.CreateWorkload(ctx, wl.Object())
}

// appSelector creates a label selector for "app={appLabel}".
func appSelector(appLabel string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": appLabel},
	}
}
