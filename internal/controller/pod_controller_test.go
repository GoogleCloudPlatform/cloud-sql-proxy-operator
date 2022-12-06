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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestPodWebhook(t *testing.T) {

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")

	cb, err := clientBuilder()
	if err != nil {
		t.Fatal("Can't create client", err)
		return
	}
	c := cb.Build()
	_, _, err = podWebhook(c)
	if err != nil {
		t.Errorf("error making webhook endpoint, %v", err)
		return
	}
	t.Log(p.Spec) //TODO implement an actual test

}

func podWebhook(c client.Client) (*PodAdmissionWebhook, context.Context, error) {
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		return nil, nil, err
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		return nil, nil, err
	}
	err = appsv1.AddToScheme(scheme)
	if err != nil {
		return nil, nil, err
	}
	d, err := admission.NewDecoder(scheme)
	if err != nil {
		return nil, nil, err
	}

	ctx := log.IntoContext(context.Background(), logger)
	r := &PodAdmissionWebhook{
		Client:  c,
		updater: workload.NewUpdater(),
		decoder: d,
	}
	return r, ctx, nil
}
