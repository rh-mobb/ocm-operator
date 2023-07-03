package controllers

import (
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

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
