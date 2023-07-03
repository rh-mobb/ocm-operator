package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/scottd018/go-utils/pkg/list"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultFinalizerSuffix = "finalizer"
)

// FinalizerName returns the finalizer name for the controller.
func FinalizerName(object client.Object) string {
	return strings.ToLower(fmt.Sprintf("%s.%s/%s",
		object.GetObjectKind().GroupVersionKind().Kind,
		object.GetObjectKind().GroupVersionKind().Group,
		defaultFinalizerSuffix,
	))
}

// HasFinalizer is a helper function to make the code more readable.
func HasFinalizer(object client.Object) bool {
	return list.Strings(object.GetFinalizers()).Has(FinalizerName(object))
}

// AddFinalizer adds finalizers to the object so that the delete lifecycle can be run
// before the object is deleted.
func AddFinalizer(ctx context.Context, r kubernetes.Client, object client.Object) error {
	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if !HasFinalizer(object) {
		original, ok := object.DeepCopyObject().(client.Object)
		if !ok {
			return ErrConvertClientObject
		}

		controllerutil.AddFinalizer(object, FinalizerName(object))

		if err := r.Patch(ctx, object, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("unable to add finalizer - %w", err)
		}
	}

	return nil
}

// RemoveFinalizer removes finalizers from the object.  It is intended to be run after an
// external object is deleted so that the delete lifecycle may continue reconciliation.
func RemoveFinalizer(ctx context.Context, r kubernetes.Client, object client.Object) error {
	if HasFinalizer(object) {
		original, ok := object.DeepCopyObject().(client.Object)
		if !ok {
			return ErrConvertClientObject
		}

		controllerutil.RemoveFinalizer(object, FinalizerName(object))

		if err := r.Patch(ctx, object, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("unable to remove finalizer - %w", err)
		}
	}

	return nil
}
