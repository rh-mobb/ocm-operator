package requeue

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/request"
)

// After returns a requeue result to requeue after a specific
// number of seconds.
func After(seconds time.Duration, err error) (ctrl.Result, error) {
	return ctrl.Result{Requeue: true, RequeueAfter: seconds}, err
}

// Skip returns a blank result to prevent a requeue.
func Skip(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

// None returns a result indicating no requeue is needed.
func None() (ctrl.Result, error) {
	return Skip(nil)
}

// Retry is a helper function to retry a requeue.
func Retry(req request.Request) (ctrl.Result, error) {
	return OnSuccess(req)
}

// OnError returns a requeue result with an error.  It is a helper
// function to reduce the amount of times you requeue with errors in a controller
// as this often times becomes unreadable.
func OnError(req request.Request, err error) (ctrl.Result, error) {
	return After(req.DefaultRequeue(), err)
}

// OnSuccess returns a requeue result without an error.  It is a helper
// function to reduce the amount of times you requeue with errors in a controller
// as this often times becomes unreadable.
func OnSuccess(req request.Request) (ctrl.Result, error) {
	return After(req.DefaultRequeue(), nil)
}
