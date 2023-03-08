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

package workload

import (
	"os"
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestMain(m *testing.M) {
	// logger is the test logger used by the testintegration tests and server.
	logger := zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}))
	ctrl.SetLogger(logger)

	result := m.Run()
	os.Exit(result)
}

func TestWorkloadMatches(t *testing.T) {
	type workloadTestCase struct {
		wl    Workload
		match bool
		desc  string
	}

	type testcases struct {
		desc string
		sel  cloudsqlapi.WorkloadSelectorSpec
		tc   []workloadTestCase
	}
	cases := []testcases{{
		desc: "match kind, name, and namespace",
		sel: cloudsqlapi.WorkloadSelectorSpec{
			Kind: "Pod",
			Name: "hello",
		},
		tc: []workloadTestCase{
			{
				wl:    workload(t, "Pod", "default", "hello"),
				match: true,
				desc:  "matching pod",
			},
			{
				wl:    workload(t, "Pod", "default", "hello", "app", "pod"),
				match: true,
				desc:  "matching pod with extra label",
			},
			{
				wl:    workload(t, "Pod", "default", "pod", "app", "pod"),
				match: false,
				desc:  "pod with different name",
			},
			{
				wl:    workload(t, "Pod", "other", "hello"),
				match: false,
				desc:  "pod with different namespace",
			},
			{
				wl:    workload(t, "Deployment", "default", "hello"),
				match: false,
				desc:  "different kind",
			},
		},
	},
		{
			desc: "match kind, namespace, and labels",
			sel: cloudsqlapi.WorkloadSelectorSpec{
				Kind: "Pod",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "hello"},
				},
			},
			tc: []workloadTestCase{
				{
					wl:    workload(t, "Pod", "default", "hello"),
					match: false,
					desc:  "label not set",
				},
				{
					wl:    workload(t, "Pod", "default", "hello", "app", "hello", "type", "frontend"),
					match: true,
					desc:  "matching pod with extra label",
				},
				{
					wl:    workload(t, "Pod", "default", "pod", "app", "nope"),
					match: false,
					desc:  "pod with different label",
				},
				{
					wl:    workload(t, "Pod", "Other", "hello", "app", "hello", "type", "frontend"),
					match: false,
					desc:  "pod with different namespace",
				},
				{
					wl:    workload(t, "Deployment", "default", "hello", "app", "hello", "type", "frontend"),
					match: false,
					desc:  "Deploymnet with different namespace",
				},
			},
		},
		{
			desc: "match namespace, and labels",
			sel: cloudsqlapi.WorkloadSelectorSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "hello"},
				},
			},
			tc: []workloadTestCase{
				{
					wl:    workload(t, "Pod", "default", "hello", "app", "hello", "type", "frontend"),
					match: true,
					desc:  "matching pod with extra label",
				},
				{
					wl:    workload(t, "Pod", "default", "pod", "app", "nope"),
					match: false,
					desc:  "pod with different label",
				},
				{
					wl:    workload(t, "Deployment", "default", "hello", "app", "hello", "type", "frontend"),
					match: true,
					desc:  "deployment with extra label",
				},
				{
					wl:    workload(t, "Deployment", "default", "pod", "app", "nope"),
					match: false,
					desc:  "deployment with different label",
				},
				{
					wl:    workload(t, "StatefulSet", "default", "things"),
					match: false,
					desc:  "StatefulSet no labels",
				},
			},
		},
		{
			desc: "match namespace, and label expression",
			sel: cloudsqlapi.WorkloadSelectorSpec{
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"hello"},
					}},
				},
			},
			tc: []workloadTestCase{
				{
					wl:    workload(t, "Pod", "default", "hello", "app", "hello", "type", "frontend"),
					match: true,
					desc:  "matching pod with extra label",
				},
				{
					wl:    workload(t, "Pod", "other", "hello", "app", "hello", "type", "frontend"),
					match: false,
					desc:  "pod with matching label, different namespace",
				},
				{
					wl:    workload(t, "Pod", "default", "pod", "app", "nope"),
					match: false,
					desc:  "pod with different label",
				},
				{
					wl:    workload(t, "Deployment", "default", "hello", "app", "hello", "type", "frontend"),
					match: true,
					desc:  "deployment with extra label",
				},
				{
					wl:    workload(t, "Deployment", "default", "pod", "app", "nope"),
					match: false,
					desc:  "deployment with different label",
				},
			},
		},
	}

	for _, sel := range cases {
		for _, tc := range sel.tc {
			t.Run(sel.desc+" "+tc.desc, func(t *testing.T) {
				gotMatch := workloadMatches(tc.wl.Object(), sel.sel, "default")
				if tc.match != gotMatch {
					t.Errorf("got %v, wants %v. selector %s test %s", gotMatch, tc.match, sel.desc, tc.desc)
				}
			})
		}
	}

}

// workload is shorthand to create workload test inputs
func workload(t *testing.T, kind, ns, name string, l ...string) Workload {
	var v Workload
	switch kind {
	case "Deployment":
		v = &DeploymentWorkload{Deployment: &appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment"}}}
	case "Pod":
		v = &PodWorkload{Pod: &corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod"}}}
	case "StatefulSet":
		v = &StatefulSetWorkload{StatefulSet: &appsv1.StatefulSet{TypeMeta: metav1.TypeMeta{Kind: "StatefulSet"}}}
	case "Job":
		v = &JobWorkload{Job: &batchv1.Job{TypeMeta: metav1.TypeMeta{Kind: "Job"}}}
	case "CronJob":
		v = &CronJobWorkload{CronJob: &batchv1.CronJob{TypeMeta: metav1.TypeMeta{Kind: "CronJob"}}}
	case "DaemonSet":
		v = &DaemonSetWorkload{DaemonSet: &appsv1.DaemonSet{TypeMeta: metav1.TypeMeta{Kind: "DaemonSet"}}}
	default:
		t.Fatalf("Workload kind %s not supported", kind)
	}
	v.Object().SetNamespace(ns)
	v.Object().SetName(name)
	if len(l) > 0 {
		if len(l)%2 != 0 {
			t.Fatalf("labels list must have an even number of elements")
		}
		labels := map[string]string{}
		for i := 0; i < len(l); i += 2 {
			labels[l[i]] = l[i+1]
		}
		v.Object().SetLabels(labels)
	}

	return v
}
