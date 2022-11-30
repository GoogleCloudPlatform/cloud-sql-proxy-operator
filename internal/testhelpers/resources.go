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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func BuildDeployment(name types.NamespacedName, appLabel string) *appsv1.Deployment {
	var two int32 = 2
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &two,
			Strategy: appsv1.DeploymentStrategy{Type: "RollingUpdate"},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busyboxon"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "busyboxon"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "busybox",
						Image:           "busybox",
						ImagePullPolicy: "IfNotPresent",
						Command:         []string{"sh", "-c", "echo Container 1 is Running ; sleep 30"},
					}},
				},
			},
		},
	}
}

func (p *TestCaseParams) CreateWorkload(o client.Object) error {
	p.T.Helper()

	err := p.Client.Create(p.Ctx, o)
	if err != nil {
		return err
	}

	err = RetryUntilSuccess(p.T, 5, 1*time.Second, func() error {
		return p.Client.Get(p.Ctx, client.ObjectKeyFromObject(o), o)
	})
	if err != nil {
		return err
	}
	return nil
}

// GetAuthProxyWorkloadAfterReconcile finds an AuthProxyWorkload resource named key, waits for its
// "UpToDate" condition to be "True", and the returns it. Fails after 30 seconds
// if the containers does not match.
func (p *TestCaseParams) GetAuthProxyWorkloadAfterReconcile(key types.NamespacedName) (*v1alpha1.AuthProxyWorkload, error) {
	p.T.Helper()
	createdPodmod := &v1alpha1.AuthProxyWorkload{}
	// We'll need to retry getting this newly created resource, given that creation may not immediately happen.
	err := RetryUntilSuccess(p.T, 6, 5*time.Second, func() error {
		err := p.Client.Get(p.Ctx, key, createdPodmod)
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
	return createdPodmod, nil
}

// CreateBusyboxDeployment creates a simple busybox deployment, using the
// key as its namespace and name. It also sets the label "app"= appLabel.
func (p *TestCaseParams) CreateBusyboxDeployment(name types.NamespacedName, appLabel string) (*appsv1.Deployment, error) {
	p.T.Helper()

	d := BuildDeployment(name, appLabel)

	err := p.Client.Create(p.Ctx, d)
	if err != nil {
		return nil, err
	}

	cd := &appsv1.Deployment{}
	err = RetryUntilSuccess(p.T, 5, 1*time.Second, func() error {
		return p.Client.Get(p.Ctx, types.NamespacedName{
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
func (p *TestCaseParams) GetAuthProxyWorkload(key types.NamespacedName) (*v1alpha1.AuthProxyWorkload, error) {
	p.T.Helper()
	createdPodmod := &v1alpha1.AuthProxyWorkload{}
	// We'll need to retry getting this newly created resource, given that creation may not immediately happen.
	err := RetryUntilSuccess(p.T, 6, 5*time.Second, func() error {
		err := p.Client.Get(p.Ctx, key, createdPodmod)
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

// ListPods lists all the pods in a particular deployment.
func ListPods(ctx context.Context, c client.Client, ns string, selector *metav1.LabelSelector) (*corev1.PodList, error) {

	podList := &corev1.PodList{}
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("unable to make pod selector for deployment %v", err)
	}

	err = c.List(ctx, podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		return nil, fmt.Errorf("unable to list pods for deployment %v", err)
	}

	return podList, nil
}

// ExpectPodContainerCount finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func (p *TestCaseParams) ExpectPodContainerCount(podSelector *metav1.LabelSelector, count int, allOrAny string) error {

	p.T.Helper()

	var (
		countBadPods int
		countPods    int
	)

	err := RetryUntilSuccess(p.T, 6, 10*time.Second, func() error {
		countBadPods = 0
		pods, err := ListPods(p.Ctx, p.Client, p.Namespace, podSelector)
		if err != nil {
			return err
		}
		countPods = len(pods.Items)
		if len(pods.Items) == 0 {
			return fmt.Errorf("got 0 pods, want at least 1 pod")
		}
		for _, pod := range pods.Items {
			got := len(pod.Spec.Containers)
			if got != count {
				countBadPods++
				p.T.Logf("got %d containers, want %d containers on pod %v: ", got, count, pod.Name)
			}
		}
		switch {
		case allOrAny == "all" && countBadPods > 0:
			return fmt.Errorf("got the wrong number of containers on %d of %d pods", countBadPods, len(pods.Items))
		case allOrAny == "any" && countBadPods == countPods:
			return fmt.Errorf("got the wrong number of containers on %d of %d pods", countBadPods, len(pods.Items))
		}
		return nil
	})

	if err != nil {
		p.T.Errorf("want %v containers, got the wrong number of containers on %d of %d pods", count, countBadPods, countPods)
		return err
	}

	return nil
}

// ExpectContainerCount finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func (p *TestCaseParams) ExpectContainerCount(key types.NamespacedName, count int) error {

	p.T.Helper()

	var (
		got        int
		deployment = &appsv1.Deployment{}
	)
	err := RetryUntilSuccess(p.T, 6, 5*time.Second, func() error {
		err := p.Client.Get(p.Ctx, key, deployment)
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
		p.T.Errorf("want %v containers, got %v number of containers did not resolve after waiting for reconcile", count, got)
		return err
	}

	p.T.Logf("Container len is now %v", got)
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
func (p *TestCaseParams) CreateAuthProxyWorkload(key types.NamespacedName, appLabel string, connectionString string, kind string) error {
	p.T.Helper()
	proxy := BuildAuthProxyWorkload(key, connectionString)
	proxy.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind: kind,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": appLabel},
		},
	}
	proxy.Spec.AuthProxyContainer = &v1alpha1.AuthProxyContainerSpec{Image: p.ProxyImageURL}
	err := p.Client.Create(p.Ctx, proxy)
	if err != nil {
		p.T.Errorf("Unable to create entity %v", err)
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
