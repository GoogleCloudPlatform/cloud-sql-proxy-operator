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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const BusyboxDeployYaml = `apiVersion: apps/appsv1
kind: Deployment
metadata:
  name: busybox-deployment-
  labels:
    app: busyboxon
spec:
  d: 2
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

const BusyboxStatefulSetYaml = `apiVersion: apps/appsv1
kind: StatefulSet
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

const BusyboxDaemonSetYaml = `apiVersion: apps/appsv1
kind: ReplicaSet
metadata:
  name: busybox-deployment-
  labels:
    app: busyboxon
spec:
  serviceName: busybox
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

// Run 10 jobs that each take 30 seconds to complete, one at a time
const BusyboxJob = `apiVersion: batch/v1
kind: Job
metadata:
  name: busybox-deployment-
  labels:
    app: busyboxon
spec:
  template:
    metadata:
      labels:
        app: busyboxon
    spec:
      restartPolicy: "Never"
      containers:
      - name: busybox
        image: busybox
        imagePullPolicy: IfNotPresent
        command: ['sh', '-c', 'echo Container 1 is Running ; sleep 30']
  backoffLimit: 2
  completions: 10
  parallelism: 1
`
const BusyboxCronJob = `apiVersion: batch/v1
kind: CronJob
metadata:
  name: busybox-deployment-
  labels:
    app: busyboxon
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            app: busyboxon
        spec:
          restartPolicy: "Never"
          containers:
          - name: busybox
            image: busybox
            imagePullPolicy: IfNotPresent
            command: ['sh', '-c', 'echo Container 1 is Running ; sleep 30']
      backoffLimit: 2
`

func CreateWorkload(ctx context.Context, tctx *TestCaseParams,
	name types.NamespacedName, appLabel string, yaml string, o client.Object) error {
	tctx.T.Helper()

	err := yaml2.Unmarshal([]byte(yaml), &o)
	if err != nil {
		return err
	}
	o.SetName(name.Name)
	o.SetNamespace(name.Namespace)
	o.SetLabels(map[string]string{"app": appLabel})

	err = tctx.Client.Create(ctx, o)
	if err != nil {
		return err
	}

	err = RetryUntilSuccess(tctx.T, 5, 1*time.Second, func() error {
		return tctx.Client.Get(ctx, types.NamespacedName{
			Namespace: name.Namespace,
			Name:      name.Name,
		}, o)
	})
	if err != nil {
		return err
	}
	return nil
}

// GetAuthProxyWorkloadAfterReconcile finds an AuthProxyWorkload resource named key, waits for its
// "UpToDate" condition to be "True", and the returns it. Fails after 30 seconds
// if the containers does not match.
func GetAuthProxyWorkloadAfterReconcile(ctx context.Context, tctx *TestCaseParams,
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

// ExpectContainerCount finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func ExpectContainerCount(tp *TestCaseParams, ns string, podSelector *metav1.LabelSelector, count int, allOrAny string) error {

	tp.T.Helper()

	var (
		countBadPods int
		countPods    int
	)

	err := RetryUntilSuccess(tp.T, 6, 10*time.Second, func() error {
		countBadPods = 0
		pods, err := ListPods(tp.Ctx, tp.Client, ns, podSelector)
		if err != nil {
			return err
		}
		countPods = len(pods.Items)
		if len(pods.Items) == 0 {
			return fmt.Errorf("got 0 pods, want at least 1 pod")
		}
		for _, p := range pods.Items {
			got := len(p.Spec.Containers)
			if got != count {
				countBadPods++
				tp.T.Logf("got %d containers, want %d containers on pod %v: ", got, count, p.Name)
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
		tp.T.Errorf("want %v containers, got the wrong number of containers on %d of %d pods", count, countBadPods, countPods)
		return err
	}

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
func CreateAuthProxyWorkload(ctx context.Context, tctx *TestCaseParams, key types.NamespacedName, appLabel string, connectionString string, kind string) error {
	tctx.T.Helper()
	p := BuildAuthProxyWorkload(key, connectionString)
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind: kind,
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
