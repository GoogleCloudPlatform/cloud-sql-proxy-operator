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
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testintegration"
	"k8s.io/apimachinery/pkg/types"
)

func TestMain(m *testing.M) {
	teardown, err := testintegration.EnvTestSetup()

	if err != nil {
		testintegration.Log.Error(err, "errors while initializing kubernetes cluster")
		if teardown != nil {
			teardown()
		}
		os.Exit(1)
	}

	code := m.Run()
	teardown()
	os.Exit(code)
}

func newTestCaseClient(name string) *testhelpers.TestCaseClient {
	return &testhelpers.TestCaseClient{
		Client:           testintegration.Client,
		Namespace:        testhelpers.NewNamespaceName(name),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
	}
}

func TestCreateAndDeleteResource(t *testing.T) {
	ctx := testintegration.TestContext()
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

func TestModifiesNewDeployment(t *testing.T) {
	ctx := testintegration.TestContext()
	tcc := newTestCaseClient("modifynew")

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
	tcc := newTestCaseClient("modifyexisting")

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

	// Then we simulate the deployment pods being replaced
	err = tcc.Client.Delete(ctx, rs1)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(pods); i++ {
		err = tcc.Client.Delete(ctx, pods[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	_, _, err = tcc.CreateDeploymentReplicaSetAndPods(ctx, d)
	if err != nil {
		t.Fatal(err)
	}

	// and check for 2 containers
	err = tcc.ExpectPodContainerCount(ctx, d.Spec.Selector, 2, "all")
	if err != nil {
		t.Fatal(err)
	}
}
