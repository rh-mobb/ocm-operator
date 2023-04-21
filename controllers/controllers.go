package controllers

import (
	"fmt"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultFinalizerSuffix = "finalizer"
)

// requeueAfter returns a requeue result to requeue after a specific
// number of seconds.
func requeueAfter(seconds time.Duration) ctrl.Result {
	return ctrl.Result{RequeueAfter: seconds}
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

// finalizerName returns the finalizer name for the controller.
func finalizerName(object client.Object) string {
	return strings.ToLower(fmt.Sprintf("%s.%s/%s",
		object.GetObjectKind().GroupVersionKind().Kind,
		object.GetObjectKind().GroupVersionKind().Group,
		defaultFinalizerSuffix,
	))
}
