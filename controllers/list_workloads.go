// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// listWorkloads produces a list of Workload's that match the WorkloadSelectorSpec
// in the specified namespace.
func (r *AuthProxyWorkloadReconciler) listWorkloads(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	if workloadSelector.Namespace != "" {
		ns = workloadSelector.Namespace
	}

	if workloadSelector.Name != "" {
		return r.loadByName(ctx, workloadSelector, ns)
	}

	return r.loadByLabelSelector(ctx, workloadSelector, ns)
}

// loadByName loads a single workload by name.
func (r *AuthProxyWorkloadReconciler) loadByName(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	var wl internal.Workload

	key := client.ObjectKey{Namespace: ns, Name: workloadSelector.Name}

	wl, err := internal.WorkloadForKind(workloadSelector.Kind)
	if err != nil {
		return nil, fmt.Errorf("unable to load by name %s/%s:  %v", key.Namespace, key.Name, err)
	}

	err = r.Get(ctx, key, wl.Object())
	if err != nil {
		return nil, fmt.Errorf("unable to load resource by name %s/%s:  %v", key.Namespace, key.Name, err)
	}

	return []internal.Workload{wl}, nil
}

// loadByLabelSelector loads workloads matching a label selector
func (r *AuthProxyWorkloadReconciler) loadByLabelSelector(ctx context.Context, workloadSelector cloudsqlapi.WorkloadSelectorSpec, ns string) ([]internal.Workload, error) {
	l := logf.FromContext(ctx)

	sel, err := workloadSelector.LabelsSelector()

	if err != nil {
		return nil, err
	}
	_, gk := schema.ParseKindArg(workloadSelector.Kind)
	wl, err := internal.WorkloadListForKind(gk.Kind)
	if err != nil {
		return nil, err
	}
	err = r.List(ctx, wl.List(), client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		l.Error(err, "Unable to list s for workloadSelector", "selector", sel)
		return nil, err
	}
	return wl.Workloads(), nil

}
