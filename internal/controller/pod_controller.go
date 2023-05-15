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
	"net/http"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
)

// PodAdmissionWebhook implementation of a controller-runtime webhook for all
// supported workload types: Deployment, ReplicaSet, StatefulSet, Pod, CronJob, Job
type PodAdmissionWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
	updater *workload.Updater
}

// InjectDecoder Dependency injection required by KubeBuilder controller runtime.
func (a *PodAdmissionWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

// Handle is the MutatingWebhookController implemnentation which will update
// the proxy sidecars on all workloads to match the AuthProxyWorkload config.
func (a *PodAdmissionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	l := logf.FromContext(ctx)
	p := corev1.Pod{}
	err := a.decoder.Decode(req, &p)
	if err != nil {
		l.Info("/mutate-pod request can't be processed",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	updatedPod, err := a.handleCreatePodRequest(ctx, p)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if updatedPod == nil {
		return admission.Allowed("no changes to pod")
	}

	// Marshal the updated Pod and prepare to send a response
	marshaledRes, err := json.Marshal(updatedPod)
	if err != nil {
		l.Error(err, "Unable to marshal workload result in webhook",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("unable to marshal workload result"))
	}
	l.Info("updated proxy on pod", "Operation", req.Operation, "Namespace", req.Namespace, "Name", req.Name)

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledRes)
}

// handleCreatePodRequest Finds relevant AuthProxyWorkload resources and updates the pod
// with matching resources, returning a non-nil pod when the pod was updated.
func (a *PodAdmissionWebhook) handleCreatePodRequest(ctx context.Context, p corev1.Pod) (*corev1.Pod, error) {
	l := logf.FromContext(ctx)
	wl := &workload.PodWorkload{Pod: &p}

	proxies, err := findMatchingProxies(ctx, a.Client, a.updater, wl)
	if err != nil {
		return nil, err
	}

	if len(proxies) == 0 {
		return nil, nil
	}

	// Configure the pod, adding containers for each of the proxies
	wlConfigErr := a.updater.ConfigureWorkload(wl, proxies)

	if wlConfigErr != nil {
		l.Error(wlConfigErr, "Unable to reconcile workload result in webhook: "+wlConfigErr.Error(),
			"kind", wl.Pod.Kind, "ns", wl.Pod.Namespace, "name", wl.Pod.Name)
		return nil, fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", wlConfigErr)
	}

	return wl.Pod, nil // updated pod
}

// findMatchingProxies lists all AuthProxyWorkloads that are related to this pod
// or its owners.
func findMatchingProxies(ctx context.Context, c client.Client, u *workload.Updater, wl *workload.PodWorkload) ([]*cloudsqlapi.AuthProxyWorkload, error) {
	var (
		instList = &cloudsqlapi.AuthProxyWorkloadList{}
		proxies  []*cloudsqlapi.AuthProxyWorkload
		l        = logf.FromContext(ctx)
	)

	// List all the AuthProxyWorkloads in the same namespace.
	// To avoid privilege escalation, the operator requires that the AuthProxyWorkload
	// may only affect pods in the same namespace.
	err := c.List(ctx, instList, client.InNamespace(wl.Object().GetNamespace()))
	if err != nil {
		l.Error(err, "Unable to list CloudSqlClient resources in webhook",
			"kind", wl.Pod.Kind, "ns", wl.Pod.Namespace, "name", wl.Pod.Name)
		return nil, fmt.Errorf("unable to list AuthProxyWorkloads, %v", err)
	}

	// List the owners of this pod.
	owners, err := listOwners(ctx, c, wl.Object())
	if err != nil {
		return nil, fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", err)
	}

	// Find matching AuthProxyWorkloads for this pod
	proxies = u.FindMatchingAuthProxyWorkloads(instList, wl, owners)
	if len(proxies) == 0 {
		return nil, nil // no change
	}

	return proxies, nil

}

// listOwners returns the list of this object's owners and its extended owners.
// Warning: this is a recursive function
func listOwners(ctx context.Context, c client.Client, object client.Object) ([]workload.Workload, error) {
	l := logf.FromContext(ctx)
	var owners []workload.Workload

	for _, r := range object.GetOwnerReferences() {
		key := client.ObjectKey{Namespace: object.GetNamespace(), Name: r.Name}
		var owner client.Object

		wl, err := workload.WorkloadForKind(r.Kind)
		if err != nil {
			// If the operator doesn't recognize the owner's Kind, then ignore
			// that owner.
			continue
		}

		owners = append(owners, wl)
		owner = wl.Object()

		err = c.Get(ctx, key, owner)
		if err != nil {
			switch t := err.(type) {
			case *apierrors.StatusError:
				// Ignore when the owner is not found. Sometimes owners no longer exist.
				if t.ErrStatus.Reason == metav1.StatusReasonNotFound {
					continue
				}
			}

			l.Info("could not get owner ", "owner", r.String(), "err", err)
			return nil, err
		}

		// Recursively call for the owners of the owner, and append those.
		// So that we reach Pod --> ReplicaSet --> Deployment
		ownerOwners, err := listOwners(ctx, c, owner)
		if err != nil {
			return nil, err
		}

		owners = append(owners, ownerOwners...)
	}
	return owners, nil
}

type podDeleteController struct {
	client.Client
	Scheme  *runtime.Scheme
	updater *workload.Updater
}

// newDeletePodController constructs a podDeleteController.
func newPodDeleteController(mgr ctrl.Manager, u *workload.Updater) (*podDeleteController, error) {
	r := &podDeleteController{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		updater: u,
	}
	err := r.setupWithManager(mgr)
	return r, err
}

// setupWithManager adds this AuthProxyWorkload controller to the controller-runtime
// manager.
func (r *podDeleteController) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

func (r *podDeleteController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Read the ReplicaSet
	pod := &corev1.Pod{}
	err := r.Client.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	// Pod is already deleted, ignore this change event.
	if !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	err = r.handlePodChanged(ctx, pod)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// handlePodChanged Deletes pods that meet the following criteria:
// 1. The pod is in Error or CrashLoopBackoff state.
// 2. The pod matches one or more AuthProxyWorkload resources.
// 3. The pod is missing one or more proxy sidecar containers for the resources.
func (r *podDeleteController) handlePodChanged(ctx context.Context, pod *corev1.Pod) error {

	wl := &workload.PodWorkload{Pod: pod}

	// Find all proxies that match this pod
	proxies, err := findMatchingProxies(ctx, r.Client, r.updater, wl)
	if err != nil {
		return fmt.Errorf("unable to find proxies when pod changed, %v", err)
	}

	// There are no proxies, ignore this event.
	if len(proxies) == 0 {
		return nil
	}

	// Check if this pod is in an error or waiting state and is missing
	// proxy containers.
	wlConfigErr := r.updater.CheckWorkloadContainers(wl, proxies)

	// If the pod is misconfigured, attempt to delete it.
	if wlConfigErr != nil {
		l := logf.FromContext(ctx)
		l.Info("Pod configured incorrectly. Deleting.",
			"Namespace", pod.Namespace, "Name", pod.Name, "Status", pod.Status)
		err = r.Client.Delete(ctx, pod)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to delete pod %v/%v, %v", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}
