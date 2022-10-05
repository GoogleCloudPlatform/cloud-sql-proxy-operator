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

package controllers

import (
	"context"
	"os"
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/names"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var logger = zap.New(zap.UseFlagOptions(&zap.Options{
	Development: true,
	TimeEncoder: zapcore.ISO8601TimeEncoder,
}))

func TestMain(m *testing.M) {
	// logger is the test logger used by the integration tests and server.
	ctrl.SetLogger(logger)

	result := m.Run()
	os.Exit(result)
}

func TestReconcileState11(t *testing.T) {
	wantRequeue := true

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")

	cb, err := clientBuilder()
	if err != nil {
		t.Error(err) // shouldn't ever happen
	}
	c := cb.WithObjects(p).Build()
	r, req, ctx := reconciler(p, c)

	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Error(err)
	}
	err = c.Get(ctx, types.NamespacedName{
		Namespace: p.GetNamespace(),
		Name:      p.GetName(),
	}, p)

	if err != nil {
		t.Error(err)
	}

	if res.Requeue != wantRequeue {
		t.Errorf("got %v, want %v for requeue", res.Requeue, wantRequeue)
	}
	if len(p.GetFinalizers()) == 0 {
		t.Error("got no finalizers, wants finalizer.")
	}
}

func TestReconcileDeleted(t *testing.T) {
	wantRequeue := false

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "thing",
	}

	cb, err := clientBuilder()
	if err != nil {
		t.Error(err) // shouldn't ever happen
	}
	c := cb.WithObjects(p).Build()
	r, req, ctx := reconciler(p, c)

	c.Delete(ctx, p)
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Error(err)
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

	if res.Requeue != wantRequeue {
		t.Errorf("got %v, want %v for requeue", res.Requeue, wantRequeue)
	}
}

func TestReconcileState21ByName(t *testing.T) {
	wantRequeue := false
	wantStatus := metav1.ConditionTrue
	wantReason := cloudsqlapi.ReasonNoWorkloadsFound

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Name:      "testpod",
		Namespace: "default",
	}

	assertReconcileResult(t, p, []client.Object{p}, wantRequeue, wantStatus, wantReason)
}
func TestReconcileState21BySelector(t *testing.T) {
	wantRequeue := false
	wantStatus := metav1.ConditionTrue
	wantReason := cloudsqlapi.ReasonNoWorkloadsFound

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}

	assertReconcileResult(t, p, []client.Object{p}, wantRequeue, wantStatus, wantReason)
}

func TestReconcileState22ByName(t *testing.T) {
	wantRequeue := true
	wantStatus := metav1.ConditionFalse
	wantReason := cloudsqlapi.ReasonStartedReconcile

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "thing",
	}
	p.Status.Conditions = []*metav1.Condition{{
		Type:   cloudsqlapi.ConditionUpToDate,
		Reason: cloudsqlapi.ReasonFinishedReconcile,
		Status: metav1.ConditionTrue,
	}}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{"app": "things"},
		},
	}

	assertReconcileResult(t, p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)
}

func TestReconcileState22BySelector(t *testing.T) {
	wantRequeue := true
	wantStatus := metav1.ConditionFalse
	wantReason := cloudsqlapi.ReasonStartedReconcile

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{"app": "things"},
		},
	}

	assertReconcileResult(t, p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)
}

func TestReconcileState31(t *testing.T) {
	wantRequeue := false
	wantStatus := metav1.ConditionTrue
	wantReason := cloudsqlapi.ReasonFinishedReconcile

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Generation = 1
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}
	p.Status.Conditions = []*metav1.Condition{{
		Type:   cloudsqlapi.ConditionUpToDate,
		Reason: cloudsqlapi.ReasonStartedReconcile,
		Status: metav1.ConditionFalse,
	}}

	// mimic a pod that was updated by the webhook
	resultName := cloudsqlapi.AnnotationPrefix + "/" +
		names.SafePrefixedName("app-", p.Namespace+"-"+p.Name)
	reqName := cloudsqlapi.AnnotationPrefix + "/" +
		names.SafePrefixedName("req-", p.Namespace+"-"+p.Name)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "thing",
			Namespace:   "default",
			Labels:      map[string]string{"app": "things"},
			Annotations: map[string]string{resultName: "1", reqName: "1"},
		},
	}

	assertReconcileResult(t, p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)

	// check that the generation was set
	wls := internal.WorkloadStatus(p, &internal.PodWorkload{Pod: pod})
	if wls.LastRequstGeneration != "1" {
		t.Errorf("got %v, want %v, workload status LastRequstGeneration", wls.LastRequstGeneration, "1")
	}
	if wls.LastUpdatedGeneration != "1" {
		t.Errorf("got %v, want %v. workload status LastUpdatedGeneration", wls.LastUpdatedGeneration, "1")
	}
}

func TestReconcileState32(t *testing.T) {
	wantRequeue := true
	wantStatus := metav1.ConditionFalse
	wantReason := cloudsqlapi.ReasonStartedReconcile

	p := helpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Generation = 1
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}
	p.Status.Conditions = []*metav1.Condition{{
		Type:   cloudsqlapi.ConditionUpToDate,
		Reason: cloudsqlapi.ReasonStartedReconcile,
		Status: metav1.ConditionFalse,
	}}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{"app": "things"},
		},
	}

	assertReconcileResult(t, p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)

	wls := internal.WorkloadStatus(p, &internal.PodWorkload{Pod: pod})
	if wls.LastRequstGeneration != "1" {
		t.Errorf("got %v, want %v, workload status LastRequstGeneration", wls.LastRequstGeneration, "1")
	}
	if wls.LastUpdatedGeneration != "" {
		t.Errorf("got %v, want %v. workload status LastUpdatedGeneration", wls.LastUpdatedGeneration, "")
	}
}

func assertReconcileResult(t *testing.T, p *cloudsqlapi.AuthProxyWorkload, clientObjects []client.Object, wantRequeue bool, wantStatus metav1.ConditionStatus, wantReason string) {
	t.Helper()
	cb, err := clientBuilder()
	if err != nil {
		t.Error(err) // shouldn't ever happen
	}

	c := cb.WithObjects(clientObjects...).Build()

	r, req, ctx := reconciler(p, c)
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Error(err)
	}
	if res.Requeue != wantRequeue {
		t.Errorf("got %v, want %v for requeue", res.Requeue, wantRequeue)
	}

	for _, o := range clientObjects {
		c.Get(ctx, types.NamespacedName{
			Namespace: o.GetNamespace(),
			Name:      o.GetName(),
		}, o)
	}

	cond := findCondition(p.Status.Conditions, cloudsqlapi.ConditionUpToDate)
	if cond == nil {
		t.Error("UpToDate condition was nil, wants condition to exist.")
		return
	}
	if cond.Status != wantStatus {
		t.Errorf("got %v, want %v for UpToDate condition status", cond.Status, wantStatus)
	}
	if cond.Reason != wantReason {
		t.Errorf("got %v, want %v for UpToDate condition reason", cond.Reason, wantReason)
	}
}

func clientBuilder() (*fake.ClientBuilder, error) {
	scheme, err := cloudsqlapi.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return fake.NewClientBuilder().WithScheme(scheme), nil

}

func reconciler(p *cloudsqlapi.AuthProxyWorkload, cb client.Client) (*AuthProxyWorkloadReconciler, ctrl.Request, context.Context) {
	ctx := log.IntoContext(context.Background(), logger)
	r := &AuthProxyWorkloadReconciler{}
	r.Client = cb
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
	}
	return r, req, ctx
}
