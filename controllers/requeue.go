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

// RequeueOnError returns a requeue result with an error.  It is a helper
// function to reduce the amount of times you requeue with errors in a controller
// as this often times becomes unreadable.
func RequeueOnError(request Request, err error) (ctrl.Result, error) {
	return RequeueAfter(request.DefaultRequeue()), err
}

// Requeue returns a requeue result without an error.  It is a helper
// function to reduce the amount of times you requeue without errors in a controller
// as this often times becomes unreadable.
func Requeue(request Request) (ctrl.Result, error) {
	return RequeueAfter(request.DefaultRequeue()), nil
}

// NoRequeueOnError returns a non-requeue result with an error.  It is a helper
// function to reduce the amount of times you non-requeue with errors in a controller
// as this often times becomes unreadable.
func NoRequeueOnError(err error) (ctrl.Result, error) {
	return NoRequeue(), err
}

// NoRequeueWithoutError returns a non-requeue result without an error.  It is a helper
// function to reduce the amount of times you non-requeue without errors in a controller
// as this often times becomes unreadable.
func NoRequeueWithoutError() (ctrl.Result, error) {
	return NoRequeue(), nil
}

// CustomRequeueOnError returns a requeue result with an error.
func CustomRequeueOnError(requeue time.Duration, err error) (ctrl.Result, error) {
	return RequeueAfter(requeue), err
}

// ReconcileEnd is a helper function that returns a result at the end of a reconciliation
// process.
func ReconcileEnd(request Request) (ctrl.Result, error) {
	return Requeue(request)
}

// ReconcileContinue is a helper function that returns a result to continue the
// reconciliation process by entering the next phase.
func ReconcileContinue() (ctrl.Result, error) {
	return NoRequeueWithoutError()
}

// ReconcileStop is a helper function that returns a result to stop the
// reconciliation process.
func ReconcileStop() (ctrl.Result, error) {
	return NoRequeueWithoutError()
}
