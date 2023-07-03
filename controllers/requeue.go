package controllers

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	defaultRequeue                = 30 * time.Second
	defaultMissingUpstreamRequeue = 60 * time.Second
)

// RequeueAfter returns a requeue result to requeue after a specific
// number of seconds.
func RequeueAfter(seconds time.Duration) ctrl.Result {
	return ctrl.Result{Requeue: true, RequeueAfter: seconds}
}

// NoRequeue returns a blank result to prevent a requeue.
func NoRequeue() ctrl.Result {
	return ctrl.Result{}
}

// RequeueOnError returns a requeue result with an error.
func RequeueOnError(request Request, err error) (ctrl.Result, error) {
	return RequeueAfter(request.DefaultRequeue()), err
}

// CustomRequeueOnError returns a requeue result with an error.
func CustomRequeueOnError(requeue time.Duration, err error) (ctrl.Result, error) {
	return RequeueAfter(requeue), err
}
