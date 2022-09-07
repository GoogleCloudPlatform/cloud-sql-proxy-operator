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
	"fmt"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/names"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const DefaultContainerImage = "us-central1-docker.pkg.dev/csql-operator-test/test76e6d646e2caac1c458c/proxy-v2:latest" //TODO get the right name here.
var defaultContainerResources = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		"cpu":    resource.MustParse("1.0"),
		"memory": resource.MustParse("1Gi"),
	},
}

var (
	l = logf.Log.WithName("internal.workload")
)

// ReconcileWorkload finds all AuthProxyWorkload resources matching this workload and then
// updates the workload's containers. This does not save the updated workload.
func ReconcileWorkload(instList cloudsqlapi.AuthProxyWorkloadList, workload Workload) (bool, []*cloudsqlapi.AuthProxyWorkload, *ConfigError) {
	// if a workload has an owner, then ignore it.
	if len(workload.GetObject().GetOwnerReferences()) > 0 {
		return false, []*cloudsqlapi.AuthProxyWorkload{}, nil
	}

	matchingAuthProxyWorkloads := FilterMatchingInstances(instList, workload)
	updated, err := UpdateWorkloadContainers(workload, matchingAuthProxyWorkloads)
	if updated {
		return true, matchingAuthProxyWorkloads, err
	} else {
		return false, []*cloudsqlapi.AuthProxyWorkload{}, nil
	}

}

// FilterMatchingInstances returns a list of AuthProxyWorkload whose selectors match
// the workload.
func FilterMatchingInstances(wlList cloudsqlapi.AuthProxyWorkloadList, workload Workload) []*cloudsqlapi.AuthProxyWorkload {
	matchingAuthProxyWorkloads := make([]*cloudsqlapi.AuthProxyWorkload, 0, len(wlList.Items))
	for i, _ := range wlList.Items {
		csqlWorkload := &wlList.Items[i]
		if WorkloadMatches(workload, csqlWorkload.Spec.Workload, csqlWorkload.Namespace) {
			// need to update workload
			l.Info("Found matching workload", "workload", workload.GetObject().GetNamespace()+"/"+workload.GetObject().GetName(), "wlSelector", csqlWorkload.Spec.Workload, "AuthProxyWorkload", csqlWorkload.Namespace+"/"+csqlWorkload.Name)
			matchingAuthProxyWorkloads = append(matchingAuthProxyWorkloads, csqlWorkload)
		}
	}
	return matchingAuthProxyWorkloads
}

// WorkloadMatches tests if a workload matches a modifier based on its name, kind, and selectors
func WorkloadMatches(wl Workload, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) bool {
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

type WorkloadUpdateStatus struct {
	InstanceGeneration    string
	LastRequstGeneration  string
	RequestGeneration     string
	LastUpdatedGeneration string
	UpdatedGeneration     string
}

// MarkWorkloadNeedsUpdate Updates annotations on the workload indicating that it may need an update.
// returns true if the workload actually needs an update.
func MarkWorkloadNeedsUpdate(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload) (bool, WorkloadUpdateStatus) {
	return updateWorkloadAnnotations(csqlWorkload, workload, false)
}

// MarkWorkloadUpdated Updates annotations on the workload indicating that it
// has been updated, returns true of any modifications were made to the workload.
// for the AuthProxyWorkload.
func MarkWorkloadUpdated(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload) (bool, WorkloadUpdateStatus) {
	return updateWorkloadAnnotations(csqlWorkload, workload, true)
}

// updateWorkloadAnnotations adds annotations to the workload
// to track which generation of a AuthProxyWorkload needs to be applied, and which
// generation has been applied. The AuthProxyWorkload controller is responsible for
// tracking which version should be applied, The workload admission webhook is
// responsible for applying the AuthProxyWorkloads that apply to a workload
// when the workload is created or modified.
func updateWorkloadAnnotations(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload, doingUpdate bool) (bool, WorkloadUpdateStatus) {
	s := WorkloadUpdateStatus{}
	doUpdate := false
	reqName := names.SafePrefixedName("csqlr-", csqlWorkload.Name)
	resultName := names.SafePrefixedName("csqlu-", csqlWorkload.Name)
	s.InstanceGeneration = fmt.Sprintf("%d", csqlWorkload.GetGeneration())

	ann := workload.GetObject().GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	s.LastRequstGeneration = ann[reqName]
	s.LastUpdatedGeneration = ann[resultName]

	if s.LastRequstGeneration != s.InstanceGeneration {
		ann[reqName] = s.InstanceGeneration
		if doingUpdate {
			ann[resultName] = s.InstanceGeneration
		}
		doUpdate = true
	}
	if s.LastUpdatedGeneration != s.InstanceGeneration {
		if doingUpdate {
			ann[resultName] = s.InstanceGeneration
		}
		doUpdate = true
	}
	workload.GetObject().SetAnnotations(ann)
	s.RequestGeneration = ann[reqName]
	s.UpdatedGeneration = ann[resultName]

	return doUpdate, s
}

// Workload interface a standard way to access the pod definition for the
// 5 major kinds of interfaces: Deployment, Pod, StatefulSet, Job, and Cronjob.
// These methods are used by the ModifierStore to update the contents of the
// workload's pod template (or the pod itself) so that it will contain
// necessary configuration and other details before it starts, or if the
// configuration changes.
type Workload interface {
	GetPodSpec() corev1.PodSpec
	SetPodSpec(spec corev1.PodSpec)
	GetPodObjectMeta() metav1.ObjectMeta
	SetPodObjectMeta(meta metav1.ObjectMeta)
	GetObject() client.Object
}

func WorkloadKey(wl Workload) string {
	return wl.GetObject().GetObjectKind().GroupVersionKind().String() + "/" + client.ObjectKeyFromObject(wl.GetObject()).String()
}

// A workload for deployments, returning the pod template spec
type DeploymentWorkload struct {
	Deployment *appsv1.Deployment
}

func (d *DeploymentWorkload) GetPodSpec() corev1.PodSpec {
	return d.Deployment.Spec.Template.Spec
}

func (d *DeploymentWorkload) SetPodSpec(spec corev1.PodSpec) {
	d.Deployment.Spec.Template.Spec = spec
}

func (d *DeploymentWorkload) GetPodObjectMeta() metav1.ObjectMeta {
	return d.Deployment.Spec.Template.ObjectMeta
}

func (d *DeploymentWorkload) SetPodObjectMeta(meta metav1.ObjectMeta) {
	d.Deployment.Spec.Template.ObjectMeta = meta
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

func (d *StatefulSetWorkload) GetPodObjectMeta() metav1.ObjectMeta {
	return d.StatefulSet.Spec.Template.ObjectMeta
}

func (d *StatefulSetWorkload) SetPodObjectMeta(meta metav1.ObjectMeta) {
	d.StatefulSet.Spec.Template.ObjectMeta = meta
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

func (d *PodWorkload) GetPodObjectMeta() metav1.ObjectMeta {
	return d.Pod.ObjectMeta
}

func (d *PodWorkload) SetPodObjectMeta(meta metav1.ObjectMeta) {
	d.Pod.ObjectMeta = meta
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

func (d *JobWorkload) GetPodObjectMeta() metav1.ObjectMeta {
	return d.Job.Spec.Template.ObjectMeta
}

func (d *JobWorkload) SetPodObjectMeta(meta metav1.ObjectMeta) {
	d.Job.Spec.Template.ObjectMeta = meta
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

func (d *CronJobWorkload) GetPodObjectMeta() metav1.ObjectMeta {
	return d.CronJob.Spec.JobTemplate.Spec.Template.ObjectMeta
}

func (d *CronJobWorkload) SetPodObjectMeta(meta metav1.ObjectMeta) {
	d.CronJob.Spec.JobTemplate.Spec.Template.ObjectMeta = meta
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

func (d *DaemonSetWorkload) GetPodObjectMeta() metav1.ObjectMeta {
	return d.DaemonSet.Spec.Template.ObjectMeta
}

func (d *DaemonSetWorkload) SetPodObjectMeta(meta metav1.ObjectMeta) {
	d.DaemonSet.Spec.Template.ObjectMeta = meta
}
func (d *DaemonSetWorkload) GetObject() client.Object {
	return d.DaemonSet
}
