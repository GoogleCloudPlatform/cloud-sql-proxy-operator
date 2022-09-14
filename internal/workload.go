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

package internal

import (
	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Workload interface a standard way to access the pod definition for the
// 5 major kinds of interfaces: Deployment, Pod, StatefulSet, Job, and Cronjob.
// These methods are used by the ModifierStore to update the contents of the
// workload's pod template (or the pod itself) so that it will contain
// necessary configuration and other details before it starts, or if the
// configuration changes.
type Workload interface {
	GetPodSpec() corev1.PodSpec
	SetPodSpec(spec corev1.PodSpec)
	GetObject() client.Object
}

// workloadMatches tests if a workload matches a modifier based on its name, kind, and selectors.
func workloadMatches(wl Workload, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) bool {
	if workloadSelector.Kind != "" && wl.GetObject().GetObjectKind().GroupVersionKind().Kind != workloadSelector.Kind {
		return false
	}
	if workloadSelector.Name != "" && wl.GetObject().GetName() != workloadSelector.Name {
		return false
	}
	if ns != "" && wl.GetObject().GetNamespace() != ns {
		return false
	}

	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return false
	}
	if !sel.Empty() && !sel.Matches(labels.Set(wl.GetObject().GetLabels())) {
		return false
	}

	return true
}

type DeploymentWorkload struct {
	Deployment *appsv1.Deployment
}

func (d *DeploymentWorkload) GetPodSpec() corev1.PodSpec {
	return d.Deployment.Spec.Template.Spec
}

func (d *DeploymentWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.Deployment.Spec.Template.Spec = spec
}

func (d *DeploymentWorkload) GetObject() client.Object {
	return d.Deployment
}

type StatefulSetWorkload struct {
	StatefulSet *appsv1.StatefulSet
}

func (d *StatefulSetWorkload) GetPodSpec() corev1.PodSpec {
	return d.StatefulSet.Spec.Template.Spec
}

func (d *StatefulSetWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.StatefulSet.Spec.Template.Spec = spec
}

func (d *StatefulSetWorkload) GetObject() client.Object {
	return d.StatefulSet
}

type PodWorkload struct {
	Pod *corev1.Pod
}

func (d *PodWorkload) GetPodSpec() corev1.PodSpec {
	return d.Pod.Spec
}

func (d *PodWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.Pod.Spec = spec
}

func (d *PodWorkload) GetObject() client.Object {
	return d.Pod
}

type JobWorkload struct {
	Job *batchv1.Job
}

func (d *JobWorkload) GetPodSpec() corev1.PodSpec {
	return d.Job.Spec.Template.Spec
}

func (d *JobWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.Job.Spec.Template.Spec = spec
}

func (d *JobWorkload) GetObject() client.Object {
	return d.Job
}

type CronJobWorkload struct {
	CronJob *batchv1.CronJob
}

func (d *CronJobWorkload) GetPodSpec() corev1.PodSpec {
	return d.CronJob.Spec.JobTemplate.Spec.Template.Spec
}

func (d *CronJobWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.CronJob.Spec.JobTemplate.Spec.Template.Spec = spec
}

func (d *CronJobWorkload) GetObject() client.Object {
	return d.CronJob
}

type DaemonSetWorkload struct {
	DaemonSet *appsv1.DaemonSet
}

func (d *DaemonSetWorkload) GetPodSpec() corev1.PodSpec {
	return d.DaemonSet.Spec.Template.Spec
}

func (d *DaemonSetWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.DaemonSet.Spec.Template.Spec = spec
}

func (d *DaemonSetWorkload) GetObject() client.Object {
	return d.DaemonSet
}
