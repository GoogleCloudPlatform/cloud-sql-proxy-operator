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
	"net/http"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/util/json"
)

// SetupWorkloadControllers Watch changes for Istio resources managed by the operator
func SetupWorkloadControllers(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register("/mutate-workloads", &webhook.Admission{
		Handler: &WorkloadAdmissionWebhook{
			Client: mgr.GetClient(),
		}})

	return nil
}

// WorkloadAdmissionWebhook implementation of a controller-runtime webhook for all
// supported workload types: Deployment, DaemonSet, StatefulSet, Pod, CronJob, Job
type WorkloadAdmissionWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
}

// InjectDecoder Dependency injection required by KubeBuilder controller runtime
func (a *WorkloadAdmissionWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

// Handle is the MutatingWebhookController implementation. It uses the ModifierStore
// configured in main.go to update the
func (a *WorkloadAdmissionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	l := logf.FromContext(ctx)

	wl, response, done := a.makeWorkload(req)
	if done {
		return response
	}

	var updated bool
	var instList cloudsqlapi.AuthProxyWorkloadList
	err := a.Client.List(ctx, &instList, client.InNamespace(wl.GetObject().GetNamespace()))
	if err != nil {
		l.Error(err, "Unable to list CloudSqlClient resources in webhook")
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("unable to list CloudSqlClient resources"))
	}

	updated, matchingInstances, wlConfigErr := internal.ReconcileWorkload(instList, wl)
	if wlConfigErr != nil {
		l.Error(wlConfigErr, "Unable to reconcile workload result in webhook: "+wlConfigErr.Error())
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("there is an AuthProxyWorkloadConfiguration error reconciling this workload %v", wlConfigErr))
	}

	if updated {
		l.Info(fmt.Sprintf("Workload operation %s on kind %s named %s/%s requires update", req.Operation, req.Kind, req.Namespace, req.Name))
		ann := wl.GetObject().GetAnnotations()
		if ann == nil {
			ann = map[string]string{}
		}
		needsUpdateVersion := ann[NeedsUpdateAnnotation]
		for _, inst := range matchingInstances {
			l.Info(fmt.Sprintf("inst %v %v/%v updated at instance resource version %v",
				wl.GetObject().GetObjectKind().GroupVersionKind().String(),
				wl.GetObject().GetNamespace(), wl.GetObject().GetName(),
				inst.GetResourceVersion()))
			if inst.GetResourceVersion() == needsUpdateVersion {
				ann[WasUpdatedAnnotation] = matchingInstances[0].GetResourceVersion()
			}
		}
		wl.GetObject().SetAnnotations(ann)
	}

	result := wl.GetObject()
	marshaledPod, err := json.Marshal(result)
	if err != nil {
		l.Error(err, "Unable to marshal workload result in webhook")
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("unable to marshal workload result"))
	}

	if updated {
		l.Info(fmt.Sprintf("Modified %s %s/%s", req.Kind, req.Namespace, req.Name))
	}

	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)

}

func (a *WorkloadAdmissionWebhook) makeWorkload(req admission.Request) (internal.Workload, admission.Response, bool) {
	var wl internal.Workload
	switch req.Kind.Kind {
	case "Deployment":
		d := appsv1.Deployment{}
		err := a.decoder.Decode(req, &d)
		if err != nil {
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
		wl = &internal.DeploymentWorkload{Deployment: &d}
	case "CronJob":
		d := batchv1.CronJob{}
		err := a.decoder.Decode(req, &d)
		if err != nil {
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
		wl = &internal.CronJobWorkload{CronJob: &d}
	case "Job":
		d := batchv1.Job{}
		err := a.decoder.Decode(req, &d)
		if err != nil {
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
		wl = &internal.JobWorkload{Job: &d}
	case "StatefulSet":
		d := appsv1.StatefulSet{}
		err := a.decoder.Decode(req, &d)
		if err != nil {
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
		wl = &internal.StatefulSetWorkload{StatefulSet: &d}
	case "DaemonSet":
		d := appsv1.DaemonSet{}
		err := a.decoder.Decode(req, &d)
		if err != nil {
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
		wl = &internal.DaemonSetWorkload{DaemonSet: &d}
	case "Pod":
		d := corev1.Pod{}
		err := a.decoder.Decode(req, &d)
		if err != nil {
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
		wl = &internal.PodWorkload{Pod: &d}
	default:
		return nil, admission.Errored(http.StatusInternalServerError, fmt.Errorf("unsupported resource kind %s", req.Kind.Kind)), true
	}
	return wl, admission.Response{}, false
}
