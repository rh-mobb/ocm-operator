package controllers

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultFinalizerSuffix = "finalizer"
)

type Workload interface {
	client.Object

	GetConditions() []metav1.Condition
	SetConditions([]metav1.Condition)
}

// requeue returns the default controller result when a custom one
// is not needed.
func requeue() ctrl.Result {
	return ctrl.Result{Requeue: true}
}

// requeueAfter returns a requeue result to requeue after a specific
// number of seconds.
func requeueAfter(seconds time.Duration) ctrl.Result {
	return ctrl.Result{Requeue: true, RequeueAfter: seconds * time.Second}
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

// containsString determines if a string is in an array of strings.
func containsString(list []string, str string) bool {
	for item := range list {
		if str == list[item] {
			return true
		}
	}

	return false
}
