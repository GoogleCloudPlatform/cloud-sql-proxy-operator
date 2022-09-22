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
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	proxyImageUrl := tctx.ProxyImageURL
	connectionString := tctx.ConnectionString

	testContext := context.Background()

	CreateOrPatchNamespace(t, testContext, c, ns)

	const (
		pwlName            = "newdeploy"
		deploymentName     = "busybox"
		deploymentAppLabel = "busybox"
	)
	ctx := testContext
	key := types.NamespacedName{Name: pwlName, Namespace: ns}

	t.Log("Creating cloud sql instance")
	err := CreateAuthProxyWorkload(t, c, pwlName, ns, deploymentAppLabel, ctx, connectionString, proxyImageUrl)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}
	t.Log("Waiting for cloud sql instance to begin the reconcile loop loop")
	_, err = GetAuthProxyWorkload(t, ctx, c, key)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}

	t.Log("Creating deployment")
	deploymentLookupKey := types.NamespacedName{Name: deploymentName, Namespace: ns}
	deployment, err := CreateBusyboxDeployment(t, ctx, c, deploymentLookupKey, deploymentAppLabel)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 2 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 2)
	}
	t.Log("Waiting for deployment reconcile to complete")
	err = RetryUntilSuccess(t, 6, 5*time.Second, func() error {
		err := c.Get(ctx, deploymentLookupKey, deployment)
		if err != nil {
			return err
		}
		if count := len(deployment.Spec.Template.Spec.Containers); count != 2 {
			return fmt.Errorf("deployment found, but reconcile not complete yet. only %d containers", count)
		}
		return nil
	})
	if err != nil {
		t.Errorf("number of containers did not resolve to 2 after waiting for reconcile")
	} else {
		t.Log("Container len is now 2")
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
	proxyImageUrl := tctx.ProxyImageURL
	connectionString := tctx.ConnectionString

	ctx := context.Background()
	CreateOrPatchNamespace(t, ctx, c, ns)
	t.Logf("Creating namespace %v", ns)

	csqlInstanceLookupKey := types.NamespacedName{Name: pwlName, Namespace: ns}
	deploymentLookupKey := types.NamespacedName{Name: deploymentName, Namespace: ns}

	t.Log("Creating deployment")
	deployment, err := CreateBusyboxDeployment(t, ctx, c, deploymentLookupKey, deploymentAppLabel)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}
	// expect 1 container... no cloudsql instance yet
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 1 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 1)
	}

	t.Log("Creating cloud sql instance")
	err = CreateAuthProxyWorkload(t, c, pwlName, ns, deploymentAppLabel, ctx, connectionString, proxyImageUrl)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}
	t.Log("Waiting for cloud sql instance to begin the reconcile loop loop")
	updatedI, err := GetAuthProxyWorkload(t, ctx, c, csqlInstanceLookupKey)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}
	t.Logf("status: %v", updatedI.Status)

	t.Logf("Waiting for deployment reconcile to complete")
	err = ExpectContainerCount(t, c, err, ctx, deploymentLookupKey, deployment, 2)

	updatedI, err = GetAuthProxyWorkload(t, ctx, c, csqlInstanceLookupKey)

	//TODO Add workload status to the CRD
	//t.Log("status: %{v}", updatedI.Status, len(updatedI.Status.WorkloadStatus))
	//if wlStatus := GetConditionStatus(updatedI.Status.WorkloadStatus[0].Conditions, cloudsqlv1.ConditionUpToDate); wlStatus != metav1.ConditionTrue {
	//	t.Errorf("wants %v got %v, up-to-date workload status condition", metav1.ConditionTrue, wlStatus)
	//}

	if testRemove {
		t.Logf("Deleting for cloud sql instance")
		err = c.Delete(ctx, updatedI)

		t.Logf("Waiting for deployment reconcile to complete")
		err = ExpectContainerCount(t, c, err, ctx, deploymentLookupKey, deployment, 1)
	}

}
