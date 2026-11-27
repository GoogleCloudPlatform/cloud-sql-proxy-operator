// Copyright 2026 Google LLC.
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

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
)

func TestUpgradeDefaultProxyOnStartupRetriesOnConflict(t *testing.T) {
	defaultImage := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "default-image",
	}, "project:region:db")
	customImage := testhelpers.BuildAuthProxyWorkload(types.NamespacedName{
		Namespace: "default",
		Name:      "custom-image",
	}, "project:region:db")
	customImage.Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{Image: "example.com/proxy:1.0"}

	cb, _, err := clientBuilder()
	if err != nil {
		t.Fatal(err)
	}
	base := cb.WithObjects(defaultImage, customImage).Build()

	updates := map[string]int{}
	c := interceptor.NewClient(base, interceptor.Funcs{
		Update: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			updates[obj.GetName()]++
			if updates[obj.GetName()] == 1 {
				// The reconciler writes the resource after the startup task read
				// it, so the startup task's first update carries a stale
				// resourceVersion.
				latest := &cloudsqlapi.AuthProxyWorkload{}
				if err := cl.Get(ctx, client.ObjectKeyFromObject(obj), latest); err != nil {
					return err
				}
				latest.Labels = map[string]string{"written-by": "reconciler"}
				if err := cl.Update(ctx, latest); err != nil {
					return err
				}
			}
			return cl.Update(ctx, obj, opts...)
		},
	})

	u := &upgradeDefaultProxyOnStartup{c: c, reader: c}
	err = u.Start(log.IntoContext(context.Background(), logger))
	if err != nil {
		t.Fatal(err)
	}

	if got := updates["default-image"]; got != 2 {
		t.Errorf("got %d updates for default-image, want 2 (one conflict, one retry)", got)
	}
	if got := updates["custom-image"]; got != 0 {
		t.Errorf("got %d updates for custom-image, want 0", got)
	}

	got := &cloudsqlapi.AuthProxyWorkload{}
	err = base.Get(context.Background(), client.ObjectKeyFromObject(defaultImage), got)
	if err != nil {
		t.Fatal(err)
	}
	if got.Labels["written-by"] != "reconciler" {
		t.Errorf("got labels %v, want the reconciler's write to survive the retry", got.Labels)
	}
}
