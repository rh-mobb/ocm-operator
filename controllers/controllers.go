package controllers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcilerError returns an error for the reconciler.  It is a helper function to
// pass consistent errors across multiple different controllers.
func reconcilerError(request reconcile.Request, message string, err error) error {
	return fmt.Errorf(
		"request=%s/%s, message=%s - %w",
		request.Namespace,
		request.Name,
		message,
		err,
	)
}
