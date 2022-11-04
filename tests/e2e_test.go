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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
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

func TestCreateResource(t *testing.T) {
	tctx := params(t, "create")
	testhelpers.TestCreateResource(tctx)
}

func TestDeleteResource(t *testing.T) {
	tctx := params(t, "delete")
	testhelpers.TestDeleteResource(tctx)
}
func TestModifiesNewDeployment(t *testing.T) {
	tp := params(t, "modifynew")

	testhelpers.CreateOrPatchNamespace(tp.Ctx, tp)

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	ctx := tp.Ctx
	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	t.Log("Creating AuthProxyWorkload")
	err := testhelpers.CreateAuthProxyWorkload(ctx, tp, key,
		deploymentAppLabel, tp.ConnectionString)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = testhelpers.GetAuthProxyWorkloadAfterReconcile(ctx, tp, key)
	if err != nil {
		t.Error("unable to create AuthProxyWorkload", err)
		return
	}

	t.Log("Creating deployment")
	d, err := testhelpers.CreateBusyboxDeployment(ctx, tp, key, deploymentAppLabel)
	if err != nil {
		t.Error("unable to create deployment", err)
		return
	}

	testhelpers.ExpectContainerCount(tp, d, 2)

}

func TestModifiesExistingDeployment(t *testing.T) {
	const (
		pwlName            = "db-mod"
		deploymentName     = "deploy-mod"
		deploymentAppLabel = "existing-mod"
	)
	var (
		tp  = params(t, "modifyexisting")
		ctx = tp.Ctx
	)

	testhelpers.CreateOrPatchNamespace(ctx, tp)
	t.Logf("Creating namespace %v", tp.Namespace)

	pKey := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}
	dKey := types.NamespacedName{Name: deploymentName, Namespace: tp.Namespace}

	t.Log("Creating deployment")
	deployment, err := testhelpers.CreateBusyboxDeployment(ctx, tp, dKey, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}

	// expect 1 container... no cloudsql instance yet
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 1 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 1)
		return
	}

	t.Log("Creating cloud sql instance")
	err = testhelpers.CreateAuthProxyWorkload(ctx, tp, pKey, deploymentAppLabel, tp.ConnectionString)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop ")
	_, err = testhelpers.GetAuthProxyWorkloadAfterReconcile(ctx, tp, pKey)
	if err != nil {
		t.Error(err)
		return
	}

	testhelpers.ExpectContainerCount(tp, deployment, 2)

}
