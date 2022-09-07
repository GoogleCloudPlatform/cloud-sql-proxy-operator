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

package controllers

import (
	"context"
	"fmt"
	"time"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "cloudsql.cloud.google.com/AuthProxyWorkload-finalizer"

const requeueAfter = 30 * time.Second

// AuthProxyWorkloadReconciler reconciles a AuthProxyWorkload object
type AuthProxyWorkloadReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	recentlyDeleted map[types.NamespacedName]bool
}

//+kubebuilder:rbac:groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cloudsql.cloud.google.com,resources=authproxyworkloads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cloudsql.cloud.google.com,resources=authproxyworkloads/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=daemonsets/finalizers,verbs=update

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch,resources=jobs/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=pods/finalizers,verbs=update

//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch,resources=cronjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// This will compare the AuthProxyWorkload object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *AuthProxyWorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	var err error
	var result ctrl.Result

	var resource cloudsqlapi.AuthProxyWorkload

	l.Info("Reconcile loop started AuthProxyWorkload", "name", req.NamespacedName)
	if err = r.Get(ctx, req.NamespacedName, &resource); err != nil {
		if r.recentlyDeleted[req.NamespacedName] {
			return ctrl.Result{}, nil
		} else {
			l.Error(err, "unable to fetch resource")

			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, err
		}
	}

	if resource.ObjectMeta.DeletionTimestamp.IsZero() {
		l.Info("Reconcile add/update for AuthProxyWorkload", "name", resource.GetName(), "namespace", resource.GetNamespace(), "gen", resource.GetGeneration())
		r.recentlyDeleted[req.NamespacedName] = false
		// the object has beeen added or updated...
		result, err = r.doAddUpdate(ctx, req, &resource, l)
	} else {
		l.Info("Reconcile delete for AuthProxyWorkload", "name", resource.GetName(), "namespace", resource.GetNamespace(), "gen", resource.GetGeneration())
		r.recentlyDeleted[req.NamespacedName] = true
		// the object has been deleted
		result, err = r.doDelete(ctx, &resource, l)
	}

	return result, err
}

func (r *AuthProxyWorkloadReconciler) doAddUpdate(ctx context.Context, req ctrl.Request, resource *cloudsqlapi.AuthProxyWorkload, l logr.Logger) (ctrl.Result, error) {
	doUpdate := false
	orig := resource.DeepCopy()

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if !controllerutil.ContainsFinalizer(resource, finalizerName) {
		//controllerutil.AddFinalizer(resource, finalizerName)
		resource.GetObjectMeta().SetFinalizers(append(resource.GetObjectMeta().GetFinalizers(), finalizerName))
		doUpdate = true
	}

	workloads, result, err := r.markWorkloadsForUpdate(ctx, resource, l)
	if len(workloads) > 0 {
		meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
			Type:               cloudsqlapi.ConditionUpToDate,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: resource.GetGeneration(),
			Reason:             "StartingWorkloadReconcile",
			Message:            "New generation found, needs to reconcile workloads",
		})
		err = r.Client.Status().Update(ctx, resource)
		if err != nil {
			l.Info("Error updating condition UpToDate", "err", err)
		}
	}

	r.saveWorkloads(ctx, workloads, l)

	meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type:               cloudsqlapi.ConditionUpToDate,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: resource.GetGeneration(),
		Reason:             "FinishedWorkloadReconcile",
		Message:            fmt.Sprintf("Reconciled %d matching workloads", len(workloads)),
	})
	err = r.Client.Status().Update(ctx, resource)
	if err != nil {
		l.Info("Error updating condition UpToDate", "err", err)
	}

	if doUpdate {
		err := r.Patch(ctx, resource, client.MergeFrom(orig))
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return result, err
}

// markWorkloadsForUpdate lists all workloads related to a cloudsql instance and
// updates the needs update annotations using internal.UpdateWorkloadAnnotation.
// Once the workload is saved, the workload admission mutate webhook will
// apply the correct containers to this instance.
func (r *AuthProxyWorkloadReconciler) markWorkloadsForUpdate(ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload, l logr.Logger) ([]internal.Workload, ctrl.Result, error) {
	var workloads []internal.Workload

	wls, err := r.listWorkloads(ctx, resource.Spec.Workload, resource.GetNamespace())
	if err != nil {
		return nil, ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, err
	}

	// all matching workloads get a new annotation that will be removed
	// when the reconcile loop for workloads is completed.
	for _, wl := range wls {
		needsUpdate, status := internal.MarkWorkloadNeedsUpdate(resource, wl)
		l.Info("Needs update workload ", "name", wl.GetObject().GetName(),
			"needsUpdate", needsUpdate,
			"status", status)

		if needsUpdate {
			s := newStatus(wl)
			meta.SetStatusCondition(&s.Conditions, metav1.Condition{
				Type:               cloudsqlapi.ConditionUpToDate,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: resource.GetGeneration(),
				Reason:             "NeedsUpdate",
				Message:            "Workload needs an update",
			})
			replaceStatus(&resource.Status.WorkloadStatus, s)
			workloads = append(workloads, wl)
		} else {
			s := newStatus(wl)
			meta.SetStatusCondition(&s.Conditions, metav1.Condition{
				Type:               cloudsqlapi.ConditionUpToDate,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: resource.GetGeneration(),
				Reason:             "NoUpdateNeeded",
				Message:            "No update needed for this workload",
			})
			replaceStatus(&resource.Status.WorkloadStatus, s)
		}
	}

	return workloads, ctrl.Result{}, nil
}

func replaceStatus(statuses *[]cloudsqlapi.WorkloadStatus, updatedStatus cloudsqlapi.WorkloadStatus) {

	updated := false
	for i, _ := range *statuses {
		s := (*statuses)[i]
		if s.Name == updatedStatus.Name &&
			s.Namespace == updatedStatus.Namespace &&
			s.Kind == updatedStatus.Kind &&
			s.Version == updatedStatus.Version {
			(*statuses)[i] = updatedStatus
			updated = true
		}
	}
	if !updated {
		*statuses = append(*statuses, updatedStatus)
	}
}

func (r *AuthProxyWorkloadReconciler) doDelete(ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload, l logr.Logger) (ctrl.Result, error) {
	// The object is being deleted
	if controllerutil.ContainsFinalizer(resource, finalizerName) {

		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(resource, finalizerName)
		if err := r.Update(ctx, resource); err != nil {
			return ctrl.Result{}, err
		}

	}
	wls, result, err := r.markWorkloadsForUpdate(ctx, resource, l)
	r.saveWorkloads(ctx, wls, l)
	return result, err

}

func makeWorkloadUpdate(workload internal.Workload, newAnnotations map[string]string) controllerutil.MutateFn {
	return func() error {
		workload.GetObject().SetAnnotations(newAnnotations)
		return nil
	}
}

// saveWorkloads Safely patch the workloads with updated annotations
func (r *AuthProxyWorkloadReconciler) saveWorkloads(ctx context.Context, wls []internal.Workload, l logr.Logger) {
	for _, wl := range wls {
		_, err := controllerutil.CreateOrPatch(ctx, r.Client, wl.GetObject(), makeWorkloadUpdate(wl, wl.GetObject().GetAnnotations()))
		if err != nil {
			l.Error(err, "Unable to patch workload", "name", wl.GetObject().GetName())
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthProxyWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudsqlapi.AuthProxyWorkload{}).
		Complete(r)
}

func (r *AuthProxyWorkloadReconciler) listWorkloads(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, csqlWorkloadNs string) ([]internal.Workload, error) {
	var ns string
	if workloadSelector.Namespace != "" {
		ns = workloadSelector.Namespace
	} else {
		ns = csqlWorkloadNs
	}
	switch workloadSelector.Kind {
	case "Deployment":
		return r.listDeployments(ctx, workloadSelector, ns)
	case "Pod":
		return r.listPods(ctx, workloadSelector, ns)
	case "StatefulSet":
		return r.listStatefulSets(ctx, workloadSelector, ns)
	case "Job":
		return r.listJobs(ctx, workloadSelector, ns)
	case "CronJob":
		return r.listCronJobs(ctx, workloadSelector, ns)
	case "DaemonSet":
		return r.listDaemonSets(ctx, workloadSelector, ns)
	default:
		return nil, fmt.Errorf("Unknown kind for pod workloadSelector: %s", workloadSelector.Kind)
	}
}

func (r *AuthProxyWorkloadReconciler) listPods(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := corev1.PodList{}

	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}

	err = r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "podmod", workloadSelector)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.PodWorkload{Pod: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d pods for %v", len(pods), workloadSelector))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listDeployments(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := appsv1.DeploymentList{}
	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}

	err = r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "podmod", workloadSelector)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.DeploymentWorkload{Deployment: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), workloadSelector))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listStatefulSets(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := appsv1.StatefulSetList{}
	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}

	err = r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "podmod", workloadSelector)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.StatefulSetWorkload{StatefulSet: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), workloadSelector))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listJobs(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := batchv1.JobList{}
	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}

	err = r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "podmod", workloadSelector)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.JobWorkload{Job: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), workloadSelector))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listCronJobs(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := batchv1.CronJobList{}
	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}

	err = r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "podmod", workloadSelector)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.CronJobWorkload{CronJob: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), workloadSelector))
	return pods, err
}

func (r *AuthProxyWorkloadReconciler) listDaemonSets(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := log.FromContext(ctx)
	podList := appsv1.DaemonSetList{}
	sel, err := workloadSelector.LabelsSelector()
	if err != nil {
		return nil, err
	}

	err = r.List(ctx, &podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "podmod", workloadSelector)
		return nil, err
	}

	pods := make([]internal.Workload, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &internal.DaemonSetWorkload{DaemonSet: podList.Items[i].DeepCopy()}
	}
	l.Info(fmt.Sprintf("Listed %d deployments for %v", len(pods), workloadSelector))
	return pods, err
}

func newStatus(workload internal.Workload) cloudsqlapi.WorkloadStatus {
	return cloudsqlapi.WorkloadStatus{
		Kind:      workload.GetObject().GetObjectKind().GroupVersionKind().Kind,
		Version:   workload.GetObject().GetObjectKind().GroupVersionKind().GroupVersion().Identifier(),
		Namespace: workload.GetObject().GetNamespace(),
		Name:      workload.GetObject().GetName(),
	}
}
