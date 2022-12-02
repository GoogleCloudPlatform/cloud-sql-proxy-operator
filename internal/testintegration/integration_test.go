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
		Ctx:              testintegration.TestContext(),
	}
}

func TestCreateAndDeleteResource(t *testing.T) {
	tcc := newTestCaseClient("create")
	err := tcc.CreateAndDeleteResource()
	if err != nil {
		t.Error(err)
	}

}

func TestModifiesNewDeployment(t *testing.T) {
	tcc := newTestCaseClient("modifynew")

	err := tcc.CreateOrPatchNamespace()
	if err != nil {
		t.Fatalf("can't create namespace, %v", err)
	}

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	key := types.NamespacedName{Name: pwlName, Namespace: tcc.Namespace}

	t.Log("Creating AuthProxyWorkload")
	err = tcc.CreateAuthProxyWorkload(key, deploymentAppLabel, tcc.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = tcc.GetAuthProxyWorkloadAfterReconcile(key)
	if err != nil {
		t.Error("unable to create AuthProxyWorkload", err)
		return
	}

	t.Log("Creating deployment")
	d := testhelpers.BuildDeployment(key, deploymentAppLabel)
	err = tcc.CreateWorkload(d)

	if err != nil {
		t.Error("unable to create deployment", err)
		return
	}

	t.Log("Creating deployment replicas")
	_, _, err = tcc.CreateDeploymentReplicaSetAndPods(d)
	if err != nil {
		t.Error("unable to create pods", err)
		return
	}

	err = tcc.ExpectPodContainerCount(d.Spec.Selector, 2, "all")
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
	tp := newTestCaseClient("modifyexisting")

	err := tp.CreateOrPatchNamespace()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Creating namespace %v", tp.Namespace)

	pKey := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}
	dKey := types.NamespacedName{Name: deploymentName, Namespace: tp.Namespace}

	t.Log("Creating deployment")
	d := testhelpers.BuildDeployment(dKey, deploymentAppLabel)
	err = tp.CreateWorkload(d)
	if err != nil {
		t.Error(err)
		return
	}
	_, _, err = tp.CreateDeploymentReplicaSetAndPods(d)
	if err != nil {
		t.Errorf("Unable to create pods and replicaset for deployment, %v", err)
		return
	}

	// expect 1 container... no cloudsql instance yet
	tp.ExpectPodContainerCount(d.Spec.Selector, 1, "all")

	t.Log("Creating cloud sql instance")
	err = tp.CreateAuthProxyWorkload(pKey, deploymentAppLabel, tp.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop ")
	_, err = tp.GetAuthProxyWorkloadAfterReconcile(pKey)
	if err != nil {
		t.Error(err)
		return
	}

	//TODO implement the new reconcile algorithm before finishing this test.
	// Then, we should assert 2 containers on all pods.

}
