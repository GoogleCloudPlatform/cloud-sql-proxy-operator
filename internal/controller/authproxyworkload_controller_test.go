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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
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

func TestMain(_ *testing.M) {
	// logger is the test logger used by the testintegration tests and server.
	ctrl.SetLogger(logger)

	//result := m.Run()
	//os.Exit(result)
	fmt.Printf("TODO: Reenable when new reconcile algorithm is complete.")
}

func TestReconcileState11(t *testing.T) {

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")

	runReconcileTestcase(t, p, []client.Object{p}, true, "", "")
}

func TestReconcileDeleted(t *testing.T) {

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
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
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind:      "Pod",
		Name:      "testpod",
		Namespace: "default",
	}

	runReconcileTestcase(t, p, []client.Object{p}, false, metav1.ConditionTrue, v1alpha1.ReasonNoWorkloadsFound)
}
func TestReconcileState21BySelector(t *testing.T) {
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}

	runReconcileTestcase(t, p, []client.Object{p}, false, metav1.ConditionTrue, v1alpha1.ReasonNoWorkloadsFound)
}

func TestReconcileState22ByName(t *testing.T) {
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "thing",
	}
	p.Status.Conditions = []*metav1.Condition{{
		Type:   v1alpha1.ConditionUpToDate,
		Reason: v1alpha1.ReasonFinishedReconcile,
		Status: metav1.ConditionTrue,
	}}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{"app": "things"},
		},
	}

	runReconcileTestcase(t, p, []client.Object{p, pod}, true, metav1.ConditionFalse, v1alpha1.ReasonStartedReconcile)
}

func TestReconcileState22BySelector(t *testing.T) {
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
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

	runReconcileTestcase(t, p, []client.Object{p, pod}, true, metav1.ConditionFalse, v1alpha1.ReasonStartedReconcile)
}

func TestReconcileState31(t *testing.T) {
	var wantRequeue bool
	wantStatus := metav1.ConditionTrue
	wantReason := v1alpha1.ReasonFinishedReconcile

	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Generation = 1
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}
	p.Status.Conditions = []*metav1.Condition{{
		Type:   v1alpha1.ConditionUpToDate,
		Reason: v1alpha1.ReasonStartedReconcile,
		Status: metav1.ConditionFalse,
	}}

	// mimic a pod that was updated by the webhook
	resultName := v1alpha1.AnnotationPrefix + "/" +
		workload.SafePrefixedName("app-", p.Namespace+"-"+p.Name)
	reqName := v1alpha1.AnnotationPrefix + "/" +
		workload.SafePrefixedName("req-", p.Namespace+"-"+p.Name)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "thing",
			Namespace:   "default",
			Labels:      map[string]string{"app": "things"},
			Annotations: map[string]string{resultName: "1", reqName: "1"},
		},
	}
	wantWls := workload.WorkloadUpdateStatus{LastGeneration: "1", ThisGeneration: "1"}
	_, _, r := runReconcileTestcase(t, p, []client.Object{p, pod}, wantRequeue, wantStatus, wantReason)
	assertWorkloadUpdateStatus(t, r, p, pod, wantWls)
	wls := r.updater.Status(p, &workload.PodWorkload{Pod: pod})
	if wls.LastGeneration != wantWls.LastGeneration {
		t.Errorf("got %v, want %v, workload status LastRequstGeneration", wls.LastGeneration, wantWls.LastGeneration)
	}
	if wls.ThisGeneration != wantWls.ThisGeneration {
		t.Errorf("got %v, want %v. workload status LastUpdatedGeneration", wls.ThisGeneration, wantWls.ThisGeneration)
	}

}

func TestReconcileState32(t *testing.T) {
	p := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "test",
	}, "project:region:db")
	p.Generation = 1
	p.Finalizers = []string{finalizerName}
	p.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind:      "Pod",
		Namespace: "default",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "things"},
		},
	}
	p.Status.Conditions = []*metav1.Condition{{
		Type:   v1alpha1.ConditionUpToDate,
		Reason: v1alpha1.ReasonStartedReconcile,
		Status: metav1.ConditionFalse,
	}}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thing",
			Namespace: "default",
			Labels:    map[string]string{"app": "things"},
		},
	}
	wantWls := workload.WorkloadUpdateStatus{LastGeneration: "", ThisGeneration: "1"}

	_, _, r := runReconcileTestcase(t, p, []client.Object{p, pod}, true, metav1.ConditionFalse, v1alpha1.ReasonStartedReconcile)
	assertWorkloadUpdateStatus(t, r, p, pod, wantWls)
}

func assertWorkloadUpdateStatus(t *testing.T, r *AuthProxyWorkloadReconciler, p *v1alpha1.AuthProxyWorkload, pod *corev1.Pod, wantWls workload.WorkloadUpdateStatus) {
	wls := r.updater.Status(p, &workload.PodWorkload{Pod: pod})
	if wls.LastGeneration != wantWls.LastGeneration {
		t.Errorf("got %v, want %v, workload status LastRequstGeneration", wls.LastGeneration, wantWls.LastGeneration)
	}
}

func runReconcileTestcase(t *testing.T, p *v1alpha1.AuthProxyWorkload, clientObjects []client.Object, wantRequeue bool, wantStatus metav1.ConditionStatus, wantReason string) (context.Context, client.WithWatch, *AuthProxyWorkloadReconciler) {
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

	if wantStatus != "" || wantReason != "" {
		cond := findCondition(p.Status.Conditions, v1alpha1.ConditionUpToDate)
		if cond == nil {
			t.Error("UpToDate condition was nil, wants condition to exist.")
			return ctx, c, nil
		}
		if wantStatus != "" && cond.Status != wantStatus {
			t.Errorf("got %v, want %v for UpToDate condition status", cond.Status, wantStatus)
		}
		if wantReason != "" && cond.Reason != wantReason {
			t.Errorf("got %v, want %v for UpToDate condition reason", cond.Reason, wantReason)
		}
	}

	return ctx, c, r
}

func clientBuilder() (*fake.ClientBuilder, error) {
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return fake.NewClientBuilder().WithScheme(scheme), nil

}

func reconciler(p *v1alpha1.AuthProxyWorkload, cb client.Client) (*AuthProxyWorkloadReconciler, ctrl.Request, context.Context) {
	ctx := log.IntoContext(context.Background(), logger)
	r := &AuthProxyWorkloadReconciler{
		Client:          cb,
		recentlyDeleted: &recentlyDeletedCache{},
		updater:         workload.NewUpdater(),
	}
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
	}
	return r, req, ctx
}
