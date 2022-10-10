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

package testhelpers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
)

const busyboxDeployYaml = `apiVersion: apps/appsv1
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

// CreateBusyboxDeployment creates a simple busybox deployment, using the
// key as its namespace and name. It also sets the label "app"= appLabel.
func CreateBusyboxDeployment(ctx context.Context, tctx *TestCaseParams,
	name types.NamespacedName, appLabel string) (*appsv1.Deployment, error) {
	tctx.T.Helper()

	d := &appsv1.Deployment{}

	err := yaml2.Unmarshal([]byte(busyboxDeployYaml), &d)
	if err != nil {
		return nil, err
	}
	d.Name = name.Name
	d.Namespace = name.Namespace
	d.Labels = map[string]string{"app": appLabel}

	err = tctx.Client.Create(ctx, d)
	if err != nil {
		return nil, err
	}

	cd := &appsv1.Deployment{}
	err = RetryUntilSuccess(tctx.T, 5, 1*time.Second, func() error {
		return tctx.Client.Get(ctx, types.NamespacedName{
			Namespace: name.Namespace,
			Name:      name.Name,
		}, cd)
	})
	if err != nil {
		return nil, err
	}
	return cd, nil
}

// GetAuthProxyWorkload finds an AuthProxyWorkload resource named key, waits for its
// "UpToDate" condition to be "True", and the returns it. Fails after 30 seconds
// if the containers does not match.
func GetAuthProxyWorkload(ctx context.Context, tctx *TestCaseParams,
	key types.NamespacedName) (*v1alpha1.AuthProxyWorkload, error) {
	tctx.T.Helper()
	createdPodmod := &v1alpha1.AuthProxyWorkload{}
	// We'll need to retry getting this newly created resource, given that creation may not immediately happen.
	err := RetryUntilSuccess(tctx.T, 6, 5*time.Second, func() error {
		err := tctx.Client.Get(ctx, key, createdPodmod)
		if err != nil {
			return err
		}
		if GetConditionStatus(createdPodmod.Status.Conditions, v1alpha1.ConditionUpToDate) != metav1.ConditionTrue {
			return errors.New("AuthProxyWorkload found, but reconcile not complete yet")
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
func ExpectContainerCount(ctx context.Context, tctx *TestCaseParams, key types.NamespacedName, count int) error {

	tctx.T.Helper()

	var (
		got        int
		deployment = &appsv1.Deployment{}
	)
	err := RetryUntilSuccess(tctx.T, 6, 5*time.Second, func() error {
		err := tctx.Client.Get(ctx, key, deployment)
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
		tctx.T.Errorf("want %v containers, got %v number of containers did not resolve after waiting for reconcile", count, got)
		return err
	}

	tctx.T.Logf("Container len is now %v", got)
	return nil
}

func BuildAuthProxyWorkload(key types.NamespacedName, connectionString string) *v1alpha1.AuthProxyWorkload {
	return &v1alpha1.AuthProxyWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "AuthProxyWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: v1alpha1.AuthProxyWorkloadSpec{
			Instances: []v1alpha1.InstanceSpec{{
				ConnectionString: connectionString,
			}},
		},
	}
}

// CreateAuthProxyWorkload creates an AuthProxyWorkload in the kubernetes cluster.
func CreateAuthProxyWorkload(ctx context.Context, tctx *TestCaseParams,
	key types.NamespacedName, appLabel string, connectionString string) error {
	tctx.T.Helper()
	p := BuildAuthProxyWorkload(key, connectionString)
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind: "Deployment",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": appLabel},
		},
	}
	p.Spec.AuthProxyContainer = &v1alpha1.AuthProxyContainerSpec{Image: tctx.ProxyImageURL}
	err := tctx.Client.Create(ctx, p)
	if err != nil {
		tctx.T.Errorf("Unable to create entity %v", err)
		return err
	}
	return nil
}

// GetConditionStatus finds a condition where Condition.Type == condType and returns
// the status, or "" if no condition was found.
func GetConditionStatus(conditions []*metav1.Condition, condType string) metav1.ConditionStatus {
	for i := range conditions {
		if conditions[i].Type == condType {
			return conditions[i].Status
		}
	}
	return ""
}
