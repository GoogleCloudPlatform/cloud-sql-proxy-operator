/*
Copyright 2022 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package helpers

import (
	"context"
	"testing"
	"time"

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
