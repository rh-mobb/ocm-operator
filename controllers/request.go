package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MachinePoolRequest is an object that is unique to each reconciliation
// request.
// TODO: make this a generic request to be used across a variety of controllers
type MachinePoolRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.MachinePool
	Desired           *ocmv1alpha1.MachinePool
	Client            *ocm.MachinePoolClient
	Log               logr.Logger
	Trigger           controllerTrigger
	NodesReady        bool
}

func NewRequest(r *MachinePoolReconciler, ctx context.Context, req ctrl.Request) (MachinePoolRequest, error) {
	desiredState := &ocmv1alpha1.MachinePool{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, req.NamespacedName, desiredState); err != nil {
		if !apierrs.IsNotFound(err) {
			return MachinePoolRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return MachinePoolRequest{}, err
	}

	return MachinePoolRequest{
		Current:           &ocmv1alpha1.MachinePool{},
		Desired:           desiredState,
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.FromContext(ctx),
		Trigger:           trigger(desiredState),
	}, nil
}

func (request *MachinePoolRequest) hasMachinePool() bool {
	if request.Client == nil {
		return false
	}

	if request.Client.MachinePool == nil {
		return false
	}

	return (request.Client.MachinePool.Object != nil)
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
