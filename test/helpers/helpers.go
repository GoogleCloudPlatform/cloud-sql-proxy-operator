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

package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateOrPatchNamespace ensures that a namespace exists with the given name
// in kubernetes, or fails the test as fatal.
func CreateOrPatchNamespace(t *testing.T, ctx context.Context, k8sClient client.Client, name string) {
	var newNs = corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_, err := controllerutil.CreateOrPatch(ctx, k8sClient, &newNs, func() error {
		newNs.ObjectMeta.Name = name
		return nil
	})
	if err != nil {
		t.Fatalf("unable to verify existance of namespace %v, %v", name, err)
	}

	var gotNs corev1.Namespace
	err = RetryUntilSuccess(t, 5, time.Second*5, func() error {
		return k8sClient.Get(ctx, client.ObjectKey{Name: name}, &gotNs)
	})

	if err != nil {
		t.Fatalf("unable to verify existance of namespace %v, %v", name, err)
	}

}

// RetryUntilSuccess runs `f` until it no longer returns an error, or it has
// returned an error `attempts` number of times. It waits `sleep` duration
// between failed attempts. It returns the error from the last attempt.
func RetryUntilSuccess(t TestLogger, attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			t.Logf("retrying after attempt %d, %v", i, err)
			time.Sleep(sleep)
			sleep *= 2
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

// TestLogger interface hides the Logf function from *testing.T so that
// the helper package can be reused in both integration and e2e tests.
type TestLogger interface {
	Logf(format string, args ...interface{})
}

type TestSetupLogger struct {
	logr.Logger
}

func (l *TestSetupLogger) Logf(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}
