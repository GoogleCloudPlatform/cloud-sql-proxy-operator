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

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
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
	resource := &cloudsqlapi.AuthProxyWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cloudsqlapi.GroupVersion.String(),
			Kind:       "AuthProxyWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wantName,
			Namespace: namespace,
		},
		Spec: cloudsqlapi.AuthProxyWorkloadSpec{
			Workload: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Deployment",
				Name: "busybox",
			},
			Instances: []cloudsqlapi.InstanceSpec{{
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
	retrievedResource := &cloudsqlapi.AuthProxyWorkload{}
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
	err := CreateAuthProxyWorkload(ctx, tctx, key, "app", expectedConnStr, "Deployment")
	if err != nil {
		t.Errorf("Unable to create auth proxy workload %v", err)
		return
	}

	res, err := GetAuthProxyWorkloadAfterReconcile(ctx, tctx, key)
	if err != nil {
		t.Errorf("Unable to find entity after create %v", err)
		return
	}

	resourceYaml, _ := yaml.Marshal(res)
	t.Logf("Resource Yaml: %s", string(resourceYaml))

	if connStr := res.Spec.Instances[0].ConnectionString; connStr != expectedConnStr {
		t.Errorf("was %v, wants %v, spec.cloudSqlInstance", connStr, expectedConnStr)
	}

	if wlstatus := GetConditionStatus(res.Status.Conditions, cloudsqlapi.ConditionUpToDate); wlstatus != metav1.ConditionTrue {
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
