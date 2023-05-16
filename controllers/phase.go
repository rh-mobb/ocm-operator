package controllers

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Phase defines an individual phase in the controller reconciliation process.
type Phase struct {
	Name     string
	Function func() (ctrl.Result, error)
}

// Execute executes the phases of controller reconciliation.
func Execute(request Request, req reconcile.Request, phases ...Phase) (ctrl.Result, error) {
	for execute := range phases {
		// run each phase function and return if we receive any errors
		result, err := phases[execute].Function()
		if err != nil || result.Requeue {
			return result, ReconcileError(
				req,
				fmt.Sprintf("%s phase reconciliation error", phases[execute].Name),
				err,
			)
		}
	}

	return NoRequeue(), nil
}
