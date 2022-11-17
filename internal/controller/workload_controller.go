// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// PodAdmissionWebhook implementation of a controller-runtime webhook for all
// supported workload types: Deployment, ReplicaSet,DaemonSet, StatefulSet, Pod, CronJob, Job
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

// Handle is the MutatingWebhookController implementation which will update
// the PodTemplateSpec annotations on workloads to include the proxies
// that should be added.
func (a *WorkloadAdmissionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	l := logf.FromContext(ctx)
	wl, err := workload.WorkloadForKind(req.Kind.Kind)
	if err != nil {
		l.Info("/mutate-workload request can't be processed",
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	err = a.decoder.Decode(req, wl.Object())
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

	// Reconfigure the PodSpec
	updated, matchingInstances, wlConfigErr = a.updater.ReconcileWorkload(instList, wl)

	if wlConfigErr != nil {
		l.Error(wlConfigErr, "Unable to reconcile workload result in webhook: "+wlConfigErr.Error(),
			"kind", req.Kind.Kind, "ns", req.Namespace, "name", req.Name)
		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", wlConfigErr))
	}

	// If the pod was updated, log some information
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
