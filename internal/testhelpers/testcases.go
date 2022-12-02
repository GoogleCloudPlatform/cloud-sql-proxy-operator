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

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type TestCaseClient struct {
	Ctx              context.Context
	Client           client.Client
	Namespace        string
	ConnectionString string
	ProxyImageURL    string
}

func NewNamespaceName(prefix string) string {
	return fmt.Sprintf("test%s%d", prefix, rand.IntnRange(1000, 9999))
}

// CreateResource creates a new workload resource in the TestCaseClient's namespace
// waits until the resource exists.
func (cc *TestCaseClient) CreateResource(_ context.Context) (*v1alpha1.AuthProxyWorkload, error) {
	const (
		name            = "instance1"
		expectedConnStr = "proj:inst:db"
	)
	var (
		ns = cc.Namespace
	)
	err := cc.CreateOrPatchNamespace()
	if err != nil {
		return nil, fmt.Errorf("can't create namespace, %v", err)
	}
	key := types.NamespacedName{Name: name, Namespace: ns}
	err = cc.CreateAuthProxyWorkload(key, "app", expectedConnStr, "Deployment")
	if err != nil {
		return nil, fmt.Errorf("unable to create auth proxy workload %v", err)
	}

	res, err := cc.GetAuthProxyWorkloadAfterReconcile(key)
	if err != nil {
		return nil, fmt.Errorf("unable to find entity after create %v", err)
	}

	if connStr := res.Spec.Instances[0].ConnectionString; connStr != expectedConnStr {
		return nil, fmt.Errorf("was %v, wants %v, spec.cloudSqlInstance", connStr, expectedConnStr)
	}

	if wlstatus := GetConditionStatus(res.Status.Conditions, v1alpha1.ConditionUpToDate); wlstatus != metav1.ConditionTrue {
		return nil, fmt.Errorf("was %v, wants %v, status.condition[up-to-date]", wlstatus, metav1.ConditionTrue)
	}
	return res, nil
}

// WaitForFinalizerOnResource queries the client to see if the resource has
// a finalizer.
func (cc *TestCaseClient) WaitForFinalizerOnResource(ctx context.Context, res *v1alpha1.AuthProxyWorkload) error {

	// Make sure the finalizer was added before deleting the resource.
	return RetryUntilSuccess(3, DefaultRetryInterval, func() error {
		err := cc.Client.Get(ctx, client.ObjectKeyFromObject(res), res)
		if err != nil {
			return err
		}
		if len(res.Finalizers) == 0 {
			return errors.New("waiting for finalizer to be set")
		}
		return nil
	})
}

// DeleteResourceAndWait issues a delete request for the resource and then waits for the resource
// to actually be deleted. This will return an error if the resource is not deleted within 15 seconds.
func (cc *TestCaseClient) DeleteResourceAndWait(ctx context.Context, res *v1alpha1.AuthProxyWorkload) error {

	err := cc.Client.Delete(ctx, res)
	if err != nil {
		return err
	}

	err = RetryUntilSuccess(3, DefaultRetryInterval, func() error {
		err = cc.Client.Get(ctx, client.ObjectKeyFromObject(res), res)
		// The test passes when this returns an error,
		// because that means the resource was deleted.
		if err != nil {
			return nil
		}
		return fmt.Errorf("was nil, wants error when looking up deleted AuthProxyWorkload resource")
	})

	if err != nil {
		return err
	}

	return nil
}

func TestModifiesNewDeployment(tcc *TestCaseClient, t *testing.T) {
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
		t.Error(err)
		return
	}

	t.Log("Creating deployment")
	deployment, err := tcc.CreateBusyboxDeployment(key, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 2 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 2)
	}

	t.Log("Waiting for deployment reconcile to complete")
	err = tcc.ExpectContainerCount(key, 2)

	if err != nil {
		t.Errorf("number of containers did not resolve to 2 after waiting for reconcile")
	}
}

func TestModifiesExistingDeployment(tcc *TestCaseClient, t *testing.T) func() {
	const (
		pwlName            = "db-mod"
		deploymentName     = "deploy-mod"
		deploymentAppLabel = "existing-mod"
	)

	ctx := tcc.Ctx
	err := tcc.CreateOrPatchNamespace()
	if err != nil {
		t.Fatalf("can't create namespace, %v", err)
	}
	t.Logf("Creating namespace %v", tcc.Namespace)

	pKey := types.NamespacedName{Name: pwlName, Namespace: tcc.Namespace}
	dKey := types.NamespacedName{Name: deploymentName, Namespace: tcc.Namespace}

	t.Log("Creating deployment")
	deployment, err := tcc.CreateBusyboxDeployment(dKey, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return func() {}
	}
	// expect 1 container... no cloudsql instance yet
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 1 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 1)
	}

	t.Log("Creating cloud sql instance")
	err = tcc.CreateAuthProxyWorkload(pKey, deploymentAppLabel, tcc.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return func() {}

	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop ")
	updatedI, err := tcc.GetAuthProxyWorkloadAfterReconcile(pKey)
	if err != nil {
		t.Error(err)
		return func() {}

	}
	status, _ := yaml.Marshal(updatedI.Status)

	t.Logf("status: %v", string(status))

	t.Logf("Waiting for deployment reconcile to complete")
	err = tcc.ExpectContainerCount(dKey, 2)
	if err != nil {
		t.Error(err)
		return func() {}

	}

	updatedI, err = tcc.GetAuthProxyWorkloadAfterReconcile(pKey)
	if err != nil {
		t.Error(err)
		return func() {}

	}

	// TODO Add workload status to the CRD
	// t.Log("status: %{v}", updatedI.Status, len(updatedI.Status.WorkloadStatus))
	// if wlStatus := GetConditionStatus(updatedI.Status.WorkloadStatus[0].Conditions, cloudsqlv1.ConditionUpToDate); wlStatus != metav1.ConditionTrue {
	//    t.Errorf("wants %v got %v, up-to-date workload status condition", metav1.ConditionTrue, wlStatus)
	// }

	return func() {
		t.Logf("Deleting for cloud sql instance")
		err = tcc.Client.Delete(ctx, updatedI)
		if err != nil {
			t.Error(err)
			return
		}

		t.Logf("Waiting for deployment reconcile to complete")
		err = tcc.ExpectContainerCount(dKey, 1)
		if err != nil {
			t.Error(err)
			return
		}
	}
}
