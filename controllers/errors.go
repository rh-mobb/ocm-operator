package controllers

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ErrMissingClusterID = errors.New("unable to find cluster id")
)

// TypeConvertError returns an error indicating a generic controllers.Request interface
// could not be converted to its underlying type.
func TypeConvertError(t interface{}) error {
	return fmt.Errorf("unable to convert controllers.Request interface to underlying request type [%T]", t)
}

// AddFinalizerError returns an error when registering a finalizer.
func AddFinalizerError(err error) error {
	return fmt.Errorf("unable to add finalizers - %w", err)
}

// RemoveFinalizerError returns an error when removing a finalizer.
func RemoveFinalizerError(err error) error {
	return fmt.Errorf("unable to remove finalizers - %w", err)
}

// ReconcileError returns an error for the reconciler.  It is a helper function to
// pass consistent errors across multiple different controllers.
func ReconcileError(request reconcile.Request, message string, err error) error {
	// return a nil error if we received a nil error
	if err == nil {
		return nil
	}

	return fmt.Errorf(
		"request=%s/%s, message=%s - %w",
		request.Namespace,
		request.Name,
		message,
		err,
	)
}
