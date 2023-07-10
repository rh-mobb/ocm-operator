package phases

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/requeue"
)

// NewPhase returns a new instance of a phase.
func NewPhase(name string, f func() (ctrl.Result, error)) phase {
	return phase{
		Name:     name,
		Function: f,
	}
}

// Phase defines an individual phase in the controller reconciliation process.
type phase struct {
	Name     string
	Function func() (ctrl.Result, error)
}

// Next is a helper function for code readability to proceed to the next phase.
func Next() (ctrl.Result, error) {
	return requeue.Skip(nil)
}
