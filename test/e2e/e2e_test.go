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

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/e2e"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMain(m *testing.M) {
	teardown, err := e2e.E2eTestSetup()
	if teardown != nil {
		defer teardown()
	}

	if err != nil {
		e2e.Log.Error(err, "errors while initializing e2e test connection")
		os.Exit(1)
	}

	code := m.Run()

	os.Exit(code)
}

func TestCreateEntity(t *testing.T) {
	ctx := e2e.TestContext()

	const cloudSqlInstanceName = "podmod-busybox"
	var ns = fmt.Sprintf("testcreate%d", rand.IntnRange(1000, 9999))

	helpers.CreateOrPatchNamespace(t, ctx, e2e.Client, ns)

	err := helpers.CreateAuthProxyWorkload(t, e2e.Client, cloudSqlInstanceName, ns, "app", ctx, "proj:inst:db", e2e.ProxyImageURL)
	if err != nil {
		t.Errorf("Error %v", err)
		return
	}

	podmodLookupKey := types.NamespacedName{Name: cloudSqlInstanceName, Namespace: ns}

	createdPodmod, err := helpers.GetAuthProxyWorkload(t, ctx, e2e.Client, podmodLookupKey)
	if err != nil {
		t.Errorf("Unable to find entity after create %v", err)
		return
	}

	if want, got := "proj:inst:db", createdPodmod.Spec.Instances[0].ConnectionString; got != want {
		t.Errorf("got %v, wants %v, spec.cloudSqlInstance", got, want)
	}

	if want, got := metav1.ConditionTrue, helpers.GetConditionStatus(createdPodmod.Status.Conditions, cloudsqlapi.ConditionUpToDate); want != got {
		t.Errorf("got %v, wants %v, status.condition[up-to-date]", got, want)
	}

}

func TestModifiesNewDeployment(t *testing.T) {
	ctx := e2e.TestContext()
	ns := helpers.NewNamespaceName("newdeploy")
	tctx := &helpers.TestcaseContext{
		T:                t,
		Client:           e2e.Client,
		Namespace:        ns,
		ConnectionString: e2e.Infrastructure.InstanceConnectionString,
		ProxyImageURL:    e2e.ProxyImageURL,
	}
	helpers.TestModifiesNewDeployment(tctx)

	time.Sleep(1 * time.Second) // pause a moment for the k8s api to catch up

	podList, err := e2e.ListDeploymentPods(ctx, client.ObjectKey{Namespace: ns, Name: "busybox"})
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
	tctx := &helpers.TestcaseContext{
		T:                t,
		Client:           e2e.Client,
		Namespace:        helpers.NewNamespaceName("existingdeploy"),
		ConnectionString: e2e.Infrastructure.InstanceConnectionString,
		ProxyImageURL:    e2e.ProxyImageURL,
	}
	helpers.TestModifiesExistingDeployment(tctx, false)
}
