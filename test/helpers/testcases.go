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

package helpers

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type TestCaseParams struct {
	T                *testing.T
	Client           client.Client
	Namespace        string
	ConnectionString string
	ProxyImageURL    string
}

func NewNamespaceName(prefix string) string {
	return fmt.Sprintf("test%s%d", prefix, rand.IntnRange(1000, 9999))
}

func TestModifiesNewDeployment(tp *TestCaseParams) {
	t := tp.T
	testContext := context.Background()

	CreateOrPatchNamespace(testContext, tp)

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	ctx := testContext
	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	t.Log("Creating AuthProxyWorkload")
	err := CreateAuthProxyWorkload(ctx, tp, key,
		deploymentAppLabel, tp.ConnectionString)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = GetAuthProxyWorkload(ctx, tp, key)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Creating deployment")
	deployment, err := CreateBusyboxDeployment(ctx, tp, key, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 2 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 2)
	}

	t.Log("Waiting for deployment reconcile to complete")
	err = ExpectContainerCount(ctx, tp, key, 2)

	if err != nil {
		t.Errorf("number of containers did not resolve to 2 after waiting for reconcile")
	}
}

func TestModifiesExistingDeployment(tp *TestCaseParams) func() {
	const (
		pwlName            = "db-mod"
		deploymentName     = "deploy-mod"
		deploymentAppLabel = "existing-mod"
	)

	ctx := context.Background()
	CreateOrPatchNamespace(ctx, tp)
	tp.T.Logf("Creating namespace %v", tp.Namespace)

	pKey := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}
	dKey := types.NamespacedName{Name: deploymentName, Namespace: tp.Namespace}

	tp.T.Log("Creating deployment")
	deployment, err := CreateBusyboxDeployment(ctx, tp, dKey, deploymentAppLabel)
	if err != nil {
		tp.T.Error(err)
		return func() {}
	}
	// expect 1 container... no cloudsql instance yet
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 1 {
		tp.T.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 1)
	}

	tp.T.Log("Creating cloud sql instance")
	err = CreateAuthProxyWorkload(ctx, tp, pKey, deploymentAppLabel, tp.ConnectionString)
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}

	tp.T.Log("Waiting for cloud sql instance to begin the reconcile loop loop")
	updatedI, err := GetAuthProxyWorkload(ctx, tp, pKey)
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}
	status, _ := yaml.Marshal(updatedI.Status)

	tp.T.Logf("status: %v", string(status))

	tp.T.Logf("Waiting for deployment reconcile to complete")
	err = ExpectContainerCount(ctx, tp, dKey, 2)
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}

	updatedI, err = GetAuthProxyWorkload(ctx, tp, pKey)
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}

	// TODO Add workload status to the CRD
	// t.Log("status: %{v}", updatedI.Status, len(updatedI.Status.WorkloadStatus))
	// if wlStatus := GetConditionStatus(updatedI.Status.WorkloadStatus[0].Conditions, cloudsqlv1.ConditionUpToDate); wlStatus != metav1.ConditionTrue {
	//    t.Errorf("wants %v got %v, up-to-date workload status condition", metav1.ConditionTrue, wlStatus)
	// }

	return func() {
		tp.T.Logf("Deleting for cloud sql instance")
		err = tp.Client.Delete(ctx, updatedI)
		if err != nil {
			tp.T.Error(err)
			return
		}

		tp.T.Logf("Waiting for deployment reconcile to complete")
		err = ExpectContainerCount(ctx, tp, dKey, 1)
		if err != nil {
			tp.T.Error(err)
			return
		}
	}
}
