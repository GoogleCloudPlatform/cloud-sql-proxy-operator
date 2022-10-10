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

// Package integration_test has testintegration tests that run a local kubernetes
// api server and ensure that the interaction between kubernetes and the
// operator works correctly.
package testintegration_test

import (
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testintegration"
)

func TestMain(m *testing.M) {
	teardown, err := testintegration.EnvTestSetup()

	if err != nil {
		testintegration.Log.Error(err, "errors while initializing kubernetes cluster")
		if teardown != nil {
			teardown()
		}
		os.Exit(1)
	}

	code := m.Run()
	teardown()
	os.Exit(code)
}

func testCaseParams(t *testing.T, name string) *testhelpers.TestCaseParams {
	return &testhelpers.TestCaseParams{
		T:                t,
		Client:           testintegration.Client,
		Namespace:        testhelpers.NewNamespaceName(name),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
		Ctx:              testintegration.TestContext(),
	}
}

func TestCreateResource(t *testing.T) {
	tctx := testCaseParams(t, "create")
	testhelpers.TestCreateResource(tctx)

}

func TestDeleteResource(t *testing.T) {
	tctx := testCaseParams(t, "delete")
	testhelpers.TestDeleteResource(tctx)

}

func TestModifiesNewDeployment(t *testing.T) {
	tctx := testCaseParams(t, "modifynew")
	testhelpers.TestModifiesNewDeployment(tctx)
}

func TestModifiesExistingDeployment(t *testing.T) {
	tctx := testCaseParams(t, "modifyexisting")
	testRemove := testhelpers.TestModifiesExistingDeployment(tctx)
	testRemove()
}
