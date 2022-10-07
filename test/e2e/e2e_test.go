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

package e2e_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/e2e"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMain(m *testing.M) {
	teardown, err := e2e.SetupTests()
	if teardown != nil {
		defer teardown()
	}

	if err != nil {
		fmt.Println("errors while initializing e2e test", err)
		os.Exit(1)
	}

	code := m.Run()

	os.Exit(code)
}

func TestCreateResource(t *testing.T) {
	tctx := e2e.Params(t, "create")
	helpers.TestCreateResource(tctx)
}

func TestDeleteResource(t *testing.T) {
	tctx := e2e.Params(t, "delete")
	helpers.TestDeleteResource(tctx)
}

func TestModifiesNewDeployment(t *testing.T) {
	tctx := e2e.Params(t, "newdeploy")
	helpers.TestModifiesNewDeployment(tctx)

	var podList *v1.PodList
	err := helpers.RetryUntilSuccess(t, 5, 10*time.Second, func() error {
		var err error
		podList, err = e2e.ListDeploymentPods(tctx.Ctx, client.ObjectKey{Namespace: tctx.Namespace, Name: "newdeploy"})
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
	tctx := e2e.Params(t, "modifydeploy")
	helpers.TestModifiesExistingDeployment(tctx)
}
