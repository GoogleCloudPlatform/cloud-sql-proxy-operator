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
	"fmt"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// upgradeDefaultProxyOnStartup is a LeaderElectionRunnable task that will run
// as soon as ControllerRuntime is initialized. It will list all AuthProxyWorkload
// resources in the cluster, and force an update on all resources with the
// default proxy image. If the operator has a different DefaultProxyImage than
// the one used when that AuthProxyWorkload was last reconciled, then Reconcile
// will update the associated workloads in accordance with the RolloutStrategy.
type upgradeDefaultProxyOnStartup struct {
	c client.Client
	// reader reads straight from the API server so that the update below
	// starts from the current resourceVersion instead of the cached one.
	reader client.Reader
}

// Start lists all the AuthProxyWorkload resources and triggers the update on
// the resources with a default proxy image.
func (c *upgradeDefaultProxyOnStartup) Start(ctx context.Context) error {
	l := &cloudsqlapi.AuthProxyWorkloadList{}

	err := c.reader.List(ctx, l)
	if err != nil {
		return fmt.Errorf("can't list AuthProxyWorkload on startup, %v", err)
	}

	for i := range l.Items {
		p := &l.Items[i]
		useDefaultImage := p.Spec.AuthProxyContainer == nil || p.Spec.AuthProxyContainer.Image == ""

		if !useDefaultImage {
			continue
		}

		// If an APW has a default image, then perform an "update" on it so that
		// the reconcile function runs and triggers the appropriate rolling updates.
		log.FromContext(ctx).Info(fmt.Sprintf("Upgrading workload default images for %s/%s", p.Namespace, p.Name))
		err = c.update(ctx, client.ObjectKeyFromObject(p))
		if err != nil {
			log.FromContext(ctx).Error(err, "unable to update AuthProxyWorkload during startup upgrade",
				"name", p.Name, "namespace", p.Namespace)
		}
	}
	return nil
}

// update re-reads the AuthProxyWorkload and updates it. The reconciler is
// already running when Start is called, so it may write the resource between
// the list and this update; on a conflict the update is retried from a fresh
// read instead of being lost.
func (c *upgradeDefaultProxyOnStartup) update(ctx context.Context, key client.ObjectKey) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		p := &cloudsqlapi.AuthProxyWorkload{}
		if err := c.reader.Get(ctx, key, p); err != nil {
			return client.IgnoreNotFound(err)
		}
		return c.c.Update(ctx, p)
	})
}

func (c *upgradeDefaultProxyOnStartup) NeedLeaderElection() bool {
	return true // only run on the leader
}
