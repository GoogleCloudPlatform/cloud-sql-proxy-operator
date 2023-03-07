// Copyright 2023 Google LLC
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
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestPodWebhookWithDeploymentOwners(t *testing.T) {
	_, scheme, err := clientBuilder()
	if err != nil {
		t.Fatal(err)
	}

	// Proxy workload
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	addFinalizers(p)
	addSelectorWorkload(p, "Deployment", "app", "webapp")

	// Deployment that matches the proxy
	dMatch := testhelpers.BuildDeployment(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "webapp")
	dMatch.ObjectMeta.Labels = map[string]string{"app": "webapp"}

	// Deployment that does not match the proxy
	dNoMatch := testhelpers.BuildDeployment(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "webapp")
	dNoMatch.ObjectMeta.Labels = map[string]string{"app": "other"}

	// Deployment matches the proxy and is owned by another resource
	// called CustomApp
	dWithOwner := testhelpers.BuildDeployment(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "webapp")
	dWithOwner.ObjectMeta.Labels = map[string]string{"app": "webapp"}
	deploymentOwner := &v1.PartialObjectMetadata{
		TypeMeta:   v1.TypeMeta{Kind: "CustomApp", APIVersion: "v1"},
		ObjectMeta: v1.ObjectMeta{Name: "custom-app", Namespace: "default"},
	}
	err = controllerutil.SetOwnerReference(deploymentOwner, dWithOwner, scheme)
	if err != nil {
		t.Fatal(err)
	}

	data := []struct {
		name       string
		p          *cloudsqlapi.AuthProxyWorkload
		d          *appsv1.Deployment
		wantUpdate bool
	}{
		{
			name:       "Deployment Pod with matching Workload",
			p:          p,
			d:          dMatch,
			wantUpdate: true,
		},
		{
			name:       "Deployment Pod with no match",
			p:          p,
			d:          dNoMatch,
			wantUpdate: false,
		},
		{
			name:       "Deployment Pod with unknown owner",
			p:          p,
			d:          dWithOwner,
			wantUpdate: true,
		},
	}
	for _, tc := range data {
		t.Run(tc.name, func(t *testing.T) {
			cb, scheme, err := clientBuilder()
			if err != nil {
				t.Fatal(err)
			}

			rs, hash, err := testhelpers.BuildDeploymentReplicaSet(tc.d, scheme)
			if err != nil {
				t.Fatal(err)
			}
			pods, err := testhelpers.BuildDeploymentReplicaSetPods(tc.d, rs, hash, scheme)
			if err != nil {
				t.Fatal(err)
			}

			c := cb.WithObjects(p).WithObjects(rs).WithObjects(tc.d).Build()
			wh, ctx, err := podWebhookController(c)
			if err != nil {
				t.Fatal(err)
			}

			pod, errRes := wh.handleCreatePodRequest(ctx, *pods[0])

			if errRes != nil {
				t.Fatal("got error, want no error")
			}
			if tc.wantUpdate && pod == nil {
				t.Fatal("got nil, want not nil workload indicating pod updates")
			}
			if !tc.wantUpdate && pod != nil {
				t.Fatal("got non-nil workload, want nil indicating no pod updates")
			}

			if err != nil {
				t.Fatal(err)
			}

		})
	}

}

func podWebhookController(cb client.Client) (*PodAdmissionWebhook, context.Context, error) {
	ctx := log.IntoContext(context.Background(), logger)
	d, err := admission.NewDecoder(cb.Scheme())
	if err != nil {
		return nil, nil, err
	}
	r := &PodAdmissionWebhook{
		Client:  cb,
		decoder: d,
		updater: workload.NewUpdater("cloud-sql-proxy-operator/dev"),
	}

	return r, ctx, nil
}
