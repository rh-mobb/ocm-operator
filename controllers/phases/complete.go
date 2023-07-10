package phases

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"

	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
)

// Complete will perform all actions required to successful complete a reconciliation request.  It will
// requeue after the interval value requested by the controller configuration to ensure that the
// object remains in its desired state at a specific interval.
func Complete(req request.Request, trigger triggers.Trigger, log logr.Logger) (ctrl.Result, error) {
	if err := conditions.Update(req, conditions.Reconciled(trigger)); err != nil {
		return requeue.OnError(req, conditions.UpdateReconcilingConditionError(err))
	}

	log.Info("completed object reconciliation", request.LogValues(req)...)
	log.Info(fmt.Sprintf("reconciling again in %s", req.DefaultRequeue()), request.LogValues(req)...)

	// requeue the reconciliation based on the default controller reconciliation value
	return requeue.OnSuccess(req)
}

// CompleteDestroy will perform all actions required to successfully complete a delete reconciliation req.
func CompleteDestroy(req request.Request, log logr.Logger) (ctrl.Result, error) {
	if err := controllers.RemoveFinalizer(req.GetContext(), req.GetReconciler(), req.GetObject()); err != nil {
		return requeue.OnError(req, controllers.RemoveFinalizerError(err))
	}

	log.Info("completed object deletion", request.LogValues(req)...)

	// do not requeue since the object is now deleted
	return requeue.None()
}
