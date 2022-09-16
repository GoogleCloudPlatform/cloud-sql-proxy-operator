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
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/integration"
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
	tctx := &helpers.TestCaseParams{
		T:                t,
		Client:           integration.Client,
		Namespace:        helpers.NewNamespaceName("create"),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
		Ctx:              integration.TestContext(),
	}
	helpers.TestCreateResource(tctx)

}

func TestDeleteResource(t *testing.T) {
	tctx := &helpers.TestCaseParams{
		T:                t,
		Client:           integration.Client,
		Namespace:        helpers.NewNamespaceName("delete"),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
		Ctx:              integration.TestContext(),
	}

	helpers.TestDeleteResource(tctx)

}

func TestModifiesNewDeployment(t *testing.T) {
	tctx := &helpers.TestCaseParams{
		T:                t,
		Client:           integration.Client,
		Namespace:        helpers.NewNamespaceName("modifiesnewdeployment"),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
		Ctx:              integration.TestContext(),
	}
	helpers.TestModifiesNewDeployment(tctx)
}

func TestModifiesExistingDeployment(t *testing.T) {
	tctx := &helpers.TestCaseParams{
		T:                t,
		Client:           integration.Client,
		Namespace:        helpers.NewNamespaceName("modifiesexistingdeployment"),
		ConnectionString: "region:project:inst",
		ProxyImageURL:    "proxy-image:latest",
		Ctx:              integration.TestContext(),
	}
	testRemove := helpers.TestModifiesExistingDeployment(tctx)
	testRemove()
}
