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
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

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

	//TODO implement pod webhook

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
