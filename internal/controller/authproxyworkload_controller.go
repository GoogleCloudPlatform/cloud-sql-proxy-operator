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

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	"github.com/go-logr/logr"
)

const finalizerName = cloudsqlapi.AnnotationPrefix + "/AuthProxyWorkload-finalizer"

var (
	requeueNow       = ctrl.Result{Requeue: true}
	requeueWithDelay = ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}
)

type recentlyDeletedCache struct {
	lock   sync.RWMutex
	values map[types.NamespacedName]bool
}

func (c *recentlyDeletedCache) set(k types.NamespacedName, deleted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.values == nil {
		c.values = map[types.NamespacedName]bool{}
	}
	c.values[k] = deleted
}

func (c *recentlyDeletedCache) get(k types.NamespacedName) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	deleted, ok := c.values[k]
	if !ok {
		return false
	}
	return deleted
}

// AuthProxyWorkloadReconciler reconciles a AuthProxyWorkload object
type AuthProxyWorkloadReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	recentlyDeleted *recentlyDeletedCache
	updater         *workload.Updater
}

// NewAuthProxyWorkloadManager constructs an AuthProxyWorkloadReconciler
func NewAuthProxyWorkloadReconciler(mgr ctrl.Manager, u *workload.Updater) (*AuthProxyWorkloadReconciler, error) {
	r := &AuthProxyWorkloadReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		recentlyDeleted: &recentlyDeletedCache{},
		updater:         u,
	}
	err := r.SetupWithManager(mgr)
	return r, err
}

// SetupWithManager adds this AuthProxyWorkload controller to the controller-runtime
// manager.
func (r *AuthProxyWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := true
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudsqlapi.AuthProxyWorkload{}).
		WithOptions(controller.Options{SkipNameValidation: &b}).
		Complete(r)
}

//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets;replicasets,verbs=update;patch
//+kubebuilder:rbac:groups=apps,resources=*,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=*,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=*,verbs=get;list;watch

//+kubebuilder:rbac:groups=cloudsql.cloud.google.com,resources=authproxyworkloads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cloudsql.cloud.google.com,resources=authproxyworkloads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cloudsql.cloud.google.com,resources=authproxyworkloads/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile updates the state of the cluster so that AuthProxyWorkload instances
// have their configuration reflected correctly on workload PodSpec configuration.
// This reconcile loop runs when an AuthProxyWorkload is added, modified or deleted.
// It updates annotations on matching workloads indicating those workload that
// need to be updated.
//
// As this controller's Reconcile() function patches the annotations on workloads,
// the PodAdmissionWebhook.Handle() method is called by k8s api, which is
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
// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *AuthProxyWorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	var err error

	resource := &cloudsqlapi.AuthProxyWorkload{}

	l.Info("Reconcile loop started AuthProxyWorkload", "name", req.NamespacedName)
	if err = r.Client.Get(ctx, req.NamespacedName, resource); err != nil {
		// The resource can't be loaded.
		// If it was recently deleted, then ignore the error and don't requeue.
		if r.recentlyDeleted.get(req.NamespacedName) {
			return ctrl.Result{}, nil
		}

		// otherwise, report the error and requeue. This is likely caused by a delay
		// in reaching consistency in the eventually-consistent kubernetes API.
		l.Error(err, "unable to fetch resource")
		return requeueWithDelay, err
	}

	// If this was deleted, doDelete()
	// DeletionTimestamp metadata field is set by k8s when a resource
	// has been deleted but the finalizers are still present. We check that this
	// value is not zero To determine when a resource is deleted and waiting for
	// completion of finalizers.
	if !resource.ObjectMeta.DeletionTimestamp.IsZero() {
		l.Info("Reconcile delete for AuthProxyWorkload",
			"name", resource.GetName(),
			"namespace", resource.GetNamespace(),
			"gen", resource.GetGeneration())
		r.recentlyDeleted.set(req.NamespacedName, true)
		// the object has been deleted
		return r.doDelete(ctx, resource)
	}

	l.Info("Reconcile add/update for AuthProxyWorkload",
		"name", resource.GetName(),
		"namespace", resource.GetNamespace(),
		"gen", resource.GetGeneration())
	r.recentlyDeleted.set(req.NamespacedName, false)
	return r.doCreateUpdate(ctx, l, resource)
}

// doDelete removes our finalizer and updates the related workloads
// when the reconcile loop receives an AuthProxyWorkload that was deleted.
func (r *AuthProxyWorkloadReconciler) doDelete(ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload) (ctrl.Result, error) {

	// Mark all related workloads as needing to be updated
	allWorkloads, err := r.updateWorkloadStatus(ctx, resource)
	if err != nil {
		return requeueNow, err
	}

	_, err = r.updateWorkloadAnnotations(ctx, resource, allWorkloads)
	if err != nil {
		return requeueNow, err
	}

	// Remove the finalizer so that the object can be fully deleted
	if controllerutil.ContainsFinalizer(resource, finalizerName) {
		controllerutil.RemoveFinalizer(resource, finalizerName)
		err = r.Client.Update(ctx, resource)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// doCreateUpdate reconciles an AuthProxyWorkload resource that has been created
// or updated, making sure that related workloads get updated.
//
// This is implemented as a state machine. The current state is determined using
// - the absence or presence of this controller's finalizer
// - the success or error when retrieving workloads related to this resource
// - the number of workloads needing updates
// - the condition `UpToDate` status and reason
//
// States:
// |  state  | finalizer| fetch err | len(wl) | outOfDateCount | Name                                  |
// |---------|----------|-----------|---------|----------------|---------------------------------------|
// | 0       | *        | *         | *       |                | start                                 |
// | 1.1     | absent   | *         | *       |                | needs finalizer                       |
// | 1.2     | present  | error     | *       |                | can't list workloads                  |
// | 2.1     | present  | nil       | == 0    |                | no workloads to reconcile             |
// | 3.1     | present  | nil       | > 0     | > 0 , err      | workload update needed, and failed    |
// | 3.2     | present  | nil       | > 0     | > 0            | workload update needed, and succeeded |
// | 3.3     | present  | nil       | > 0     | == 0           | workloads reconciled                  |
//
//		start ----x
//		          |---> 1.1 --> (requeue, goto start)
//		          |---> 1.2 --> (requeue, goto start)
//		          |---> 2.1 --> (end)
//		          |
//	            |---> 3.1 ---> (requeue, goto start)
//	            |---> 3.2 ---> (requeue, goto start)
//	            |---> 3.3 ---> (end)
func (r *AuthProxyWorkloadReconciler) doCreateUpdate(ctx context.Context, l logr.Logger, resource *cloudsqlapi.AuthProxyWorkload) (ctrl.Result, error) {
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
	allWorkloads, err := r.updateWorkloadStatus(ctx, resource)
	if err != nil {
		// State 1.2 - unable to read workloads, abort and try again after a delay.
		return requeueWithDelay, err
	}

	// State 2: If workload reconcile has not yet started, then start it.

	// State 2.1: When there are no workloads, then mark this as "UpToDate" true,
	// do not requeue.
	if len(allWorkloads) == 0 {
		return r.reconcileResult(ctx, l, resource, orig, cloudsqlapi.ReasonNoWorkloadsFound, "No workload updates needed", true)
	}

	// State 3.*: Workloads already exist. Some may need to be updated to roll out
	// changes.
	outOfDateCount, err := r.updateWorkloadAnnotations(ctx, resource, allWorkloads)
	if err != nil {
		return requeueNow, err
	}

	// State 3.2 Successfully updated all workload PodTemplateSpec annotations, requeue
	if outOfDateCount > 0 {
		message := fmt.Sprintf("Reconciled %d matching workloads. %d workloads need updates", len(allWorkloads), outOfDateCount)
		return r.reconcileResult(ctx, l, resource, orig, cloudsqlapi.ReasonWorkloadNeedsUpdate, message, false)
	}

	// State 3.3 Workload PodTemplateSpec annotations are all up to date
	message := fmt.Sprintf("Reconciled %d matching workloads complete", len(allWorkloads))
	return r.reconcileResult(ctx, l, resource, orig, cloudsqlapi.ReasonFinishedReconcile, message, true)
}

// needsAnnotationUpdate returns true when the workload was annotated with
// a different generation of the resource.
func (r *AuthProxyWorkloadReconciler) needsAnnotationUpdate(wl workload.Workload, resource *cloudsqlapi.AuthProxyWorkload) bool {
	// This workload is not mutable. Ignore it.
	if _, ok := wl.(workload.WithMutablePodTemplate); !ok {
		return false
	}

	if isRolloutStrategyNone(resource) {
		return false
	}

	k, v := r.updater.PodAnnotation(resource)
	// Check if the correct annotation exists
	an := wl.PodTemplateAnnotations()
	if an != nil && an[k] == v {
		return false
	}

	return true
}

// updateAnnotation applies an annotation to the workload for the resource.
func (r *AuthProxyWorkloadReconciler) updateAnnotation(wl workload.Workload, resource *cloudsqlapi.AuthProxyWorkload) {
	mpt, ok := wl.(workload.WithMutablePodTemplate)

	// This workload is not mutable. Ignore it.
	if !ok {
		return
	}

	// The user has set "None" as the rollout strategy. Ignore it.
	if isRolloutStrategyNone(resource) {
		return
	}

	k, v := r.updater.PodAnnotation(resource)

	// add the annotation if needed...
	an := wl.PodTemplateAnnotations()
	if an == nil {
		an = make(map[string]string)
	}

	an[k] = v
	mpt.SetPodTemplateAnnotations(an)
}

// isRolloutStrategyNone returns true when user has set "None" as the rollout strategy.
func isRolloutStrategyNone(resource *cloudsqlapi.AuthProxyWorkload) bool {
	return resource.Spec.AuthProxyContainer != nil &&
		resource.Spec.AuthProxyContainer.RolloutStrategy == cloudsqlapi.NoneStrategy
}

// workloadsReconciled  State 3.1: If workloads are all up to date, mark the condition
// "UpToDate" true and do not requeue.
func (r *AuthProxyWorkloadReconciler) reconcileResult(ctx context.Context, l logr.Logger, resource, orig *cloudsqlapi.AuthProxyWorkload, reason, message string, upToDate bool) (ctrl.Result, error) {

	status := metav1.ConditionFalse
	result := requeueNow
	if upToDate {
		status = metav1.ConditionTrue
		result = ctrl.Result{}
	}

	// Workload updates are complete, update the status
	resource.Status.Conditions = replaceCondition(resource.Status.Conditions, &metav1.Condition{
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

	err := r.Client.Update(ctx, resource)
	if err != nil {
		l.Info("Error adding finalizer. Will requeue for reconcile.", "err", err)
		return requeueNow, err
	}

	l.Info("Added finalizer. Will requeue quickly for reconcile", "err", err)
	return requeueNow, err
}

// patchAuthProxyWorkloadStatus uses the PATCH method to incrementally update
// the AuthProxyWorkload.Status field.
func (r *AuthProxyWorkloadReconciler) patchAuthProxyWorkloadStatus(
	ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload, orig *cloudsqlapi.AuthProxyWorkload) error {
	err := r.Client.Status().Patch(ctx, resource, client.MergeFrom(orig))
	if err != nil {
		return err
	}
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: resource.GetNamespace(),
		Name:      resource.GetName(),
	}, orig)
	return err
}

// updateWorkloadStatus lists all workloads related to a cloudsql instance and
// updates the needs update annotations using internal.UpdateWorkloadAnnotation.
// Once the workload is saved, the workload admission mutate webhook will
// apply the correct containers to this instance.
func (r *AuthProxyWorkloadReconciler) updateWorkloadStatus(ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload) (matching []workload.Workload, retErr error) {

	matching, err := r.listWorkloads(ctx, resource.Spec.Workload, resource.GetNamespace())
	if err != nil {
		return nil, err
	}

	for _, wl := range matching {
		// update the status condition for a workload
		s := newStatus(wl)
		s.Conditions = replaceCondition(s.Conditions, &metav1.Condition{
			Type:               cloudsqlapi.ConditionWorkloadUpToDate,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: resource.GetGeneration(),
			Reason:             cloudsqlapi.ReasonUpToDate,
			Message:            "No update needed for this workload",
		})
		resource.Status.WorkloadStatus = replaceStatus(resource.Status.WorkloadStatus, s)
	}

	return matching, nil
}

// replaceStatus replace a status with the same name, namespace, kind, and version,
// or appends updatedStatus to statuses
func replaceStatus(statuses []*cloudsqlapi.WorkloadStatus, updatedStatus *cloudsqlapi.WorkloadStatus) []*cloudsqlapi.WorkloadStatus {

	var updated bool
	for i := range statuses {
		s := statuses[i]
		if s.Name == updatedStatus.Name &&
			s.Namespace == updatedStatus.Namespace &&
			s.Kind == updatedStatus.Kind &&
			s.Version == updatedStatus.Version {
			statuses[i] = updatedStatus
			updated = true
		}
	}
	if !updated {
		statuses = append(statuses, updatedStatus)
	}
	return statuses
}

func findCondition(conds []*metav1.Condition, name string) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == name {
			return conds[i]
		}
	}
	return nil
}

// replaceCondition replace a status with the same name, namespace, kind, and version,
// or appends updatedStatus to statuses
func replaceCondition(conds []*metav1.Condition, newC *metav1.Condition) []*metav1.Condition {
	for i := range conds {
		c := conds[i]
		if c.Type != newC.Type {
			continue
		}

		if conds[i].Status == newC.Status && !conds[i].LastTransitionTime.IsZero() {
			newC.LastTransitionTime = conds[i].LastTransitionTime
		} else {
			newC.LastTransitionTime = metav1.NewTime(time.Now())
		}
		conds[i] = newC
		return conds
	}

	newC.LastTransitionTime = metav1.NewTime(time.Now())
	conds = append(conds, newC)
	return conds
}

// newStatus creates a WorkloadStatus from a workload with identifying
// fields filled in.
func newStatus(wl workload.Workload) *cloudsqlapi.WorkloadStatus {
	return &cloudsqlapi.WorkloadStatus{
		Kind:      wl.Object().GetObjectKind().GroupVersionKind().Kind,
		Version:   wl.Object().GetObjectKind().GroupVersionKind().GroupVersion().Identifier(),
		Namespace: wl.Object().GetNamespace(),
		Name:      wl.Object().GetName(),
	}
}

// listWorkloads produces a list of Workload's that match the WorkloadSelectorSpec
// in the specified namespace.
func (r *AuthProxyWorkloadReconciler) listWorkloads(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]workload.Workload, error) {

	if workloadSelector.Name != "" {
		return r.loadByName(ctx, workloadSelector, ns)
	}

	return r.loadByLabelSelector(ctx, workloadSelector, ns)
}

// loadByName loads a single workload by name.
func (r *AuthProxyWorkloadReconciler) loadByName(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]workload.Workload, error) {
	var wl workload.Workload

	key := client.ObjectKey{Namespace: ns, Name: workloadSelector.Name}

	wl, err := workload.WorkloadForKind(workloadSelector.Kind)
	if err != nil {
		return nil, fmt.Errorf("unable to load by name %s/%s:  %v", key.Namespace, key.Name, err)
	}

	err = r.Client.Get(ctx, key, wl.Object())
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // empty list when no named workload is found. It is not an error.
		}
		return nil, fmt.Errorf("unable to load resource by name %s/%s:  %v", key.Namespace, key.Name, err)
	}

	return []workload.Workload{wl}, nil
}

// loadByLabelSelector loads workloads matching a label selector
func (r *AuthProxyWorkloadReconciler) loadByLabelSelector(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]workload.Workload, error) {
	l := log.FromContext(ctx)

	sel, err := workloadSelector.LabelsSelector()

	if err != nil {
		return nil, err
	}
	_, gk := schema.ParseKindArg(workloadSelector.Kind)
	wl, err := workload.WorkloadListForKind(gk.Kind)
	if err != nil {
		return nil, err
	}
	err = r.Client.List(ctx, wl.List(), client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}
	return wl.Workloads(), nil

}

func (r *AuthProxyWorkloadReconciler) updateWorkloadAnnotations(ctx context.Context, resource *cloudsqlapi.AuthProxyWorkload, workloads []workload.Workload) (int, error) {
	var outOfDate int
	for _, wl := range workloads {
		if r.needsAnnotationUpdate(wl, resource) {
			outOfDate++

			_, err := controllerutil.CreateOrPatch(ctx, r.Client, wl.Object(), func() error {
				r.updateAnnotation(wl, resource)
				return nil
			})

			// Failed to update one of the workloads PodTemplateSpec annotations.
			if err != nil {
				return 0, fmt.Errorf("reconciled %d matching workloads. Error removing proxy from workload %v: %v", len(workloads), wl.Object().GetName(), err)
			}
		}
	}

	return outOfDate, nil

}
