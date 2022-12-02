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
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestCreateAndDeleteResource(t *testing.T) {
	tcc := newTestCaseClient("create")
	res, err := tcc.CreateResource(tcc.Ctx)
	if err != nil {
		t.Error(err)
	}
	err = tcc.WaitForFinalizerOnResource(tcc.Ctx, res)
	if err != nil {
		t.Error(err)
	}
	err = tcc.DeleteResourceAndWait(tcc.Ctx, res)
	if err != nil {
		t.Error(err)
	}

}

func TestModifiesNewDeployment(t *testing.T) {
	tcc := newTestCaseClient("newdeploy")
	testhelpers.TestModifiesNewDeployment(tcc, t)

	var podList *v1.PodList
	err := testhelpers.RetryUntilSuccess(5, testhelpers.DefaultRetryInterval, func() error {
		var err error
		podList, err = listDeploymentPods(tcc.Ctx, client.ObjectKey{Namespace: tcc.Namespace, Name: "newdeploy"})
		return err
	})

	if err != nil {
		t.Fatalf("Error while listing pods for deployment %v", err)
	}
	if podCount := len(podList.Items); podCount == 0 {
		t.Fatalf("got %v pods, wants more than 0", podCount)
	}
	if containerCount := len(podList.Items[0].Spec.Containers); containerCount != 2 {
		t.Errorf("got %v containers, wants 2", containerCount)
	}
}

func TestModifiesExistingDeployment(t *testing.T) {
	tcc := newTestCaseClient("modifydeploy")
	testhelpers.TestModifiesExistingDeployment(tcc, t)
}
