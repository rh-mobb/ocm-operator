package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
)

// MachinePoolRequest is an object that is unique to each reconciliation
// request.
// TODO: make this a generic request to be used across a variety of controllers
type MachinePoolRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.MachinePool
	Original          *ocmv1alpha1.MachinePool
	Desired           *ocmv1alpha1.MachinePool
	Log               logr.Logger
	Trigger           controllerTrigger
}

func NewRequest(r *MachinePoolReconciler, ctx context.Context, req ctrl.Request) (MachinePoolRequest, error) {
	original := &ocmv1alpha1.MachinePool{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return MachinePoolRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return MachinePoolRequest{}, err
	}

	// ensure the our managed labels do not conflict with what was submitted
	// to the cluster
	//
	// NOTE: this is implemented via CRD CEL validations, however leaving in
	// place for clusters that may not have this feature gate enabled as CEL
	// is in beta currently.
	if original.HasManagedLabels() {
		return MachinePoolRequest{}, fmt.Errorf(
			"spec.labels cannot contain reserved labels [%+v] - %w",
			original.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	return MachinePoolRequest{
		Original:          original,
		Desired:           original.DesiredState(),
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.FromContext(ctx),
		Trigger:           trigger(original),
	}, nil
}

func (request *MachinePoolRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
	)
}
