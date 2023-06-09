// Copyright 2022 Google LLC.
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

// Package integration_test has testintegration tests that run a local kubernetes
// api server and ensure that the interaction between kubernetes and the
// operator works correctly.
package testintegration_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testintegration"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// defaultClient holds the k8s client for the default EnvTestHarness which is
// shared among tests that do not mess with the manager's lifecycle.
var defaultClient client.Client

func TestMain(m *testing.M) {
	// start up a shared EnvTestHarness to be reused across test cases that do not
	// impact the lifecycle of the manager. This makes tests cases more efficient
	// because it takes 2-3 minutes to start up a new EnvTestHarness.
	var err error
	th, err := testintegration.NewTestHarness()

	if err != nil {
		testintegration.Log.Error(err, "errors while initializing kubernetes cluster")
		th.Teardown()
		os.Exit(1)
	}

	defaultClient = th.Client

	code := m.Run()
	th.Teardown()
	os.Exit(code)
}

// newTestCaseClient Creates a new TestCaseClient providing unique namespace and
// other default values.
func newTestCaseClient(name string, c client.Client) *testhelpers.TestCaseClient {
	return &testhelpers.TestCaseClient{
		Client:           c,
		Namespace:        testhelpers.NewNamespaceName(name),
		ConnectionString: "region:project:inst",
	}
}

func TestCreateAndDeleteResource(t *testing.T) {
	ctx := testintegration.TestContext()
	tcc := newTestCaseClient("create", defaultClient)
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

func TestModifiesNewDeployment(t *testing.T) {
	ctx := testintegration.TestContext()
	tcc := newTestCaseClient("modifynew", defaultClient)

	err := tcc.CreateOrPatchNamespace(ctx)
	if err != nil {
		t.Fatalf("can't create namespace, %v", err)
	}

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	key := types.NamespacedName{Name: pwlName, Namespace: tcc.Namespace}

	t.Log("Creating AuthProxyWorkload")
	_, err = tcc.CreateAuthProxyWorkload(ctx, key, deploymentAppLabel, tcc.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = tcc.GetAuthProxyWorkloadAfterReconcile(ctx, key)
	if err != nil {
		t.Error("unable to create AuthProxyWorkload", err)
		return
	}

	t.Log("Creating deployment")
	d := testhelpers.BuildDeployment(key, deploymentAppLabel)
	err = tcc.CreateWorkload(ctx, d)

	if err != nil {
		t.Error("unable to create deployment", err)
		return
	}

	t.Log("Creating deployment replicas")
	_, _, err = tcc.CreateDeploymentReplicaSetAndPods(ctx, d)
	if err != nil {
		t.Error("unable to create pods", err)
		return
	}

	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 2, "all")
	if err != nil {
		t.Error(err)
	}

}

func TestModifiesExistingDeployment(t *testing.T) {
	const (
		pwlName            = "db-mod"
		deploymentName     = "deploy-mod"
		deploymentAppLabel = "existing-mod"
	)
	ctx := testintegration.TestContext()
	tcc := newTestCaseClient("modifyexisting", defaultClient)

	err := tcc.CreateOrPatchNamespace(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Creating namespace %v", tcc.Namespace)

	pKey := types.NamespacedName{Name: pwlName, Namespace: tcc.Namespace}
	dKey := types.NamespacedName{Name: deploymentName, Namespace: tcc.Namespace}

	t.Log("Creating deployment")
	d := testhelpers.BuildDeployment(dKey, deploymentAppLabel)
	err = tcc.CreateWorkload(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	rs1, pods, err := tcc.CreateDeploymentReplicaSetAndPods(ctx, d)
	if err != nil {
		t.Fatalf("Unable to create pods and replicaset for deployment, %v", err)
	}

	// expect 1 container... no cloudsql instance yet
	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 1, "all")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Creating cloud sql instance")
	_, err = tcc.CreateAuthProxyWorkload(ctx, pKey, deploymentAppLabel, tcc.ConnectionString, "Deployment")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop ")
	_, err = tcc.GetAuthProxyWorkloadAfterReconcile(ctx, pKey)
	if err != nil {
		t.Fatal(err)
	}
	// user must manually trigger the pods to be recreated.
	// so we simulate that by asserting that after the update, there is only
	// 1 container on the pods.

	// expect 1 container... no cloudsql instance yet
	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 1, "all")
	if err != nil {
		t.Fatal(err)
	}

	err = recreatePodsAfterDeploymentUpdate(ctx, tcc, d, rs1, pods)
	if err != nil {
		t.Fatal(err)
	}
	// and check for 2 containers
	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 2, "all")
	if err != nil {
		t.Fatal(err)
	}
}

// TestUpdateWorkloadContainerWhenDefaultProxyImageChanges is the test that
// demonstrates that when the operator's default image changes, it will
// automatically update the proxy container image on existing deployments.
func TestUpdateWorkloadContainerWhenDefaultProxyImageChanges(t *testing.T) {
	ctx := testintegration.TestContext()
	// Use a fresh EnvTestHarness because we are messing with the operator
	// lifecycle.
	th, err := testintegration.NewTestHarness()
	defer th.Teardown()

	tcc := newTestCaseClient("updateimage", th.Client)

	err = tcc.CreateOrPatchNamespace(ctx)
	if err != nil {
		t.Fatalf("can't create namespace, %v", err)
	}

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	key := types.NamespacedName{Name: pwlName, Namespace: tcc.Namespace}

	t.Log("Creating AuthProxyWorkload")
	p, err := tcc.CreateAuthProxyWorkload(ctx, key, deploymentAppLabel, tcc.ConnectionString, "Deployment")
	if err != nil {
		t.Fatal(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = tcc.GetAuthProxyWorkloadAfterReconcile(ctx, key)
	if err != nil {
		t.Fatal("unable to create AuthProxyWorkload", err)
		return
	}

	t.Log("Creating deployment")
	d := testhelpers.BuildDeployment(key, deploymentAppLabel)
	err = tcc.CreateWorkload(ctx, d)
	if err != nil {
		t.Fatal("unable to create deployment", err)
		return
	}

	t.Log("Creating deployment replicas")
	rs, pl, err := tcc.CreateDeploymentReplicaSetAndPods(ctx, d)
	if err != nil {
		t.Fatal("unable to create pods", err)
		return
	}

	// Check that proxy container was added to pods
	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 2, "all")
	if err != nil {
		t.Fatal(err)
	}

	// Check that the pods have the expected default proxy image
	pods, err := testhelpers.ListPods(ctx, tcc.Client, tcc.Namespace, d.Spec.Selector)
	for _, p := range pods.Items {
		if got, want := p.Spec.Containers[1].Image, workload.DefaultProxyImage; got != want {
			t.Errorf("got %v, want %v image before operator upgrade", got, want)
		}
	}

	// Restart the manager with a new default proxy image
	const newDefault = "gcr.io/cloud-sql-connectors/cloud-sql-proxy:999.9.9"
	th.StopManager()
	err = th.StartManager(newDefault)
	if err != nil {
		t.Fatal("can't restart container", err)
	}

	// Get the related deployment. Make sure that annotations were
	// set on the pod template
	err = testhelpers.RetryUntilSuccess(24, testhelpers.DefaultRetryInterval, func() error {
		ud := &appsv1.Deployment{}
		err = tcc.Client.Get(ctx, client.ObjectKeyFromObject(d), ud)
		wantK, wantV := workload.PodAnnotation(p, newDefault)
		gotV := ud.Spec.Template.Annotations[wantK]
		if gotV != wantV {
			return fmt.Errorf("got %s, want %s for podspec annotation on deployment", gotV, wantV)
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	// Recreate the ReplicaSet and Pods as would happen when the deployment
	// PodTemplate changed.
	err = recreatePodsAfterDeploymentUpdate(ctx, tcc, d, rs, pl)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the new pods have the new default proxy image
	pods, err = testhelpers.ListPods(ctx, tcc.Client, tcc.Namespace, d.Spec.Selector)
	for _, p := range pods.Items {
		if got, want := p.Spec.Containers[1].Image, newDefault; got != want {
			t.Errorf("got %v, want %v image before operator upgrade", got, want)
		}
	}

}

// TestDeleteMisconfiguredPod is the test that
// demonstrates that when a pod is created, and the webhook does not work, then
// the PodDeleteController will attempt to delete the pod.
func TestDeleteMisconfiguredPod(t *testing.T) {
	ctx := testintegration.TestContext()

	// Use a fresh Manager Harness because we are messing with the operator
	// lifecycle.
	th, err := testintegration.NewTestHarness()
	defer th.Teardown()
	tcc := newTestCaseClient("deletemisconfig", th.Client)

	err = tcc.CreateOrPatchNamespace(ctx)
	if err != nil {
		t.Fatalf("can't create namespace, %v", err)
	}

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	key := types.NamespacedName{Name: pwlName, Namespace: tcc.Namespace}

	t.Log("Creating AuthProxyWorkload")
	_, err = tcc.CreateAuthProxyWorkload(ctx, key, deploymentAppLabel, tcc.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return
	}

	// Stop the manager before attempting to create the resources
	th.StopManager()
	t.Log("Manager is stopped")

	t.Log("Creating deployment")
	d := testhelpers.BuildDeployment(key, deploymentAppLabel)
	err = tcc.CreateWorkload(ctx, d)
	if err != nil {
		t.Fatal("unable to create deployment", err)
		return
	}

	t.Log("Creating deployment replicas")
	rs, pl, err := tcc.CreateDeploymentReplicaSetAndPods(ctx, d)
	if err != nil {
		t.Fatal("unable to create pods", err)
		return
	}

	// Check that proxy container was not added to pods, because the manager is stopped
	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 1, "all")
	if err != nil {
		t.Fatal(err)
	}

	pods, err := testhelpers.ListPods(ctx, tcc.Client, tcc.Namespace, d.Spec.Selector)
	if len(pods.Items) == 0 {
		t.Fatal("No pods found")
	}

	t.Log("Restarting the manager...")
	// Start the manager
	err = th.StartManager(workload.DefaultProxyImage)
	if err != nil {
		t.Fatal("can't restart container", err)
	}

	// Expect that old pods are deleted
	err = testhelpers.RetryUntilSuccess(24, testhelpers.DefaultRetryInterval, func() error {
		pods, err := testhelpers.ListPods(ctx, tcc.Client, tcc.Namespace, d.Spec.Selector)
		if err != nil {
			return err
		}
		var podCount int
		for _, pod := range pods.Items {
			if !pod.GetDeletionTimestamp().IsZero() {
				podCount++
			}
		}
		if podCount > 0 {
			return fmt.Errorf("got %v, want 0 pods that are not yet deleted", podCount)
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	// Recreate the ReplicaSet and Pods as would happen when the deployment
	// PodTemplate changed.
	err = recreatePodsAfterDeploymentUpdate(ctx, tcc, d, rs, pl)
	if err != nil {
		t.Fatal(err)
	}

}

// recreatePodsAfterDeploymentUpdate acts like the DeploymentController, when
// the Deployment is updated. It deletes the old ReplicaSet and Pods, and
// creates a new ReplicaSet and Pods.
func recreatePodsAfterDeploymentUpdate(ctx context.Context, tcc *testhelpers.TestCaseClient, d *appsv1.Deployment, rs1 *appsv1.ReplicaSet, pods []*corev1.Pod) error {
	// Then we simulate the deployment pods being replaced
	err := tcc.Client.Delete(ctx, rs1)
	if err != nil {
		return err
	}

	for i := 0; i < len(pods); i++ {
		err = tcc.Client.Delete(ctx, pods[i])
		if err != nil {
			return err
		}
	}
	_, _, err = tcc.CreateDeploymentReplicaSetAndPods(ctx, d)
	if err != nil {
		return err
	}
	return nil
}
