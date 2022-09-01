/*
Copyright 2022 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
	teardown, err := integration.EnvTestSetup(m)
	if teardown != nil {
		defer teardown()
	}

	if err != nil {
		integration.Log.Error(err, "errors while initializing kubernetes cluster")
		os.Exit(1)
	}
	code := m.Run()

	os.Exit(code)
}

func TestCreateResource(t *testing.T) {
	var (
		namespace   = fmt.Sprintf("testcreate-%d", rand.IntnRange(1000, 9999))
		wantName    = "instance1"
		resourceKey = types.NamespacedName{Name: wantName, Namespace: namespace}
	)

	// First, set up the k8s namespace for this test.
	helpers.CreateOrPatchNamespace(t, integration.Ctx, integration.Client, namespace)

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
	}

	// Call kubernetes to create the resource.
	err := integration.Client.Create(integration.Ctx, resource)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}

	// Wait for kubernetes to finish creating the resource, kubernetes
	// is eventually-consistent.
	retrievedResource := &cloudsqlapi.AuthProxyWorkload{}
	err = helpers.RetryUntilSuccess(t, 5, time.Second*5, func() error {
		return integration.Client.Get(integration.Ctx, resourceKey, retrievedResource)
	})
	if err != nil {
		t.Errorf("Unable to find entity after create %v", err)
		return
	}

	// Test the contents of the resource that was retrieved from kubernetes.
	if got := retrievedResource.GetName(); got != wantName {
		t.Errorf("got %v, want %v resource wantName", got, wantName)
	}
}
