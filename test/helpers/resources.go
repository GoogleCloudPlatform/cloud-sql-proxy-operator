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
	"errors"
	"testing"
	"time"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateBusyboxDeployment(t *testing.T, ctx context.Context, k8sClient client.Client, key types.NamespacedName, appLabel string) (*appsv1.Deployment, error) {
	yaml := `apiVersion: apps/appsv1
kind: Deployment
metadata:
  name: busybox-deployment-
  labels:
    app: busyboxon
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: busyboxon
  template:
    metadata:
      labels:
        app: busyboxon
        enableawait: "yes"
    spec:
      containers:
      - name: busybox
        image: busybox
        imagePullPolicy: IfNotPresent
        command: ['sh', '-c', 'echo Container 1 is Running ; sleep 3600']
`
	d := appsv1.Deployment{}
	err := yaml2.Unmarshal([]byte(yaml), &d)
	if err != nil {
		return nil, err
	}
	d.ObjectMeta.Name = key.Name
	d.ObjectMeta.Namespace = key.Namespace
	d.ObjectMeta.Labels = map[string]string{"app": appLabel}

	err = k8sClient.Create(ctx, &d)
	if err != nil {
		return nil, err
	}
	cd := appsv1.Deployment{}
	err = RetryUntilSuccess(t, 5, 1*time.Second, func() error {
		return k8sClient.Get(ctx, key, &cd)
	})
	if err != nil {
		return nil, err
	}
	return &cd, nil

}

func GetConditionStatus(conditions []metav1.Condition, condType string) metav1.ConditionStatus {
	for i, _ := range conditions {
		if conditions[i].Type == (string(condType)) {
			return conditions[i].Status
		}
	}
	return ""
}

func GetAuthProxyWorkload(t *testing.T, ctx context.Context, k8sClient client.Client, podmodLookupKey types.NamespacedName) (*cloudsqlapi.AuthProxyWorkload, error) {
	createdPodmod := &cloudsqlapi.AuthProxyWorkload{}
	// We'll need to retry getting this newly created resource, given that creation may not immediately happen.
	err := RetryUntilSuccess(t, 6, 5*time.Second, func() error {
		err := k8sClient.Get(ctx, podmodLookupKey, createdPodmod)
		if err != nil {
			return err
		}
		if GetConditionStatus(createdPodmod.Status.Conditions, cloudsqlapi.ConditionUpToDate) != metav1.ConditionTrue {
			return errors.New("AuthProxyWorkload found, but reconcile not complete yet")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdPodmod, err
}

func ExpectContainerCount(t *testing.T, k8sClient client.Client, err error, ctx context.Context, deploymentLookupKey types.NamespacedName, deployment *appsv1.Deployment, count int) error {
	var got int
	err = RetryUntilSuccess(t, 6, 5*time.Second, func() error {
		err := k8sClient.Get(ctx, deploymentLookupKey, deployment)
		if err != nil {
			return err
		}
		got = len(deployment.Spec.Template.Spec.Containers)
		if got != count {
			return errors.New("Deployment found, but reconcile not complete yet")
		}
		return nil
	})

	if err != nil {
		t.Errorf("want %v containers, got %v number of containers did not resolve after waiting for reconcile", count, got)
	} else {
		t.Logf("Container len is now %v", got)
	}
	return err
}

func CreateAuthProxyWorkload(t *testing.T, k8sClient client.Client, resourceName string, resourceNamespace string, appLabel string, ctx context.Context, connectionString string, proxyImageUrl string) error {
	podmod := &cloudsqlapi.AuthProxyWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cloudsqlapi.GroupVersion.String(),
			Kind:       "AuthProxyWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: resourceNamespace,
		},
		Spec: cloudsqlapi.AuthProxyWorkloadSpec{
			Workload: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Deployment",
				Selector: &metav1.LabelSelector{
					MatchLabels:      map[string]string{"app": appLabel},
					MatchExpressions: nil,
				},
			},
			AuthProxyContainer: &cloudsqlapi.AuthProxyContainerSpec{
				Image: proxyImageUrl,
			},
			Instances: []cloudsqlapi.InstanceSpec{{
				ConnectionString: connectionString,
			}},
		},
	}
	err := k8sClient.Create(ctx, podmod)
	if err != nil {
		t.Errorf("Unable to create entity %v", err)
		return err
	}
	return nil
}
