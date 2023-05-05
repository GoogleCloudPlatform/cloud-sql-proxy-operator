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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	l.Info("received mutate pod request: ", "Kind", req.RequestKind, "Operation", req.Operation, "Name", req.Name, "Namespace", req.Namespace, "AdmissionRequest", req.AdmissionRequest)

	updatedPod, err := a.handleCreatePodRequest(ctx, p)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if updatedPod == nil {
		l.Info("no changes", "Kind", req.RequestKind, "Operation", req.Operation, "Name", req.Name, "Namespace", req.Namespace)
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
	l.Info("updated pod", "Kind", req.RequestKind, "Operation", req.Operation, "Name", req.Name, "Namespace", req.Namespace)

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

	// Configure the pod, adding containers for each of the proxies
	wlConfigErr := a.updater.ConfigureWorkload(wl, proxies)

	if wlConfigErr != nil {
		l.Error(wlConfigErr, "Unable to reconcile workload result in webhook: "+wlConfigErr.Error(),
			"kind", wl.Pod.Kind, "ns", wl.Pod.Namespace, "name", wl.Pod.Name)
		return nil, fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", wlConfigErr)
	}

	return wl.Pod, nil // updated
}

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

		// recursively call for the owners of the owner, and append those.
		// So that we reach Pod --> ReplicaSet --> Deployment
		ownerOwners, err := listOwners(ctx, c, owner)
		if err != nil {
			return nil, err
		}

		owners = append(owners, ownerOwners...)
	}
	return owners, nil
}

type PodEventHandler struct {
	ctx context.Context
	u   *workload.Updater
	l   logr.Logger
	mgr manager.Manager
}

// NeedLeaderElection implements manager.LeaderElectionRunnable so that
// the PodEventHandler only runs on the leader, not on other redundant
// pods.
func (h *PodEventHandler) NeedLeaderElection() bool {
	return true
}

// Start implements manager.Runnable which will start the informer receiving
// pod change events on the operator's leader instance.
func (h *PodEventHandler) Start(ctx context.Context) error {
	h.ctx = ctx

	i, err := h.mgr.GetCache().GetInformerForKind(ctx, schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	})
	if err != nil {
		return fmt.Errorf("Unable to get pod informer, %v", err)
	}

	_, err = i.AddEventHandler(h)
	if err != nil {
		return fmt.Errorf("Unable to register pod event handler, %v", err)
	}
	return nil

}

// OnAdd is called by the informer when a Pod is added.
func (h *PodEventHandler) OnAdd(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}
	h.handlePodChanged(pod)
}

// OnUpdate is called by the informer when a Pod is updated.
func (h *PodEventHandler) OnUpdate(_, newObj interface{}) {
	newPod, ok := newObj.(*corev1.Pod)
	if !ok {
		return
	}
	h.handlePodChanged(newPod)
}

// OnUpdate is called by the informer when a Pod is deleted.
func (h *PodEventHandler) OnDelete(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}
	h.l.Info("Update pod: %v", "pod", pod)
}

// handlePodChanged Deletes pods that meet the following criteria:
// 1. The pod is in Error or CrashLoopBackoff state.
// 2. The pod should have one or more proxy sidecar containers.
// 3. The pod is missing one or more proxy sidecar containers.
func (h *PodEventHandler) handlePodChanged(pod *corev1.Pod) {
	wl := &workload.PodWorkload{Pod: pod}
	c := h.mgr.GetClient()

	proxies, err := findMatchingProxies(h.ctx, c, h.u, wl)
	if err != nil {
		h.l.Error(err, "Unable to find proxies when pod changed")
		return
	}

	// There are no proxies, nothing more to do.
	if len(proxies) == 0 {
		return
	}

	// Configure the pod, adding containers for each of the proxies
	wlConfigErr := h.u.CheckWorkloadContainers(wl, proxies)

	// If the pod has a config error, and the pod is not deleted, delete it.
	if wlConfigErr != nil && pod.ObjectMeta.DeletionTimestamp.IsZero() {
		h.l.Info("Pod configured incorrectly. Deleting.", "Namespace", pod.Namespace, "Name", pod.Name, "Status", pod.Status)
		err = c.Delete(h.ctx, pod)
		if err != nil && !apierrors.IsNotFound(err) {
			h.l.Error(err, "Unable to delete pod.", "Namespace", pod.Namespace, "Name", pod.Name)
		}
	}

}
