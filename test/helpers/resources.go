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
	"time"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateBusyboxDeployment creates a simple busybox deployment, using the
// key as its namespace and name. It also sets the label "app"= appLabel
func CreateBusyboxDeployment(
	ctx context.Context,
	t *testing.T,
	k8sClient client.Client,
	key types.NamespacedName,
	appLabel string,
) (*appsv1.Deployment, error) {
	t.Helper()
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

// GetAuthProxyWorkload finds an AuthProxyWorkload resource named key, waits for its
// "UpToDate" condition to be "True", and the returns it. Fails after 30 seconds
// if the containers does not match.
func GetAuthProxyWorkload(
	ctx context.Context,
	t *testing.T,
	k8sClient client.Client,
	key types.NamespacedName,
) (*cloudsqlapi.AuthProxyWorkload, error) {
	t.Helper()
	createdPodmod := &cloudsqlapi.AuthProxyWorkload{}
	// We'll need to retry getting this newly created resource, given that creation may not immediately happen.
	err := RetryUntilSuccess(t, 6, 5*time.Second, func() error {
		err := k8sClient.Get(ctx, key, createdPodmod)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdPodmod, err
}

// ExpectContainerCount finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func ExpectContainerCount(
	ctx context.Context,
	t *testing.T,
	k8sClient client.Client,
	key types.NamespacedName,
	deployment *appsv1.Deployment,
	count int,
) error {
	t.Helper()
	var got int
	err := RetryUntilSuccess(t, 6, 5*time.Second, func() error {
		err := k8sClient.Get(ctx, key, deployment)
		if err != nil {
			return err
		}
		got = len(deployment.Spec.Template.Spec.Containers)
		if got != count {
			return fmt.Errorf("deployment found, got %v, want %v containers", got, count)
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

// CreateAuthProxyWorkload creates an AuthProxyWorkload in the kubernetes cluster
func CreateAuthProxyWorkload(
	ctx context.Context,
	t *testing.T,
	k8sClient client.Client,
	key types.NamespacedName,
	appLabel string,
	connectionString string,
	proxyImageURL string,
) error {
	t.Helper()
	podmod := &cloudsqlapi.AuthProxyWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cloudsqlapi.GroupVersion.String(),
			Kind:       "AuthProxyWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
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
				Image: proxyImageURL,
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
