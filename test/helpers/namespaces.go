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
		t.Fatalf("Unable to verify existance of namespace %v, %v", name, err)
	}

	var retrievedNs corev1.Namespace
	err = RetryUntilSuccess(t, 5, time.Second*5, func() error {
		return k8sClient.Get(ctx, client.ObjectKey{Namespace: "", Name: name}, &retrievedNs)
	})

	if err != nil {
		t.Fatalf("Unable to verify existance of namespace %v, %v", name, err)
	}

}
