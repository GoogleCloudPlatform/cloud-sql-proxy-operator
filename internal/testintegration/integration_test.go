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
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testintegration"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	tp := testCaseParams(t, "modifynew")

	testhelpers.CreateOrPatchNamespace(tp.Ctx, tp)

	const (
		pwlName            = "newdeploy"
		deploymentAppLabel = "busybox"
	)
	ctx := tp.Ctx
	key := types.NamespacedName{Name: pwlName, Namespace: tp.Namespace}

	t.Log("Creating AuthProxyWorkload")
	err := testhelpers.CreateAuthProxyWorkload(ctx, tp, key, deploymentAppLabel, tp.ConnectionString, "Deployment")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Waiting for AuthProxyWorkload operator to begin the reconcile loop")
	_, err = testhelpers.GetAuthProxyWorkloadAfterReconcile(ctx, tp, key)
	if err != nil {
		t.Error("unable to create AuthProxyWorkload", err)
		return
	}

	t.Log("Creating deployment")
	d := testhelpers.NewDeployment(key, deploymentAppLabel)
	err = testhelpers.CreateWorkload(tp, d)

	if err != nil {
		t.Error("unable to create deployment", err)
		return
	}
	time.Sleep(time.Second * 5)
	t.Log("Creating deployment replicas")
	_, _, err = createDeploymentReplicaSetAndPods(tp, d)
	if err != nil {
		t.Error("unable to create pods", err)
		return
	}
	time.Sleep(time.Second * 5)

	testhelpers.RetryUntilSuccess(t, 5, 5, func() error {
		dep := &appsv1.Deployment{}
		err := tp.Client.Get(tp.Ctx, key, dep)
		if err != nil {
			return err
		}
		annKey := fmt.Sprintf("cloudsql.cloud.google.com/req-%s-%s", tp.Namespace, pwlName)
		wantAnn := "1"
		var gotAnn string
		if dep.Spec.Template.Annotations != nil {
			gotAnn = dep.Spec.Template.Annotations[annKey]
		}
		if len(dep.Spec.Template.Annotations) == 0 {
			return fmt.Errorf("Expected annotations")
		}
		if wantAnn != gotAnn {
			return fmt.Errorf("got %s, want %s for annotation named %s", gotAnn, wantAnn, annKey)
		}
		return nil
	})

	testhelpers.ExpectPodContainerCount(tp, d.GetNamespace(), d.Spec.Selector, 2, "all")
}

func TestModifiesExistingDeployment(t *testing.T) {
	tctx := testCaseParams(t, "modifyexisting")
	testRemove := testhelpers.TestModifiesExistingDeployment(tctx)
	testRemove()
}

// createDeploymentReplicaSetAndPods mimics the behavior of the deployment controller
// built into kubernetes. It creates one ReplicaSet and DeploymentSpec.Replicas pods
// with the correct labels and ownership annotations as if it were in a live cluster.
// This will make it easier to test and debug the behavior of our pod injection webhooks.
func createDeploymentReplicaSetAndPods(tp *testhelpers.TestCaseParams, d *appsv1.Deployment) (*appsv1.ReplicaSet, []*corev1.Pod, error) {
	podTemplateHash := strconv.FormatUint(rand.Uint64(), 16)
	rs := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet", APIVersion: "apps/metav1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", d.Name, podTemplateHash),
			Namespace: d.Namespace,
			Annotations: map[string]string{
				"deployment.kubernetes.io/desired-replicas": "2",
				"deployment.kubernetes.io/max-replicas":     "3",
				"deployment.kubernetes.io/revision":         "1",
			},
			Generation: 1,
			Labels: map[string]string{
				"app":               d.Spec.Template.Labels["app"],
				"enablewait":        "yes",
				"pod-template-hash": podTemplateHash,
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: d.Spec.Replicas,
			Selector: d.Spec.Selector,
			Template: d.Spec.Template,
		},
	}

	controllerutil.SetOwnerReference(d, rs, tp.Client.Scheme())
	err := tp.Client.Create(tp.Ctx, rs)
	if err != nil {
		return nil, nil, err
	}

	var replicas int32
	if d.Spec.Replicas != nil {
		replicas = *d.Spec.Replicas
	} else {
		replicas = 1
	}
	var pods []*corev1.Pod
	for i := int32(0); i < replicas; i++ {
		podID := strconv.FormatUint(uint64(rand.Uint32()), 16)
		p := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:       fmt.Sprintf("%s-%s-%s", d.Name, podTemplateHash, podID),
				Namespace:  d.Namespace,
				Generation: 1,
				Labels: map[string]string{
					"app":               d.Spec.Template.Labels["app"],
					"enablewait":        "yes",
					"pod-template-hash": podTemplateHash,
				},
			},
			Spec: d.Spec.Template.Spec,
		}
		controllerutil.SetOwnerReference(rs, p, tp.Client.Scheme())
		err = tp.Client.Create(tp.Ctx, p)
		if err != nil {
			return rs, nil, err
		}
		pods = append(pods, p)
	}
	return rs, pods, nil
}
