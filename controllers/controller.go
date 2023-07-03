package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/scottd018/go-utils/pkg/list"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

var (
	ErrConvertClientObject = errors.New("unable to convert to client object")
)

const (
	defaultFinalizerSuffix = "finalizer"
	defaultRequeue         = 30 * time.Second

	LogLevelDebug = 5
)

// Controller represents the object that is performing the reconciliation
// action.
type Controller interface {
	kubernetes.Client

	NewRequest(ctx context.Context, req ctrl.Request) (Request, error)
	Reconcile(context.Context, ctrl.Request) (ctrl.Result, error)
	ReconcileCreate(Request) (ctrl.Result, error)
	ReconcileUpdate(Request) (ctrl.Result, error)
	ReconcileDelete(Request) (ctrl.Result, error)
	SetupWithManager(mgr ctrl.Manager) error
}

// Access to create and patch events are needed so the operator can create events and register
// them with the custom resources.

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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

	// set a condition notifying the resource that we are reconciling
	if err := conditions.Update(ctx, controller, request.GetObject(), conditions.Reconciling(trigger)); err != nil {
		return RequeueAfter(defaultRequeue), fmt.Errorf("unable to update condition - %w", err)
	}

	// run the reconciliation loop based on the event trigger
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

// FinalizerName returns the finalizer name for the controller.
func FinalizerName(object client.Object) string {
	return strings.ToLower(fmt.Sprintf("%s.%s/%s",
		object.GetObjectKind().GroupVersionKind().Kind,
		object.GetObjectKind().GroupVersionKind().Group,
		defaultFinalizerSuffix,
	))
}

// HasFinalizer is a helper function to make the code more readable.
func HasFinalizer(object client.Object) bool {
	return list.Strings(object.GetFinalizers()).Has(FinalizerName(object))
}

// AddFinalizer adds finalizers to the object so that the delete lifecycle can be run
// before the object is deleted.
func AddFinalizer(ctx context.Context, r kubernetes.Client, object client.Object) error {
	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if !HasFinalizer(object) {
		original, ok := object.DeepCopyObject().(client.Object)
		if !ok {
			return ErrConvertClientObject
		}

		controllerutil.AddFinalizer(object, FinalizerName(object))

		if err := r.Patch(ctx, object, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("unable to add finalizer - %w", err)
		}
	}

	return nil
}

// RemoveFinalizer removes finalizers from the object.  It is intended to be run after an
// external object is deleted so that the delete lifecycle may continue reconciliation.
func RemoveFinalizer(ctx context.Context, r kubernetes.Client, object client.Object) error {
	if HasFinalizer(object) {
		original, ok := object.DeepCopyObject().(client.Object)
		if !ok {
			return ErrConvertClientObject
		}

		controllerutil.RemoveFinalizer(object, FinalizerName(object))

		if err := r.Patch(ctx, object, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("unable to remove finalizer - %w", err)
		}
	}

	return nil
}
