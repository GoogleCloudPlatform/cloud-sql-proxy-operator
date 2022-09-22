// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// listWorkloads produces a list of Workload's that match the WorkloadSelectorSpec
// in the specified namespace.
func (r *AuthProxyWorkloadReconciler) listWorkloads(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, csqlWorkloadNs string) ([]internal.Workload, error) {
	ns := csqlWorkloadNs
	if workloadSelector.Namespace != "" {
		ns = workloadSelector.Namespace
	}

	if workloadSelector.Name != "" {
		return r.loadByName(ctx, workloadSelector, ns)
	}

	return r.loadByLabelSelector(ctx, workloadSelector, ns)
}

// loadByName loads a single workload by name.
func (r *AuthProxyWorkloadReconciler) loadByName(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	var wl internal.Workload

	key := client.ObjectKey{Namespace: ns, Name: workloadSelector.Name}

	_, gk := schema.ParseKindArg(workloadSelector.Kind)
	switch gk.Kind {
	case "Deployment":
		wl = &internal.DeploymentWorkload{Deployment: &appsv1.Deployment{}}
	case "Pod":
		wl = &internal.PodWorkload{Pod: &corev1.Pod{}}
	case "StatefulSet":
		wl = &internal.StatefulSetWorkload{StatefulSet: &appsv1.StatefulSet{}}
	case "Job":
		wl = &internal.JobWorkload{Job: &batchv1.Job{}}
	case "CronJob":
		wl = &internal.CronJobWorkload{CronJob: &batchv1.CronJob{}}
	case "DaemonSet":
		wl = &internal.DaemonSetWorkload{DaemonSet: &appsv1.DaemonSet{}}
	default:
		return nil, fmt.Errorf("unknown kind for pod workloadSelector: %s", workloadSelector.Kind)
	}
	err := r.Get(ctx, key, wl.Object())
	if err != nil {
		return nil, fmt.Errorf("unable to load resource by name %s/%s:  %v", key.Namespace, key.Name, err)
	}

	return []internal.Workload{wl}, nil
}

// loadByLabelSelector loads workloads matching a label selector
func (r *AuthProxyWorkloadReconciler) loadByLabelSelector(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}
	_, gk := schema.ParseKindArg(workloadSelector.Kind)
	switch gk.Kind {
	case "Deployment":
		return r.listDeployments(ctx, sel, ns)
	case "Pod":
		return r.listPods(ctx, sel, ns)
	case "StatefulSet":
		return r.listStatefulSets(ctx, sel, ns)
	case "Job":
		return r.listJobs(ctx, sel, ns)
	case "CronJob":
		return r.listCronJobs(ctx, sel, ns)
	case "DaemonSet":
		return r.listDaemonSets(ctx, sel, ns)
	default:
		return nil, fmt.Errorf("unknown kind for pod workloadSelector: %s", workloadSelector.Kind)
	}
}

func (r *AuthProxyWorkloadReconciler) listPods(ctx context.Context, sel labels.Selector, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := corev1.PodList{}

	err := r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.PodWorkload{Pod: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d pods for %v", len(pods), sel))
	return pods, err
}

// listDeployments is used by listWorkloads() to list all deployments related to a workloadSelector.
func (r *AuthProxyWorkloadReconciler) listDeployments(ctx context.Context, sel labels.Selector, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := appsv1.DeploymentList{}

	err := r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.DeploymentWorkload{Deployment: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), sel))
	return pods, err
}

// listDeployments is used by listWorkloads() to list all deployments related to a workloadSelector.
func (r *AuthProxyWorkloadReconciler) listStatefulSets(ctx context.Context, sel labels.Selector, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := appsv1.StatefulSetList{}

	err := r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.StatefulSetWorkload{StatefulSet: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), sel))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listJobs(ctx context.Context, sel labels.Selector, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := batchv1.JobList{}

	err := r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.JobWorkload{Job: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), sel))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listCronJobs(ctx context.Context, sel labels.Selector, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := batchv1.CronJobList{}

	err := r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.CronJobWorkload{CronJob: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), sel))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listDaemonSets(ctx context.Context, sel labels.Selector, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := appsv1.DaemonSetList{}

	err := r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.DaemonSetWorkload{DaemonSet: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), sel))
	return pods, err
}
