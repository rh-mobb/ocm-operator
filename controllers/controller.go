package controllers

import (
	"context"
	"fmt"
	"time"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

const (
	defaultRequeue = 30 * time.Second

	LogLevelDebug = 5
)

// Controller represents the object that is performing the reconciliation
// action.
type Controller interface {
	kubernetes.Client

	NewRequest(ctx context.Context, req ctrl.Request) (request.Request, error)
	Reconcile(context.Context, ctrl.Request) (ctrl.Result, error)
	ReconcileCreate(request.Request) (ctrl.Result, error)
	ReconcileUpdate(request.Request) (ctrl.Result, error)
	ReconcileDelete(request.Request) (ctrl.Result, error)
	SetupWithManager(mgr ctrl.Manager) error
}

// Access to create and patch events are needed so the operator can create events and register
// them with the custom resources.

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is a centralized, reusable reconciliation loop by which all controllers can
// use as their reconciliation function.  It requires that a new request for each reconciliation
// loop is created to track that status throughout each request.
func Reconcile(ctx context.Context, controller Controller, ctrlReq ctrl.Request) (ctrl.Result, error) {
	// create the reconcile request
	req, err := controller.NewRequest(ctx, ctrlReq)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return requeue.Skip(fmt.Errorf("unable to create request - %w", err))
		}

		return requeue.Skip(nil)
	}

	// determine what triggered the reconcile request
	trigger := triggers.GetTrigger(req.GetObject())

	// set a condition notifying the resource that we are reconciling
	if err := conditions.Update(req, conditions.Reconciling(trigger)); err != nil {
		return requeue.After(defaultRequeue, conditions.UpdateReconcilingConditionError(err))
	}

	// run the reconciliation loop based on the event trigger
	switch trigger.String() {
	case triggers.CreateString:
		return controller.ReconcileCreate(req)
	case triggers.UpdateString:
		return controller.ReconcileUpdate(req)
	case triggers.DeleteString:
		return controller.ReconcileDelete(req)
	default:
		return requeue.Skip(request.Error(req, triggers.ErrTriggerUnknown))
	}
}
