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
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
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
	wl := &workload.PodWorkload{
		Pod: &corev1.Pod{},
	}
	err := a.decoder.Decode(req, wl.Object())
	if err != nil {
		l.Info("/mutate-pod request can't be processed",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	var (
		instList    = &cloudsqlapi.AuthProxyWorkloadList{}
		proxies     []*cloudsqlapi.AuthProxyWorkload
		wlConfigErr error
	)

	// List all the AuthProxyWorkloads in the same namespace.
	// To avoid privilege escalation, the operator requires that the AuthProxyWorkload
	// may only affect pods in the same namespace.
	err = a.Client.List(ctx, instList, client.InNamespace(wl.Object().GetNamespace()))
	if err != nil {
		l.Error(err, "Unable to list CloudSqlClient resources in webhook",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("unable to list CloudSqlClient resources"))
	}

	// List the owners of this pod.
	owners, err := a.listOwners(ctx, wl.Object())
	if err != nil {
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", err))
	}

	// Find matching AuthProxyWorkloads for this pod
	proxies = a.updater.FindMatchingAuthProxyWorkloads(instList, wl, owners)
	if len(proxies) == 0 {
		return admission.PatchResponseFromRaw(req.Object.Raw, req.Object.Raw)
	}

	// Configure the pod, adding containers for each of the proxies
	wlConfigErr = a.updater.ConfigureWorkload(wl, proxies)

	if wlConfigErr != nil {
		l.Error(wlConfigErr, "Unable to reconcile workload result in webhook: "+wlConfigErr.Error(),
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", wlConfigErr))
	}

	// Log some information about the pod update
	l.Info(fmt.Sprintf("Workload operation %s on kind %s named %s/%s required an update",
		req.Operation, req.Kind, req.Namespace, req.Name))
	for _, inst := range proxies {
		l.Info(fmt.Sprintf("inst %v %v/%v updated at instance resource version %v",
			wl.Object().GetObjectKind().GroupVersionKind().String(),
			wl.Object().GetNamespace(), wl.Object().GetName(),
			inst.GetResourceVersion()))
	}

	// Marshal the updated Pod and prepare to send a response
	result := wl.Object()
	marshaledRes, err := json.Marshal(result)
	if err != nil {
		l.Error(err, "Unable to marshal workload result in webhook",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("unable to marshal workload result"))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledRes)
}

// listOwners returns the list of this object's owners and its extended owners.
// Warning: this is a recursive function
func (a *PodAdmissionWebhook) listOwners(ctx context.Context, object client.Object) ([]workload.Workload, error) {
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

		err = a.Client.Get(ctx, key, owner)
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
		ownerOwners, err := a.listOwners(ctx, owner)
		if err != nil {
			return nil, err
		}

		owners = append(owners, ownerOwners...)
	}
	return owners, nil
}
