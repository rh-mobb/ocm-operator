package controllers

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ErrMissingClusterID = errors.New("unable to find cluster id")
)

// ErrGetOCM returns an error indicating an object was unable to be retrieved from OCM.
func ErrGetOCM(request Request, err error) error {
	return fmt.Errorf(
		"unable to get [%s] with name [%s] from ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// ErrCreateOCM returns an error indicating an object was unable to be created in OCM.
func ErrCreateOCM(request Request, err error) error {
	return fmt.Errorf(
		"unable to create [%s] with name [%s] in ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// ErrUpdateOCM returns an error indicating an object was unable to be updated in OCM.
func ErrUpdateOCM(request Request, err error) error {
	return fmt.Errorf(
		"unable to update [%s] with name [%s] in ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// ErrDeleteOCM returns an error indicating an object was unable to be deleted from OCM.
func ErrDeleteOCM(request Request, err error) error {
	return fmt.Errorf(
		"unable to delete [%s] with name [%s] from ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// ErrUpdateDeletedCondition returns an error indicating an object was unable to update
// the deleted condition.
func ErrUpdateDeletedCondition(err error) error {
	return fmt.Errorf("error updating deleted condition - %w", err)
}

// ErrUpdateReconcilingCondition returns an error indicating an object was unable to update
// the reconciling condition.
func ErrUpdateReconcilingCondition(err error) error {
	return fmt.Errorf("error updating reconciling condition - %w", err)
}

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
