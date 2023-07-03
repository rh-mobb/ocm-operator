package controllers

import (
	"errors"
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	ErrMissingClusterID = errors.New("unable to find cluster id")
)

// TypeConvertError returns an error indicating a generic controllers.Request interface
// could not be converted to its underlying type.
func TypeConvertError(requeue time.Duration, t interface{}) (ctrl.Result, error) {
	return RequeueAfter(requeue), fmt.Errorf(
		"unable to convert controllers.Request interface to underlying request type [%T]",
		t,
	)
}

// FinalizerError returns an error when attempting to act upon a finalizer.
func FinalizerError(requeue time.Duration, err error) (ctrl.Result, error) {
	return RequeueAfter(requeue), err
}

// AddFinalizerError returns an error when registering a finalizer.
func AddFinalizerError(err error) error {
	return fmt.Errorf("unable to add finalizers - %w", err)
}

// RemoveFinalizerError returns an error when removing a finalizer.
func RemoveFinalizerError(err error) error {
	return fmt.Errorf("unable to remove finalizers - %w", err)
}
