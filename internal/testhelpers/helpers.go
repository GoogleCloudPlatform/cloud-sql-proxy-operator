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

package testhelpers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const DefaultRetryInterval = 5 * time.Second

// CreateOrPatchNamespace ensures that a namespace exists with the given name
// in kubernetes, or fails the test as fatal.
func (cc *TestCaseClient) CreateOrPatchNamespace(ctx context.Context) error {
	var newNS = corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: cc.Namespace},
	}
	_, err := controllerutil.CreateOrPatch(ctx, cc.Client, &newNS, func() error {
		newNS.ObjectMeta.Name = cc.Namespace
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to verify existance of namespace %v, %v", cc.Namespace, err)
	}

	var gotNS corev1.Namespace
	err = RetryUntilSuccess(5, DefaultRetryInterval, func() error {
		return cc.Client.Get(ctx, client.ObjectKey{Name: cc.Namespace}, &gotNS)
	})

	if err != nil {
		return fmt.Errorf("unable to verify existance of namespace %v, %v", cc.Namespace, err)
	}
	return nil

}
func (cc *TestCaseClient) DeleteNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: cc.Namespace},
	}

	return cc.Client.Delete(ctx, ns)
}

// RetryUntilSuccess runs `f` until it no longer returns an error, or it has
// returned an error `attempts` number of times. It sleeps duration `d`
// between failed attempts. It returns the error from the last attempt.
func RetryUntilSuccess(attempts int, d time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(d)
			d *= 2
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
