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

package helpers

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestCaseParams struct {
	T                *testing.T
	Client           client.Client
	Namespace        string
	ConnectionString string
	ProxyImageURL    string
}

func NewNamespaceName(prefix string) string {
	return fmt.Sprintf("test%s%d", prefix, rand.IntnRange(1000, 9999))
}

func TestModifiesNewDeployment(tp *TestCaseParams) {
	t := tp.T
	testContext := context.Background()

	CreateOrPatchNamespace(testContext, tp)

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	ctx := testContext
	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	t.Log("Creating cloud sql instance")
	err := CreateAuthProxyWorkload(ctx, tp, key,
		deploymentAppLabel, tp.ConnectionString)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for cloud sql instance to begin the reconcile loop loop")
	_, err = GetAuthProxyWorkload(ctx, tp, key)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Creating deployment")
	deployment, err := CreateBusyboxDeployment(ctx, tp, key.Name, deploymentAppLabel)
	if err != nil {
		t.Error(err)
		return
	}
	containerLen := len(deployment.Spec.Template.Spec.Containers)
	if containerLen != 2 {
		t.Errorf("was %v, wants %v. number of containers. It should be set by the admission controller.", containerLen, 2)
	}

	t.Log("Waiting for deployment reconcile to complete")
	err = ExpectContainerCount(ctx, tp, key, 2)

	if err != nil {
		t.Errorf("number of containers did not resolve to 2 after waiting for reconcile")
	}
}
