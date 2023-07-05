package phases

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
)

// handler represents an object that handles individual phases.
type handler struct {
	Phases  []phase
	Request request.Request
}

// NewHandler returns a new instance of a Handler.
func NewHandler(req request.Request, phases ...phase) *handler {
	return &handler{
		Phases:  phases,
		Request: req,
	}
}

// Execute executes the phases for a handler.
func (handler *handler) Execute() (ctrl.Result, error) {
	for execute := range handler.Phases {
		// run each phase function and return if we receive any errors
		result, err := handler.Phases[execute].Function()
		if err != nil || result.Requeue {
			return requeue.OnError(handler.Request, request.Error(handler.Request, fmt.Errorf(
				"error in phase [%s] - %w",
				handler.Phases[execute].Name,
				err,
			)))
		}
	}

	return requeue.Skip(nil)
}
