// Copyright 2022 Google LLC.
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
	"os"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
)

var logger = zap.New(zap.UseFlagOptions(&zap.Options{
	Development: true,
	TimeEncoder: zapcore.ISO8601TimeEncoder,
}))

func TestMain(m *testing.M) {
	// logger is the test logger used by the testintegration tests and server.
	ctrl.SetLogger(logger)

	result := m.Run()
	os.Exit(result)
}

func TestReconcileState11(t *testing.T) {

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")

	_, _, err := runReconcileTestcase(p, []client.Object{p}, true, "", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestReconcileDeleted(t *testing.T) {

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	addFinalizers(p)
	addPodWorkload(p)

	cb, _, err := clientBuilder()
	if err != nil {
		t.Error(err) // shouldn't ever happen
	}
	c := cb.WithObjects(p).Build()
	r, req, ctx := reconciler(p, c, workload.DefaultProxyImage)

	c.Delete(ctx, p)
	if err != nil {
		t.Error(err)
	}

	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Error(err)
	}
	if res.Requeue {
		t.Errorf("got %v, want %v for requeue", res.Requeue, false)
	}

	err = c.Get(ctx, types.NamespacedName{
		Namespace: p.GetNamespace(),
		Name:      p.GetName(),
	}, p)
	if err != nil {
		if !errors.IsNotFound(err) {
			t.Errorf("wants not found error, got %v", err)
		}
	} else {
		t.Error("wants not found error, got no error")
	}

}

func TestReconcileState21ByName(t *testing.T) {
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	addFinalizers(p)
	addPodWorkload(p)

	_, _, err := runReconcileTestcase(p, []client.Object{p}, false, metav1.ConditionTrue, cloudsqlapi.ReasonNoWorkloadsFound)
	if err != nil {
		t.Fatal(err)
	}

}
func TestReconcileState21BySelector(t *testing.T) {
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	addFinalizers(p)
	addSelectorWorkload(p, "Pod", "app", "things")

	_, _, err := runReconcileTestcase(p, []client.Object{p}, false, metav1.ConditionTrue, cloudsqlapi.ReasonNoWorkloadsFound)
	if err != nil {
		t.Fatal(err)
	}

}

func TestReconcileState32(t *testing.T) {
	const (
		wantRequeue = true
		wantStatus  = metav1.ConditionFalse
		wantReason  = cloudsqlapi.ReasonWorkloadNeedsUpdate
		labelK      = "app"
		labelV      = "things"
	)
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Generation = 2
	addFinalizers(p)
	addSelectorWorkload(p, "Deployment", labelK, labelV)

	// mimic a pod that was updated by the webhook
	reqName := cloudsqlapi.AnnotationPrefix + "/" + p.Name
	pod := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{labelK: labelV},
		},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{reqName: "1"}},
		}},
	}

	c, ctx, err := runReconcileTestcase(p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch the deployment and make sure the annotations show the
	// deleted resource.
	d := &appsv1.Deployment{}
	err = c.Get(ctx, types.NamespacedName{
		Namespace: pod.GetNamespace(),
		Name:      pod.GetName(),
	}, d)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := d.Spec.Template.ObjectMeta.Annotations[reqName], "2"; !strings.HasPrefix(got, "2") {
		t.Fatalf("got %v, wants annotation value to have prefix %v", got, want)
	}

}

func TestReconcileState32RolloutStrategyNone(t *testing.T) {
	const (
		wantRequeue = false
		wantStatus  = metav1.ConditionTrue
		wantReason  = cloudsqlapi.ReasonFinishedReconcile
		labelK      = "app"
		labelV      = "things"
	)

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{
		RolloutStrategy: cloudsqlapi.NoneStrategy,
	}
	p.Generation = 2
	addFinalizers(p)
	addSelectorWorkload(p, "Deployment", labelK, labelV)

	// mimic a deployment that was updated by the webhook
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{labelK: labelV},
		},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"annotation": "set",
			}},
		}},
	}

	c, ctx, err := runReconcileTestcase(p, []client.Object{p, deployment}, wantRequeue, wantStatus, wantReason)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch the deployment and make sure the annotations show the
	// deleted resource.
	d := &appsv1.Deployment{}
	err = c.Get(ctx, types.NamespacedName{
		Namespace: deployment.GetNamespace(),
		Name:      deployment.GetName(),
	}, d)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(d.Spec.Template.ObjectMeta.Annotations), 1; got != want {
		t.Fatalf("got %v annotations, wants %v annotations", got, want)
	}

}

func TestReconcileState33(t *testing.T) {
	const (
		wantRequeue = false
		wantStatus  = metav1.ConditionTrue
		wantReason  = cloudsqlapi.ReasonFinishedReconcile
		labelK      = "app"
		labelV      = "things"
	)

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Generation = 1
	addFinalizers(p)
	addSelectorWorkload(p, "Deployment", labelK, labelV)

	// mimic a pod that was updated by the webhook
	reqName, reqVal := workload.PodAnnotation(p, workload.DefaultProxyImage)
	pod := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{labelK: labelV},
		},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{reqName: reqVal}},
		}},
	}

	_, _, err := runReconcileTestcase(p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)
	if err != nil {
		t.Fatal(err)
	}

}

func TestReconcileDeleteUpdatesWorkload(t *testing.T) {
	const (
		labelK = "app"
		labelV = "things"
	)
	resource := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	resource.Generation = 1
	addFinalizers(resource)
	addSelectorWorkload(resource, "Deployment", labelK, labelV)

	k, v := workload.PodAnnotation(resource, workload.DefaultProxyImage)

	// mimic a deployment that was updated by the webhook
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{labelK: labelV},
		},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{k: v}},
		}},
	}

	// Build a client with the resource and deployment
	cb, _, err := clientBuilder()
	if err != nil {
		t.Error(err) // shouldn't ever happen
	}
	c := cb.WithObjects(resource, deployment).Build()
	r, req, ctx := reconciler(resource, c, workload.DefaultProxyImage)

	// Delete the resource
	c.Delete(ctx, resource)
	if err != nil {
		t.Error(err)
	}

	// Run Reconcile on the deleted resource
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Error(err)
	}
	if res.Requeue {
		t.Errorf("got %v, want %v for requeue", res.Requeue, false)
	}

	// Check that the resource doesn't exist anymore
	err = c.Get(ctx, types.NamespacedName{
		Namespace: resource.GetNamespace(),
		Name:      resource.GetName(),
	}, resource)
	if err != nil {
		if !errors.IsNotFound(err) {
			t.Errorf("wants not found error, got %v", err)
		}
	} else {
		t.Error("wants not found error, got no error")
	}

	// Fetch the deployment and make sure the annotations show the
	// deleted resource.
	d := &appsv1.Deployment{}
	err = c.Get(ctx, types.NamespacedName{
		Namespace: deployment.GetNamespace(),
		Name:      deployment.GetName(),
	}, d)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := d.Spec.Template.ObjectMeta.Annotations[k], "1-deleted-"; !strings.HasPrefix(got, want) {
		t.Fatalf("got %v, wants annotation value to have prefix %v", got, want)
	}

}

func TestWorkloadUpdatedAfterDefaultProxyImageChanged(t *testing.T) {
	const (
		labelK = "app"
		labelV = "things"
	)
	resource := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	resource.Generation = 1
	addFinalizers(resource)
	addSelectorWorkload(resource, "Deployment", labelK, labelV)

	// Deployment annotation should be updated to this after reconcile:
	_, wantV := workload.PodAnnotation(resource, "gcr.io/cloud-sql-connectors/cloud-sql-proxy:999.9.9")

	// mimic a deployment that was updated by the webhook
	// annotate the deployment with the default image
	k, v := workload.PodAnnotation(resource, "gcr.io/cloud-sql-connectors/cloud-sql-proxy:1.1.1")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{labelK: labelV},
		},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{k: v}},
		}},
	}

	// Build a client with the resource and deployment
	cb, _, err := clientBuilder()
	if err != nil {
		t.Error(err) // shouldn't ever happen
	}
	c := cb.WithObjects(resource, deployment).Build()

	// Create a reconciler with the default proxy image at a different version
	r, req, ctx := reconciler(resource, c, "gcr.io/cloud-sql-connectors/cloud-sql-proxy:999.9.9")

	// Run Reconcile on the resource
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Error(err)
	}

	if !res.Requeue {
		t.Errorf("got %v, want %v for requeue", res.Requeue, true)
	}

	// Fetch the deployment and make sure the annotations show the
	// deleted resource.
	d := &appsv1.Deployment{}
	err = c.Get(ctx, types.NamespacedName{
		Namespace: deployment.GetNamespace(),
		Name:      deployment.GetName(),
	}, d)
	if err != nil {
		t.Fatal(err)
	}

	if got := d.Spec.Template.ObjectMeta.Annotations[k]; got != wantV {
		t.Fatalf("got %v, wants annotation value %v", got, wantV)
	}

}

func runReconcileTestcase(p *cloudsqlapi.AuthProxyWorkload, clientObjects []client.Object, wantRequeue bool, wantStatus metav1.ConditionStatus, wantReason string) (client.WithWatch, context.Context, error) {
	cb, _, err := clientBuilder()
	if err != nil {
		return nil, nil, err // shouldn't ever happen
	}

	c := cb.WithObjects(clientObjects...).Build()

	r, req, ctx := reconciler(p, c, workload.DefaultProxyImage)
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if res.Requeue != wantRequeue {
		return nil, nil, fmt.Errorf("got %v, want %v for requeue", res.Requeue, wantRequeue)
	}

	for _, o := range clientObjects {
		c.Get(ctx, types.NamespacedName{
			Namespace: o.GetNamespace(),
			Name:      o.GetName(),
		}, o)
	}

	if wantStatus != "" || wantReason != "" {
		cond := findCondition(p.Status.Conditions, cloudsqlapi.ConditionUpToDate)
		if cond == nil {
			return nil, nil, fmt.Errorf("the UpToDate condition was nil, wants condition to exist")
		}
		if wantStatus != "" && cond.Status != wantStatus {
			return nil, nil, fmt.Errorf("got %v, want %v for UpToDate condition status", cond.Status, wantStatus)
		}
		if wantReason != "" && cond.Reason != wantReason {
			return nil, nil, fmt.Errorf("got %v, want %v for UpToDate condition reason", cond.Reason, wantReason)
		}
	}

	return c, ctx, nil
}

func clientBuilder() (*fake.ClientBuilder, *runtime.Scheme, error) {
	scheme, err := cloudsqlapi.SchemeBuilder.Build()
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
	return fake.NewClientBuilder().WithScheme(scheme), scheme, nil

}

func reconciler(p *cloudsqlapi.AuthProxyWorkload, cb client.Client, defaultProxyImage string) (*AuthProxyWorkloadReconciler, ctrl.Request, context.Context) {
	ctx := log.IntoContext(context.Background(), logger)
	r := &AuthProxyWorkloadReconciler{
		Client:          cb,
		recentlyDeleted: &recentlyDeletedCache{},
		updater:         workload.NewUpdater("cloud-sql-proxy-operator/dev", defaultProxyImage),
	}
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
	}
	return r, req, ctx
}

func addFinalizers(p *cloudsqlapi.AuthProxyWorkload) {
	p.Finalizers = []string{finalizerName}
}
func addPodWorkload(p *cloudsqlapi.AuthProxyWorkload) {
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind: "Pod",
		Name: "testpod",
	}
}
func addSelectorWorkload(p *cloudsqlapi.AuthProxyWorkload, kind, labelK, labelV string) {
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind: kind,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{labelK: labelV},
		},
	}
}
