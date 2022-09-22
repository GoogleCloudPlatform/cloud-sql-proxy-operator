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

type TestcaseContext struct {
	T                *testing.T
	Client           client.Client
	Namespace        string
	ConnectionString string
	ProxyImageURL    string
}

func NewNamespaceName(prefix string) string {
	return fmt.Sprintf("test%s%d", prefix, rand.IntnRange(1000, 9999))
}

func TestModifiesNewDeployment(tctx *TestcaseContext) {
	t := tctx.T
	c := tctx.Client
	ns := tctx.Namespace
	proxyImageURL := tctx.ProxyImageURL
	connectionString := tctx.ConnectionString

	testContext := context.Background()

	CreateOrPatchNamespace(testContext, t, c, ns)

	const (
		pwlName            = "newdeploy"
		deploymentName     = "busybox"
		deploymentAppLabel = "busybox"
	)
	ctx := testContext
	key := types.NamespacedName{Name: pwlName, Namespace: ns}

	t.Log("Creating cloud sql instance")
	err := CreateAuthProxyWorkload(ctx, t, c, types.NamespacedName{Namespace: ns, Name: pwlName}, deploymentAppLabel, connectionString, proxyImageURL)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop loop")
	_, err = GetAuthProxyWorkload(ctx, t, c, key)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Creating deployment")
	deploymentLookupKey := types.NamespacedName{Name: deploymentName, Namespace: ns}
	deployment, err := CreateBusyboxDeployment(ctx, t, c, deploymentLookupKey, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 2 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 2)
	}

	t.Log("Waiting for deployment reconcile to complete")
	err = ExpectContainerCount(ctx, t, c, deploymentLookupKey, deployment, 2)

	if err != nil {
		t.Errorf("number of containers did not resolve to 2 after waiting for reconcile")
	}
}

func TestModifiesExistingDeployment(tctx *TestcaseContext, testRemove bool) {
	const (
		pwlName            = "db-mod"
		deploymentName     = "deploy-mod"
		deploymentAppLabel = "existing-mod"
	)

	t := tctx.T
	c := tctx.Client
	ns := tctx.Namespace
	proxyImageURL := tctx.ProxyImageURL
	connectionString := tctx.ConnectionString

	ctx := context.Background()
	CreateOrPatchNamespace(ctx, t, c, ns)
	t.Logf("Creating namespace %v", ns)

	csqlInstanceLookupKey := types.NamespacedName{Name: pwlName, Namespace: ns}
	deploymentLookupKey := types.NamespacedName{Name: deploymentName, Namespace: ns}

	t.Log("Creating deployment")
	deployment, err := CreateBusyboxDeployment(ctx, t, c, deploymentLookupKey, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}
	// expect 1 container... no cloudsql instance yet
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 1 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 1)
	}

	t.Log("Creating cloud sql instance")
	err = CreateAuthProxyWorkload(ctx, t, c, types.NamespacedName{Namespace: ns, Name: pwlName}, deploymentAppLabel, connectionString, proxyImageURL)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop loop")
	updatedI, err := GetAuthProxyWorkload(ctx, t, c, csqlInstanceLookupKey)
	if err != nil {
		t.Error(err)
		return
	}
	status, _ := yaml.Marshal(updatedI.Status)

	t.Logf("status: %v", string(status))

	t.Logf("Waiting for deployment reconcile to complete")
	err = ExpectContainerCount(ctx, t, c, deploymentLookupKey, deployment, 2)
	if err != nil {
		t.Error(err)
		return
	}

	updatedI, err = GetAuthProxyWorkload(ctx, t, c, csqlInstanceLookupKey)
	if err != nil {
		t.Error(err)
		return
	}

	// TODO Add workload status to the CRD
	// t.Log("status: %{v}", updatedI.Status, len(updatedI.Status.WorkloadStatus))
	// if wlStatus := GetConditionStatus(updatedI.Status.WorkloadStatus[0].Conditions, cloudsqlv1.ConditionUpToDate); wlStatus != metav1.ConditionTrue {
	//    t.Errorf("wants %v got %v, up-to-date workload status condition", metav1.ConditionTrue, wlStatus)
	// }

	if testRemove {
		t.Logf("Deleting for cloud sql instance")
		err = c.Delete(ctx, updatedI)
		if err != nil {
			t.Error(err)
			return
		}

		t.Logf("Waiting for deployment reconcile to complete")
		err = ExpectContainerCount(ctx, t, c, deploymentLookupKey, deployment, 1)
		if err != nil {
			t.Error(err)
			return
		}
	}
}
