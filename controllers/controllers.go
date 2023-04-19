package controllers

import (
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// requeue returns the default controller result when a custom one
// is not needed.
func requeue() ctrl.Result {
	return ctrl.Result{Requeue: true}
}

// requeueAfter returns a requeue result to requeue after a specific
// number of seconds.
func requeueAfter(seconds time.Duration) ctrl.Result {
	return ctrl.Result{RequeueAfter: seconds * time.Second}
}

// noRequeue returns a blank result to prevent a requeue.
func noRequeue() ctrl.Result {
	return ctrl.Result{}
}

// reconcilerError returns an error for the reconciler.  It is a helper function to
// pass consistent errors across multiple different controllers.
func reconcilerError(request reconcile.Request, message string, err error) error {
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
