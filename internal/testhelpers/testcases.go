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

package testhelpers

import (
	"context"
	"errors"
	"fmt"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestCaseClient struct {
	Client           client.Client
	Namespace        string
	ConnectionString string
	ProxyImageURL    string
	DBRootUsername   string
	DBRootPassword   string
	DBName           string
}

func NewNamespaceName(prefix string) string {
	return fmt.Sprintf("test%s%d", prefix, rand.IntnRange(1000, 9999))
}

// CreateResource creates a new workload resource in the TestCaseClient's namespace
// waits until the resource exists.
func (cc *TestCaseClient) CreateResource(ctx context.Context) (*cloudsqlapi.AuthProxyWorkload, error) {
	const (
		name            = "instance1"
		expectedConnStr = "proj:inst:db"
	)
	ns := cc.Namespace
	err := cc.CreateOrPatchNamespace(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't create namespace, %v", err)
	}
	key := types.NamespacedName{Name: name, Namespace: ns}
	err = cc.CreateAuthProxyWorkload(ctx, key, "app", expectedConnStr, "Deployment")
	if err != nil {
		return nil, fmt.Errorf("unable to create auth proxy workload %v", err)
	}

	res, err := cc.GetAuthProxyWorkloadAfterReconcile(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("unable to find entity after create %v", err)
	}

	if connStr := res.Spec.Instances[0].ConnectionString; connStr != expectedConnStr {
		return nil, fmt.Errorf("was %v, wants %v, spec.cloudSqlInstance", connStr, expectedConnStr)
	}

	if wlstatus := GetConditionStatus(res.Status.Conditions, cloudsqlapi.ConditionUpToDate); wlstatus != metav1.ConditionTrue {
		return nil, fmt.Errorf("was %v, wants %v, status.condition[up-to-date]", wlstatus, metav1.ConditionTrue)
	}
	return res, nil
}

// WaitForFinalizerOnResource queries the client to see if the resource has
// a finalizer.
func (cc *TestCaseClient) WaitForFinalizerOnResource(ctx context.Context, res *cloudsqlapi.AuthProxyWorkload) error {

	// Make sure the finalizer was added before deleting the resource.
	return RetryUntilSuccess(3, DefaultRetryInterval, func() error {
		err := cc.Client.Get(ctx, client.ObjectKeyFromObject(res), res)
		if err != nil {
			return err
		}
		if len(res.Finalizers) == 0 {
			return errors.New("waiting for finalizer to be set")
		}
		return nil
	})
}

// DeleteResourceAndWait issues a delete request for the resource and then waits for the resource
// to actually be deleted. This will return an error if the resource is not deleted within 15 seconds.
func (cc *TestCaseClient) DeleteResourceAndWait(ctx context.Context, res *cloudsqlapi.AuthProxyWorkload) error {

	err := cc.Client.Delete(ctx, res)
	if err != nil {
		return err
	}

	err = RetryUntilSuccess(3, DefaultRetryInterval, func() error {
		err = cc.Client.Get(ctx, client.ObjectKeyFromObject(res), res)
		// The test passes when this returns an error,
		// because that means the resource was deleted.
		if err != nil {
			return nil
		}
		return fmt.Errorf("was nil, wants error when looking up deleted AuthProxyWorkload resource")
	})

	if err != nil {
		return err
	}

	return nil
}
