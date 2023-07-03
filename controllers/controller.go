package controllers

import (
	"context"
	"errors"
	"fmt"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

var (
	ErrConvertClientObject = errors.New("unable to convert to client object")
)

const (
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
