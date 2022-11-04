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

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWorkloadControllers Watch changes for Istio resources managed by the operator
func SetupWorkloadControllers(mgr ctrl.Manager, u *workload.Updater) error {
	mgr.GetWebhookServer().Register("/mutate-workloads", &webhook.Admission{
		Handler: &WorkloadAdmissionWebhook{
			Client:  mgr.GetClient(),
			updater: u,
		}})

	return nil
}

// WorkloadAdmissionWebhook implementation of a controller-runtime webhook for all
// supported workload types: Deployment, DaemonSet, StatefulSet, Pod, CronJob, Job
type WorkloadAdmissionWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
	updater *workload.Updater
}

// InjectDecoder Dependency injection required by KubeBuilder controller runtime.
func (a *WorkloadAdmissionWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

// Handle is the MutatingWebhookController implemnentation which will update
// the proxy sidecars on all workloads to match the AuthProxyWorkload config.
func (a *WorkloadAdmissionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	l := logf.FromContext(ctx)
	l.Info("/mutate-workload request received",
		"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)

	wl, err := a.makeWorkload(req)
	if err != nil {
		l.Info("/mutate-workload request can't be processed",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	var (
		updated           bool
		instList          = &cloudsqlapi.AuthProxyWorkloadList{}
		matchingInstances []*cloudsqlapi.AuthProxyWorkload
		wlConfigErr       error
	)
	err = a.Client.List(ctx, instList) // list all the AuthProxyWorkloads
	if err != nil {
		l.Error(err, "Unable to list CloudSqlClient resources in webhook",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("unable to list CloudSqlClient resources"))
	}

	l.Info("Workload before modification", "len(containers)", len(wl.PodSpec().Containers))

	switch {
	case wl.Object().GetObjectKind().GroupVersionKind().Kind == "Pod":
		owners, err := a.listOwners(ctx, wl.Object())
		if err != nil {
			return admission.Errored(http.StatusInternalServerError,
				fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", err))
		}
		updated, matchingInstances, wlConfigErr = a.updater.ReconcilePod(instList, wl, owners)
	default:
		updated, matchingInstances, wlConfigErr = a.updater.ReconcileWorkload(instList, wl)
	}

	if wlConfigErr != nil {
		l.Error(wlConfigErr, "Unable to reconcile workload result in webhook: "+wlConfigErr.Error(),
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", wlConfigErr))
	}

	if updated {
		l.Info(fmt.Sprintf("Workload operation %s on kind %s named %s/%s required an update",
			req.Operation, req.Kind, req.Namespace, req.Name))
		for _, inst := range matchingInstances {
			l.Info(fmt.Sprintf("inst %v %v/%v updated at instance resource version %v",
				wl.Object().GetObjectKind().GroupVersionKind().String(),
				wl.Object().GetNamespace(), wl.Object().GetName(),
				inst.GetResourceVersion()))
		}
	}

	result := wl.Object()
	marshaledRes, err := json.Marshal(result)
	if err != nil {
		l.Error(err, "Unable to marshal workload result in webhook",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("unable to marshal workload result"))
	}

	if updated {
		l.Info("/mutate-workload request completed. The workload was updated.",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name,
			"len(containers)", len(wl.PodSpec().Containers))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledRes)
}

// makeWorkload creates a Workload from a request.
func (a *WorkloadAdmissionWebhook) makeWorkload(req admission.Request) (workload.Workload, error) {
	wl, err := workload.WorkloadForKind(req.Kind.Kind)
	if err != nil {
		return nil, err
	}

	err = a.decoder.Decode(req, wl.Object())
	if err != nil {
		return nil, err
	}

	return wl, nil
}

// listOwners returns the list of this object's owners and its extended owners
func (a *WorkloadAdmissionWebhook) listOwners(ctx context.Context, object client.Object) ([]workload.Workload, error) {
	l := logf.FromContext(ctx)
	var owners []workload.Workload

	for _, r := range object.GetOwnerReferences() {
		key := client.ObjectKey{Namespace: object.GetNamespace(), Name: r.Name}
		var owner client.Object

		owl, err := workload.WorkloadForKind(r.Kind)
		switch {
		case err != nil:
			owner = &v1.PartialObjectMetadata{
				TypeMeta: v1.TypeMeta{Kind: r.Kind, APIVersion: r.APIVersion},
			}
		default:
			owner = owl.Object()
			owners = append(owners, owl)
		}

		err = a.Client.Get(ctx, key, owner)
		if err != nil {
			l.Info("could not get owner ",
				"owner", r.String(),
				"err", err)
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
