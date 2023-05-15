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

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
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
		updater: workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage),
	}

	return r, ctx, nil
}

func TestPodDeleteController(t *testing.T) {
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

	data := []struct {
		name                 string
		d                    *appsv1.Deployment
		wantNotFound         bool
		setPodError          bool
		setSidecarContainers bool
	}{
		{
			name:         "matching pod with error gets deleted",
			d:            dMatch,
			setPodError:  true,
			wantNotFound: true,
		},
		{
			name:         "matching pod with no error",
			d:            dMatch,
			setPodError:  false,
			wantNotFound: false,
		},
		{
			name:         "no matching workload, pod ok",
			d:            dNoMatch,
			setPodError:  false,
			wantNotFound: false,
		},
		{
			name:         "no matching workload, pod error",
			d:            dNoMatch,
			setPodError:  true,
			wantNotFound: false,
		},
		{
			name:                 "matching workload, pod error, has containers",
			d:                    dNoMatch,
			setPodError:          true,
			setSidecarContainers: true,
			wantNotFound:         false,
		},
		{
			name:                 "matching workload, pod ok, has containers",
			d:                    dNoMatch,
			setPodError:          false,
			setSidecarContainers: true,
			wantNotFound:         false,
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
			if tc.setPodError {
				pods[0].Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:  pods[0].Spec.Containers[0].Name,
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackoff"}},
				}}
			}

			cb = cb.WithObjects(p)
			cb = cb.WithObjects(tc.d)
			cb = cb.WithObjects(rs)

			for _, p := range pods {
				cb = cb.WithObjects(p)
			}
			c := cb.Build()
			h, ctx := podDeleteControllerForTest(c)
			if tc.setSidecarContainers {
				h.updater.ConfigureWorkload(&workload.PodWorkload{Pod: pods[0]}, []*cloudsqlapi.AuthProxyWorkload{p})
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pods[0].Namespace,
					Name:      pods[0].Name,
				},
			}

			h.Reconcile(ctx, req)

			var deletedPod corev1.Pod
			err = c.Get(ctx, client.ObjectKeyFromObject(pods[0]), &deletedPod)

			if !tc.wantNotFound && errors.IsNotFound(err) {
				t.Fatalf("want not found error, got %v", err)
			}
			if tc.wantNotFound && !errors.IsNotFound(err) {
				t.Fatalf("want no error, found error, got %v", err)
			}

		})
	}

}

func podDeleteControllerForTest(c client.Client) (*podDeleteController, context.Context) {
	ctx := log.IntoContext(context.Background(), logger)
	r := &podDeleteController{
		Client:  c,
		Scheme:  c.Scheme(),
		updater: workload.NewUpdater("cloud-sql-proxy-operator/dev", "proxy:1.0"),
	}
	return r, ctx
}
