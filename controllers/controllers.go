package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

const (
	defaultFinalizerSuffix = "finalizer"

	LogLevelDebug = 5
)

// Request represents a request that was sent to the controller that
// caused reconciliation.  It is used to track the status during the steps of
// controller reconciliation and pass information.  It should be able to
// return back the original object, in its pure form, that was discovered
// when the request was triggered.
type Request interface {
	GetObject() Workload
}

// Workload represents the actual object that is being reconciled.
type Workload interface {
	client.Object

	GetConditions() []metav1.Condition
	SetConditions([]metav1.Condition)
}

// Controller represents the object that is performing the reconciliation
// action.
type Controller interface {
	NewRequest(ctx context.Context, req ctrl.Request) (Request, error)
	Reconcile(context.Context, ctrl.Request) (ctrl.Result, error)
	ReconcileCreate(Request) (ctrl.Result, error)
	ReconcileUpdate(Request) (ctrl.Result, error)
	ReconcileDelete(Request) (ctrl.Result, error)
	SetupWithManager(mgr ctrl.Manager) error
}

// Reconcile is a centralized, reusable reconciliation loop by which all controllers can
// use as their reconciliation function.  It requires that a new request for each reconciliation
// loop is created to track that status throughout each request.
func Reconcile(ctx context.Context, controller Controller, req ctrl.Request) (ctrl.Result, error) {
	// create the request
	request, err := controller.NewRequest(ctx, req)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return NoRequeue(), fmt.Errorf("unable to create request - %w", err)
		}

		return NoRequeue(), nil
	}

	// determine what triggered the reconcile request
	trigger := triggers.GetTrigger(request.GetObject())

	// run the reconciliation loop based on the event trigger
	//nolint:wrapcheck
	switch trigger.String() {
	case triggers.CreateString:
		return controller.ReconcileCreate(request)
	case triggers.UpdateString:
		return controller.ReconcileUpdate(request)
	case triggers.DeleteString:
		return controller.ReconcileDelete(request)
	default:
		return NoRequeue(), ReconcileError(
			req,
			"unable to determine controller trigger",
			triggers.ErrTriggerUnknown,
		)
	}
}

// RequeueAfter returns a requeue result to requeue after a specific
// number of seconds.
func RequeueAfter(seconds time.Duration) ctrl.Result {
	return ctrl.Result{Requeue: true, RequeueAfter: seconds}
}

// NoRequeue returns a blank result to prevent a requeue.
func NoRequeue() ctrl.Result {
	return ctrl.Result{}
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

// FinalizerName returns the finalizer name for the controller.
func FinalizerName(object client.Object) string {
	return strings.ToLower(fmt.Sprintf("%s.%s/%s",
		object.GetObjectKind().GroupVersionKind().Kind,
		object.GetObjectKind().GroupVersionKind().Group,
		defaultFinalizerSuffix,
	))
}
