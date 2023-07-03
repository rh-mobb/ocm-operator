package controllers

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ErrMissingClusterID = errors.New("unable to find cluster id")
)

// GetOCMError returns an error indicating an object was unable to be retrieved from OCM.
func GetOCMError(request Request, err error) error {
	return fmt.Errorf(
		"unable to get [%s] with name [%s] from ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// CreateOCMError returns an error indicating an object was unable to be created in OCM.
func CreateOCMError(request Request, err error) error {
	return fmt.Errorf(
		"unable to create [%s] with name [%s] in ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// UpdateOCMError returns an error indicating an object was unable to be updated in OCM.
func UpdateOCMError(request Request, err error) error {
	return fmt.Errorf(
		"unable to update [%s] with name [%s] in ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// DeleteOCMError returns an error indicating an object was unable to be deleted from OCM.
func DeleteOCMError(request Request, err error) error {
	return fmt.Errorf(
		"unable to delete [%s] with name [%s] from ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// UpdateDeletedConditionError returns an error indicating an object was unable to update
// the deleted condition.
func UpdateDeletedConditionError(err error) error {
	return fmt.Errorf("error updating deleted condition - %w", err)
}

// UpdateReconcilingConditionError returns an error indicating an object was unable to update
// the reconciling condition.
func UpdateReconcilingConditionError(err error) error {
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
