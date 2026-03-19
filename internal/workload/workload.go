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

package workload

import (
	"fmt"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Workload is a standard interface to access the pod definition for the
// 7 major kinds of interfaces: Deployment, Pod, StatefulSet, ReplicaSet,
// DaemonSet, Job, and Cronjob.
// These methods are used by the ModifierStore to update the contents of the
// workload's pod template (or the pod itself) so that it will contain
// necessary configuration and other details before it starts, or if the
// configuration changes.
type Workload interface {
	PodSpec() corev1.PodSpec
	PodTemplateAnnotations() map[string]string
	Object() client.Object
}

// WithMutablePodTemplate interface applies only to workload types where the pod
// template can be changed.
type WithMutablePodTemplate interface {
	SetPodSpec(spec corev1.PodSpec)
	SetPodTemplateAnnotations(map[string]string)
}

// WorkloadList is a standard way to access the lists of the
// 7 major kinds of interfaces: DeploymentList, DaemonSetList, PodList,
// ReplicaSetList, StatefulSetList, JobList, and CronJobList.
type WorkloadList interface {
	// List returns a pointer to the ObjectList ready to be passed to client.List()
	List() client.ObjectList
	// Workloads transforms the contents of the List into a slice of Workloads
	Workloads() []Workload
}

// realWorkloadList is a generic implementation of WorkloadList that enables
// easy extension for new workload list types.
type realWorkloadList[L client.ObjectList, T client.Object] struct {
	objectList    L
	itemsAccessor func(L) []T
	wlCreator     func(T) Workload
}

func (l *realWorkloadList[L, T]) List() client.ObjectList {
	return l.objectList
}

func (l *realWorkloadList[L, T]) Workloads() []Workload {
	items := l.itemsAccessor(l.objectList)
	wls := make([]Workload, len(items))
	for i := range items {
		wls[i] = l.wlCreator(items[i])
	}
	return wls
}

// ptrSlice takes a slice and returns slice with the pointer to each element.
func ptrSlice[T any](l []T) []*T {
	p := make([]*T, len(l))
	for i := 0; i < len(l); i++ {
		p[i] = &l[i]
	}
	return p
}

// WorkloadListForKind returns a new WorkloadList initialized for a particular
// kubernetes Kind.
func WorkloadListForKind(kind string) (WorkloadList, error) {
	switch kind {
	case "Deployment":
		return &realWorkloadList[*appsv1.DeploymentList, *appsv1.Deployment]{
			objectList:    &appsv1.DeploymentList{},
			itemsAccessor: func(list *appsv1.DeploymentList) []*appsv1.Deployment { return ptrSlice[appsv1.Deployment](list.Items) },
			wlCreator:     func(v *appsv1.Deployment) Workload { return &DeploymentWorkload{Deployment: v} },
		}, nil
	case "Pod":
		return &realWorkloadList[*corev1.PodList, *corev1.Pod]{
			objectList:    &corev1.PodList{},
			itemsAccessor: func(list *corev1.PodList) []*corev1.Pod { return ptrSlice[corev1.Pod](list.Items) },
			wlCreator:     func(v *corev1.Pod) Workload { return &PodWorkload{Pod: v} },
		}, nil
	case "StatefulSet":
		return &realWorkloadList[*appsv1.StatefulSetList, *appsv1.StatefulSet]{
			objectList: &appsv1.StatefulSetList{},
			itemsAccessor: func(list *appsv1.StatefulSetList) []*appsv1.StatefulSet {
				return ptrSlice[appsv1.StatefulSet](list.Items)
			},
			wlCreator: func(v *appsv1.StatefulSet) Workload { return &StatefulSetWorkload{StatefulSet: v} },
		}, nil
	case "ReplicaSet":
		return &realWorkloadList[*appsv1.ReplicaSetList, *appsv1.ReplicaSet]{
			objectList: &appsv1.ReplicaSetList{},
			itemsAccessor: func(list *appsv1.ReplicaSetList) []*appsv1.ReplicaSet {
				return ptrSlice[appsv1.ReplicaSet](list.Items)
			},
			wlCreator: func(v *appsv1.ReplicaSet) Workload { return &ReplicaSetWorkload{ReplicaSet: v} },
		}, nil
	case "Job":
		return &realWorkloadList[*batchv1.JobList, *batchv1.Job]{
			objectList:    &batchv1.JobList{},
			itemsAccessor: func(list *batchv1.JobList) []*batchv1.Job { return ptrSlice[batchv1.Job](list.Items) },
			wlCreator:     func(v *batchv1.Job) Workload { return &JobWorkload{Job: v} },
		}, nil
	case "CronJob":
		return &realWorkloadList[*batchv1.CronJobList, *batchv1.CronJob]{
			objectList:    &batchv1.CronJobList{},
			itemsAccessor: func(list *batchv1.CronJobList) []*batchv1.CronJob { return ptrSlice[batchv1.CronJob](list.Items) },
			wlCreator:     func(v *batchv1.CronJob) Workload { return &CronJobWorkload{CronJob: v} },
		}, nil
	case "DaemonSet":
		return &realWorkloadList[*appsv1.DaemonSetList, *appsv1.DaemonSet]{
			objectList:    &appsv1.DaemonSetList{},
			itemsAccessor: func(list *appsv1.DaemonSetList) []*appsv1.DaemonSet { return ptrSlice[appsv1.DaemonSet](list.Items) },
			wlCreator:     func(v *appsv1.DaemonSet) Workload { return &DaemonSetWorkload{DaemonSet: v} },
		}, nil
	default:
		return nil, fmt.Errorf("unknown kind for pod workloadSelector: %s", kind)
	}
}

// WorkloadForKind returns a workload for a particular Kind
func WorkloadForKind(kind string) (Workload, error) {
	_, gk := schema.ParseKindArg(kind)
	switch gk.Kind {
	case "Deployment":
		return &DeploymentWorkload{Deployment: &appsv1.Deployment{}}, nil
	case "Pod":
		return &PodWorkload{Pod: &corev1.Pod{}}, nil
	case "StatefulSet":
		return &StatefulSetWorkload{StatefulSet: &appsv1.StatefulSet{}}, nil
	case "Job":
		return &JobWorkload{Job: &batchv1.Job{}}, nil
	case "CronJob":
		return &CronJobWorkload{CronJob: &batchv1.CronJob{}}, nil
	case "DaemonSet":
		return &DaemonSetWorkload{DaemonSet: &appsv1.DaemonSet{}}, nil
	case "ReplicaSet":
		return &ReplicaSetWorkload{ReplicaSet: &appsv1.ReplicaSet{}}, nil
	default:
		return nil, fmt.Errorf("unknown kind %s", kind)
	}
}

// workloadMatches tests if a workload matches a modifier based on its name, kind, and selectors.
func workloadMatches(wl client.Object, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) bool {
	kind := wl.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		switch wl.(type) {
		case *appsv1.Deployment:
			kind = "Deployment"
		case *corev1.Pod:
			kind = "Pod"
		case *appsv1.ReplicaSet:
			kind = "ReplicaSet"
		case *appsv1.StatefulSet:
			kind = "StatefulSet"
		case *appsv1.DaemonSet:
			kind = "DaemonSet"
		case *batchv1.Job:
			kind = "Job"
		case *batchv1.CronJob:
			kind = "CronJob"
		}
	}

	if workloadSelector.Kind != "" && kind != workloadSelector.Kind {
		return false
	}
	if workloadSelector.Name != "" && wl.GetName() != workloadSelector.Name {
		return false
	}
	if ns != "" && wl.GetNamespace() != ns {
		return false
	}

	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return false
	}
	if !sel.Empty() && !sel.Matches(labels.Set(wl.GetLabels())) {
		return false
	}

	return true
}

type DeploymentWorkload struct {
	Deployment *appsv1.Deployment
}

func (d *DeploymentWorkload) PodSpec() corev1.PodSpec {
	return d.Deployment.Spec.Template.Spec
}
func (d *DeploymentWorkload) PodTemplateAnnotations() map[string]string {
	return d.Deployment.Spec.Template.Annotations
}
func (d *DeploymentWorkload) SetPodTemplateAnnotations(v map[string]string) {
	d.Deployment.Spec.Template.Annotations = v
}

func (d *DeploymentWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.Deployment.Spec.Template.Spec = spec
}

func (d *DeploymentWorkload) Object() client.Object {
	return d.Deployment
}

type StatefulSetWorkload struct {
	StatefulSet *appsv1.StatefulSet
}

func (d *StatefulSetWorkload) PodSpec() corev1.PodSpec {
	return d.StatefulSet.Spec.Template.Spec
}
func (d *StatefulSetWorkload) PodTemplateAnnotations() map[string]string {
	return d.StatefulSet.Spec.Template.Annotations
}
func (d *StatefulSetWorkload) SetPodTemplateAnnotations(v map[string]string) {
	d.StatefulSet.Spec.Template.Annotations = v
}

func (d *StatefulSetWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.StatefulSet.Spec.Template.Spec = spec
}

func (d *StatefulSetWorkload) Object() client.Object {
	return d.StatefulSet
}

type PodWorkload struct {
	Pod *corev1.Pod
}

func (d *PodWorkload) PodSpec() corev1.PodSpec {
	return d.Pod.Spec
}
func (d *PodWorkload) PodTemplateAnnotations() map[string]string {
	return d.Pod.Annotations
}
func (d *PodWorkload) SetPodTemplateAnnotations(v map[string]string) {
	d.Pod.Annotations = v
}

func (d *PodWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.Pod.Spec = spec
}

func (d *PodWorkload) Object() client.Object {
	return d.Pod
}

type JobWorkload struct {
	Job *batchv1.Job
}

func (d *JobWorkload) PodSpec() corev1.PodSpec {
	return d.Job.Spec.Template.Spec
}
func (d *JobWorkload) PodTemplateAnnotations() map[string]string {
	return d.Job.Spec.Template.Annotations
}
func (d *JobWorkload) Object() client.Object {
	return d.Job
}

type CronJobWorkload struct {
	CronJob *batchv1.CronJob
}

func (d *CronJobWorkload) PodSpec() corev1.PodSpec {
	return d.CronJob.Spec.JobTemplate.Spec.Template.Spec
}
func (d *CronJobWorkload) PodTemplateAnnotations() map[string]string {
	return d.CronJob.Spec.JobTemplate.Spec.Template.Annotations
}
func (d *CronJobWorkload) Object() client.Object {
	return d.CronJob
}

type DaemonSetWorkload struct {
	DaemonSet *appsv1.DaemonSet
}

func (d *DaemonSetWorkload) PodSpec() corev1.PodSpec {
	return d.DaemonSet.Spec.Template.Spec
}
func (d *DaemonSetWorkload) PodTemplateAnnotations() map[string]string {
	return d.DaemonSet.Spec.Template.Annotations
}
func (d *DaemonSetWorkload) SetPodTemplateAnnotations(v map[string]string) {
	d.DaemonSet.Spec.Template.Annotations = v
}

func (d *DaemonSetWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.DaemonSet.Spec.Template.Spec = spec
}

func (d *DaemonSetWorkload) Object() client.Object {
	return d.DaemonSet
}

type ReplicaSetWorkload struct {
	ReplicaSet *appsv1.ReplicaSet
}

func (d *ReplicaSetWorkload) PodSpec() corev1.PodSpec {
	return d.ReplicaSet.Spec.Template.Spec
}
func (d *ReplicaSetWorkload) PodTemplateAnnotations() map[string]string {
	return d.ReplicaSet.Spec.Template.Annotations
}
func (d *ReplicaSetWorkload) SetPodTemplateAnnotations(v map[string]string) {
	d.ReplicaSet.Spec.Template.Annotations = v
}
func (d *ReplicaSetWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.ReplicaSet.Spec.Template.Spec = spec
}

func (d *ReplicaSetWorkload) Object() client.Object {
	return d.ReplicaSet
}
