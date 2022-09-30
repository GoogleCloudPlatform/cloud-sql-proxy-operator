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

// Package integration_test has integration tests that run a local kubernetes
// api server and ensure that the interaction between kubernetes and the
// operator works correctly.
package integration_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/integration"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestMain(m *testing.M) {
	teardown, err := integration.EnvTestSetup()

	if err != nil {
		integration.Log.Error(err, "errors while initializing kubernetes cluster")
		if teardown != nil {
			teardown()
		}
		os.Exit(1)
	}

	code := m.Run()
	teardown()
	os.Exit(code)
}

func TestCreateResource(t *testing.T) {
	var (
		namespace   = fmt.Sprintf("testcreate-%d", rand.IntnRange(1000, 9999))
		wantName    = "instance1"
		resourceKey = types.NamespacedName{Name: wantName, Namespace: namespace}
		ctx         = integration.TestContext()
	)

	// First, set up the k8s namespace for this test.
	helpers.CreateOrPatchNamespace(ctx, t, integration.Client, namespace)

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
				ConnectionString: "project:region:instance1",
			}},
		},
	}

	// Call kubernetes to create the resource.
	err := integration.Client.Create(ctx, resource)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}

	// Wait for kubernetes to finish creating the resource, kubernetes
	// is eventually-consistent.
	retrievedResource := &cloudsqlapi.AuthProxyWorkload{}
	err = helpers.RetryUntilSuccess(t, 5, time.Second*5, func() error {
		return integration.Client.Get(ctx, resourceKey, retrievedResource)
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

func TestModifiesNewDeployment(t *testing.T) {
	tctx := &helpers.TestcaseContext{
		T:                t,
		Client:           integration.Client,
		Namespace:        helpers.NewNamespaceName("modifiesnewdeployment"),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
	}
	helpers.TestModifiesNewDeployment(tctx)
}
