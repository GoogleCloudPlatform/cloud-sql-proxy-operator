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

package testhelpers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type TestCaseParams struct {
	Ctx              context.Context
	T                *testing.T
	Client           client.Client
	Namespace        string
	ConnectionString string
	ProxyImageURL    string
}

const testcaseParamsKey = "TestCaseParams"

func WithTestCaseParams(ctx context.Context, tp *TestCaseParams) context.Context {
	return context.WithValue(ctx, testcaseParamsKey, tp)
}
func TestCaseParamsFromContext(ctx context.Context) (*TestCaseParams, bool) {
	u, ok := ctx.Value(testcaseParamsKey).(*TestCaseParams)
	return u, ok
}

func NewNamespaceName(prefix string) string {
	return fmt.Sprintf("test%s%d", prefix, rand.IntnRange(1000, 9999))
}

func TestCreateResource(tctx *TestCaseParams) {

	var (
		namespace   = tctx.Namespace
		wantName    = "instance1"
		resourceKey = types.NamespacedName{Name: wantName, Namespace: namespace}
		ctx         = tctx.Ctx
		t           = tctx.T
	)

	// First, set up the k8s namespace for this test.
	CreateOrPatchNamespace(ctx, tctx)

	// Fill in the resource with appropriate details.
	resource := &v1alpha1.AuthProxyWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "AuthProxyWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wantName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AuthProxyWorkloadSpec{
			Workload: v1alpha1.WorkloadSelectorSpec{
				Kind: "Deployment",
				Name: "busybox",
			},
			Instances: []v1alpha1.InstanceSpec{{
				ConnectionString: tctx.ConnectionString,
			}},
		},
	}

	// Call kubernetes to create the resource.
	err := tctx.Client.Create(ctx, resource)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}

	// Wait for kubernetes to finish creating the resource, kubernetes
	// is eventually-consistent.
	retrievedResource := &v1alpha1.AuthProxyWorkload{}
	err = RetryUntilSuccess(t, 5, time.Second*5, func() error {
		return tctx.Client.Get(ctx, resourceKey, retrievedResource)
	})
	if err != nil {
		t.Errorf("unable to find entity after create %v", err)
		return
	}

	// Test the contents of the resource that was retrieved from kubernetes.
	if got := retrievedResource.GetName(); got != wantName {
		t.Errorf("got %v, want %v resource wantName", got, wantName)
	}
}

func TestDeleteResource(tctx *TestCaseParams) {
	const (
		name            = "instance1"
		expectedConnStr = "proj:inst:db"
	)
	var (
		ns  = tctx.Namespace
		t   = tctx.T
		ctx = tctx.Ctx
	)
	CreateOrPatchNamespace(ctx, tctx)
	key := types.NamespacedName{Name: name, Namespace: ns}
	err := tctx.CreateAuthProxyWorkload(key, "app", expectedConnStr, "Deployment")
	if err != nil {
		t.Errorf("Unable to create auth proxy workload %v", err)
		return
	}

	res, err := tctx.GetAuthProxyWorkload(key)
	if err != nil {
		t.Errorf("Unable to find entity after create %v", err)
		return
	}

	resourceYaml, _ := yaml.Marshal(res)
	t.Logf("Resource Yaml: %s", string(resourceYaml))

	if connStr := res.Spec.Instances[0].ConnectionString; connStr != expectedConnStr {
		t.Errorf("was %v, wants %v, spec.cloudSqlInstance", connStr, expectedConnStr)
	}

	if wlstatus := GetConditionStatus(res.Status.Conditions, v1alpha1.ConditionUpToDate); wlstatus != metav1.ConditionTrue {
		t.Errorf("was %v, wants %v, status.condition[up-to-date]", wlstatus, metav1.ConditionTrue)
	}

	// Make sure the finalizer was added before deleting the resource.
	err = RetryUntilSuccess(t, 3, 5*time.Second, func() error {
		err = tctx.Client.Get(ctx, key, res)
		if len(res.Finalizers) == 0 {
			return errors.New("waiting for finalizer to be set")
		}
		return nil
	})

	err = tctx.Client.Delete(ctx, res)
	if err != nil {
		t.Error(err)
	}

	err = RetryUntilSuccess(t, 3, 5*time.Second, func() error {
		err = tctx.Client.Get(ctx, key, res)
		// The test passes when this returns an error,
		// because that means the resource was deleted.
		if err != nil {
			return nil
		}
		return fmt.Errorf("was nil, wants error when looking up deleted AuthProxyWorkload resource")
	})
	if err != nil {
		t.Error(err)
	}

}

func TestModifiesNewDeployment(tp *TestCaseParams) {
	t := tp.T
	testContext := tp.Ctx

	CreateOrPatchNamespace(testContext, tp)

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	t.Log("Creating AuthProxyWorkload")
	err := tp.CreateAuthProxyWorkload(key, deploymentAppLabel, tp.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = tp.GetAuthProxyWorkload(key)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Creating deployment")
	deployment, err := tp.CreateBusyboxDeployment(key, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 2 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 2)
	}

	t.Log("Waiting for deployment reconcile to complete")
	err = tp.ExpectContainerCount(key, 2)

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

	ctx := tp.Ctx
	CreateOrPatchNamespace(ctx, tp)
	tp.T.Logf("Creating namespace %v", tp.Namespace)

	pKey := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}
	dKey := types.NamespacedName{Name: deploymentName, Namespace: tp.Namespace}

	tp.T.Log("Creating deployment")
	deployment, err := tp.CreateBusyboxDeployment(dKey, deploymentAppLabel)
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
	err = tp.CreateAuthProxyWorkload(pKey, deploymentAppLabel, tp.ConnectionString, "Deployment")
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}

	tp.T.Log("Waiting for cloud sql instance to begin the reconcile loop ")
	updatedI, err := tp.GetAuthProxyWorkload(pKey)
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}
	status, _ := yaml.Marshal(updatedI.Status)

	tp.T.Logf("status: %v", string(status))

	tp.T.Logf("Waiting for deployment reconcile to complete")
	err = tp.ExpectContainerCount(dKey, 2)
	if err != nil {
		tp.T.Error(err)
		return func() {}

	}

	updatedI, err = tp.GetAuthProxyWorkload(pKey)
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
		err = tp.ExpectContainerCount(dKey, 1)
		if err != nil {
			tp.T.Error(err)
			return
		}
	}
}
