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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = cloudsqlapi.AnnotationPrefix + "/AuthProxyWorkload-finalizer"

var shortRequeueResult = ctrl.Result{Requeue: true}
var longRequeueResult = ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}

// AuthProxyWorkloadReconciler reconciles a AuthProxyWorkload object
type AuthProxyWorkloadReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	recentlyDeleted map[types.NamespacedName]bool
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthProxyWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudsqlapi.AuthProxyWorkload{}).
		Complete(r)
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

// Reconcile updates the state of the cluster so that AuthProxyWorkload instances
// have their configuration reflected correctly on workload PodSpec configuration.
// This reconcile loop runs when an AuthProxyWorkload is added, modified or deleted.
// It updates annotations on matching workloads indicating those workload that
// need to be updated.
//
// As this controller's Reconcile() function patches the annotations on workloads,
// the WorkloadAdmissionWebhook.Handle() method is called by k8s api, which is
// where the PodSpec is modified to match the AuthProxyWorkload configuration.
//
// This function can only make one update to the AuthProxyWorkload per loop, so it
// is written like a state machine. It will quickly do a single update, often to
// the status, and then return. Sometimes it instructs the controller runtime to quickly
// requeue another call to Reconcile, so that it can further process the
// AuthProxyWorkload. It often takes several calls to Reconcile() to finish the
// reconcilliation of a single change to an AuthProxyWorkload.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *AuthProxyWorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	var err error

	var resource cloudsqlapi.AuthProxyWorkload

	l.Info("Reconcile loop started AuthProxyWorkload", "name", req.NamespacedName)
	if err = r.Get(ctx, req.NamespacedName, &resource); err != nil {
		// The resource can't be loaded.
		// If it was recently deleted, then ignore the error and don't requeue.
		if r.recentlyDeleted[req.NamespacedName] {
			return ctrl.Result{}, nil
		}

		// otherwise, report the error and requeue. This is likely caused by a delay
		// in reaching consistency in the eventually-consistent kubernetes API.
		l.Error(err, "unable to fetch resource")
		return longRequeueResult, err
	}

	// If this was deleted, doDelete()
	if !resource.ObjectMeta.DeletionTimestamp.IsZero() {
		l.Info("Reconcile delete for AuthProxyWorkload",
			"name", resource.GetName(),
			"namespace", resource.GetNamespace(),
			"gen", resource.GetGeneration())
		r.recentlyDeleted[req.NamespacedName] = true
		// the object has been deleted
		return r.doDelete(ctx, &resource, l)
	}

	l.Info("Reconcile add/update for AuthProxyWorkload",
		"name", resource.GetName(),
		"namespace", resource.GetNamespace(),
		"gen", resource.GetGeneration())
	r.recentlyDeleted[req.NamespacedName] = false
	return r.doAddUpdate(ctx, l, &resource)
}

// doDelete when the reconcile loop receives an AuthProxyWorkload that was deleted,
// update all the workloads and remove this proxy from them.
func (r *AuthProxyWorkloadReconciler) doDelete(ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload, l logr.Logger) (ctrl.Result, error) {

	// Mark all related workloads as needing to be updated
	wls, _, err := r.markWorkloadsForUpdate(ctx, l, resource)
	if err != nil {
		return shortRequeueResult, err
	}
	err = r.patchAnnotations(ctx, wls, l)
	if err != nil {
		return shortRequeueResult, err
	}

	// Remove the finalizer so that the object can be fully deleted
	if controllerutil.ContainsFinalizer(resource, finalizerName) {
		controllerutil.RemoveFinalizer(resource, finalizerName)
		err = r.Update(ctx, resource)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// doAddUpdate handles the reconcile loop an AuthProxyWorkload was added or updated.
// this is basically implemented as a state machine using a combiniation of
// the presence of a finalizer and the condition `UpToDate`. to determine
// what state the resource is in, and therefore what next action to take.
//
// reconcile
// loop --> 0 -x
// start        \---> 1.1 --> (requeue)
//
//	\---> 1.2 --> (requeue)
//	 x---x
//	      \---> 2.1 --> (end)
//	       \---> 2.2 --> (requeue)
//	        x---x
//	             \---> 3.1 --> (end)
//	              \---> 3.2 --> (requeue)
func (r *AuthProxyWorkloadReconciler) doAddUpdate(
	ctx context.Context, l logr.Logger, resource *cloudsqlapi.AuthProxyWorkload) (ctrl.Result, error) {
	orig := resource.DeepCopy()
	var err error
	// State 0: The reconcile loop for a single AuthProxyWorkload resource begins
	// when an AuthProxyWorkload resource is created, modified, or deleted in the k8s api
	// or when that AuthProxyWorkload resource is requeued for another reconcile loop.

	if !controllerutil.ContainsFinalizer(resource, finalizerName) {
		// State 1.1: This is a brand new thing that doesn't have a finalizer.
		// Add the finalizer and requeue for another run through the reconcile loop
		return r.applyFinalizer(ctx, l, resource)
	}

	// find all workloads that relate to this AuthProxyWorkload resource
	allWorkloads, needsUpdateWorkloads, err := r.markWorkloadsForUpdate(ctx, l, resource)
	if err != nil {
		// State 1.2 - unable to read workloads, abort and try again after a delay.
		return longRequeueResult, err
	}

	if r.readyToStartWorkloadReconcile(resource) {
		// State 2: If workload reconcile has not yet started, then start it.

		// State 2.1: When there are no workloads, then mark this as "UpToDate" true,
		// do not requeue.
		if len(needsUpdateWorkloads) == 0 {
			return r.noUpdatesNeeded(ctx, l, resource, orig)
		}

		// State 2.2: When there are workloads, then mark this as "UpToDate" false
		// with the reason "StartedReconcile" and requeue after a delay.
		return r.startWorkloadReconcile(ctx, l, resource, orig, needsUpdateWorkloads)
	}

	// State 3: Workload updates are in progress. Check if the workload updates
	// are complete.
	//
	//   State 3.1: If workloads are all up to date, mark the condition
	//   "UpToDate" true and do not requeue.
	//
	//   State 3.2: If workloads are still up to date, mark the condition
	//   "UpToDate" false and requeue for another run after a delay.
	return r.checkReconcileComplete(ctx, l, resource, orig, allWorkloads)
}

// noUpdatesNeeded no updated needed, so patch the AuthProxyWorkload
// status conditions and return.
func (r *AuthProxyWorkloadReconciler) noUpdatesNeeded(
	ctx context.Context, l logr.Logger, resource *cloudsqlapi.AuthProxyWorkload,
	orig *cloudsqlapi.AuthProxyWorkload) (ctrl.Result, error) {

	meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type:               cloudsqlapi.ConditionUpToDate,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: resource.GetGeneration(),
		Reason:             cloudsqlapi.ReasonNoWorkloadsFound,
		Message:            "No workload updates needed",
	})
	err := r.patchAuthProxyWorkloadStatus(ctx, resource, orig)
	l.Info("Reconcile found no workloads to update", "ns", resource.GetNamespace(), "name", resource.GetName())
	return ctrl.Result{}, err
}

// readyToStartWorkloadReconcile true when the reconcile loop began reconciling
// this AuthProxyWorkload's matching resources.
func (r *AuthProxyWorkloadReconciler) readyToStartWorkloadReconcile(
	resource *cloudsqlapi.AuthProxyWorkload) bool {
	s := meta.FindStatusCondition(resource.Status.Conditions, cloudsqlapi.ConditionUpToDate)
	return s == nil || (s.Status == metav1.ConditionFalse && s.Reason != cloudsqlapi.ReasonStartedReconcile)
}

// startWorkloadReconcile to begin updating matching workloads with the configuration
// in this AuthProxyWorkload resource.
func (r *AuthProxyWorkloadReconciler) startWorkloadReconcile(
	ctx context.Context, l logr.Logger, resource *cloudsqlapi.AuthProxyWorkload,
	orig *cloudsqlapi.AuthProxyWorkload, workloads []internal.Workload) (ctrl.Result, error) {

	// Submit the workloads that need to be reconciled to the workload webhook
	err := r.patchAnnotations(ctx, workloads, l)
	if err != nil {
		l.Error(err, "Unable to update workloads with needs update annotations",
			"AuthProxyWorkload", resource.GetNamespace()+"/"+resource.GetName())
		return shortRequeueResult, err
	}

	// Update the status on this AuthProxyWorkload
	meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type:               cloudsqlapi.ConditionUpToDate,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: resource.GetGeneration(),
		Reason:             cloudsqlapi.ReasonStartedReconcile,
		Message:            "New generation found, needs to reconcile workloads",
	})
	err = r.patchAuthProxyWorkloadStatus(ctx, resource, orig)
	if err != nil {
		l.Error(err, "Unable to patch status before beginning workloads",
			"AuthProxyWorkload", resource.GetNamespace()+"/"+resource.GetName())
		return shortRequeueResult, err
	}

	l.Info("Reconcile launched workload updates",
		"AuthProxyWorkload", resource.GetNamespace()+"/"+resource.GetName())
	return shortRequeueResult, nil
}

// checkReconcileComplete checks if reconcile has finished and if not, attempts
// to start the workload reconcile again.
func (r *AuthProxyWorkloadReconciler) checkReconcileComplete(
	ctx context.Context, l logr.Logger, resource *cloudsqlapi.AuthProxyWorkload,
	orig *cloudsqlapi.AuthProxyWorkload, workloads []internal.Workload) (ctrl.Result, error) {

	var foundUnreconciled bool
	for i := 0; i < len(workloads); i++ {
		wl := workloads[i]
		s := internal.WorkloadStatus(resource, wl)
		if s.LastUpdatedGeneration != s.LastRequstGeneration ||
			s.LastUpdatedGeneration != s.InstanceGeneration {
			foundUnreconciled = true
		}
	}

	if foundUnreconciled {
		return r.startWorkloadReconcile(ctx, l, resource, orig, workloads)
	}

	status := metav1.ConditionTrue
	reason := cloudsqlapi.ReasonFinishedReconcile
	message := fmt.Sprintf("Reconciled %d matching workloads complete", len(workloads))
	var result ctrl.Result

	// Workload updates are complete, update the status
	meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type:               cloudsqlapi.ConditionUpToDate,
		Status:             status,
		ObservedGeneration: resource.GetGeneration(),
		Reason:             reason,
		Message:            message,
	})
	err := r.patchAuthProxyWorkloadStatus(ctx, resource, orig)
	if err != nil {
		l.Error(err, "Unable to patch status before beginning workloads", "AuthProxyWorkload", resource.GetNamespace()+"/"+resource.GetName())
		return result, err
	}

	l.Info("Reconcile checked completion of workload updates",
		"ns", resource.GetNamespace(), "name", resource.GetName(),
		"complete", status)
	return result, nil
}

// applyFinalizer adds the finalizer so that the operator is notified when
// this AuthProxyWorkload resource gets deleted. applyFinalizer is called only
// once, when the resource first added.
func (r *AuthProxyWorkloadReconciler) applyFinalizer(
	ctx context.Context, l logr.Logger, resource *cloudsqlapi.AuthProxyWorkload) (ctrl.Result, error) {

	// The AuthProxyWorkload resource needs a finalizer, so add
	// the finalizer, exit the reconcile loop and requeue.
	controllerutil.AddFinalizer(resource, finalizerName)
	err := r.Update(ctx, resource)
	if err != nil {
		l.Info("Error adding finalizer. Will requeue for reconcile.", "err", err)
		return shortRequeueResult, err
	}

	l.Info("Added finalizer. Will requeue quickly for reconcile", "err", err)
	return shortRequeueResult, err
}

// patchAuthProxyWorkloadStatus uses the PATCH method to incrementally update
// the AuthProxyWorkload.Status field.
func (r *AuthProxyWorkloadReconciler) patchAuthProxyWorkloadStatus(
	ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload, orig *cloudsqlapi.AuthProxyWorkload) error {
	err := r.Client.Status().Patch(ctx, resource, client.MergeFrom(orig))
	if err != nil {
		return err
	}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: resource.GetNamespace(),
		Name:      resource.GetName(),
	}, orig)
	return err
}

// markWorkloadsForUpdate lists all workloads related to a cloudsql instance and
// updates the needs update annotations using internal.UpdateWorkloadAnnotation.
// Once the workload is saved, the workload admission mutate webhook will
// apply the correct containers to this instance.
func (r *AuthProxyWorkloadReconciler) markWorkloadsForUpdate(
	ctx context.Context,
	l logr.Logger,
	resource *cloudsqlapi.AuthProxyWorkload,
) (
	matching, outOfDate []internal.Workload,
	retErr error,
) {

	matching, err := r.listWorkloads(ctx, resource.Spec.Workload, resource.GetNamespace())
	if err != nil {
		return nil, nil, err
	}

	// all matching workloads get a new annotation that will be removed
	// when the reconcile loop for outOfDate is completed.
	for _, wl := range matching {
		needsUpdate, status := internal.MarkWorkloadNeedsUpdate(resource, wl)

		if needsUpdate {
			l.Info("Needs update workload ", "name", wl.Object().GetName(),
				"needsUpdate", needsUpdate,
				"status", status)
			s := newStatus(wl)
			meta.SetStatusCondition(&s.Conditions, metav1.Condition{
				Type:               cloudsqlapi.ConditionWorkloadUpToDate,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: resource.GetGeneration(),
				Reason:             cloudsqlapi.ReasonNeedsUpdate,
				Message:            fmt.Sprintf("Workload needs an update from generation %q to %q", status.LastUpdatedGeneration, status.RequestGeneration),
			})
			replaceStatus(&resource.Status.WorkloadStatus, s)
			outOfDate = append(outOfDate, wl)
			continue
		}

		s := newStatus(wl)
		meta.SetStatusCondition(&s.Conditions, metav1.Condition{
			Type:               cloudsqlapi.ConditionWorkloadUpToDate,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: resource.GetGeneration(),
			Reason:             cloudsqlapi.ReasonUpToDate,
			Message:            "No update needed for this workload",
		})
		replaceStatus(&resource.Status.WorkloadStatus, s)

	}

	return matching, outOfDate, nil
}

// replaceStatus replace a status with the same name, namespace, kind, and version,
// or appends updatedStatus to statuses
func replaceStatus(statuses *[]cloudsqlapi.WorkloadStatus, updatedStatus cloudsqlapi.WorkloadStatus) {

	updated := false
	for i := range *statuses {
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

// withNewAnnotations will Return a MutateFn that sets the annotations on a workload
// object
func withNewAnnotations(res client.Object, ann map[string]string) controllerutil.MutateFn {
	return func() error {
		res.SetAnnotations(ann)
		return nil
	}
}

// patchAnnotations Safely patch the workloads with updated annotations
func (r *AuthProxyWorkloadReconciler) patchAnnotations(ctx context.Context, wls []internal.Workload, l logr.Logger) error {
	for _, wl := range wls {
		obj := wl.Object()
		_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, withNewAnnotations(obj, obj.GetAnnotations()))
		if err != nil {
			l.Error(err, "Unable to patch workload", "name", wl.Object().GetName())
			return err
		}
	}
	return nil
}

// newStatus creates a WorkloadStatus from a workload with identifying
// fields filled in.
func newStatus(workload internal.Workload) cloudsqlapi.WorkloadStatus {
	return cloudsqlapi.WorkloadStatus{
		Kind:      workload.Object().GetObjectKind().GroupVersionKind().Kind,
		Version:   workload.Object().GetObjectKind().GroupVersionKind().GroupVersion().Identifier(),
		Namespace: workload.Object().GetNamespace(),
		Name:      workload.Object().GetName(),
	}
}
