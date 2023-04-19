/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	defaultMachinePoolRequeue time.Duration = 30
)

var (
	ErrMachinePoolReservedLabel = errors.New(
		fmt.Sprintf(
			"problem with system reserved labels: %s, %s",
			ocm.LabelPrefixManaged,
			ocm.LabelPrefixName,
		),
	)
)

// MachinePoolReconciler reconciles a MachinePool object
type MachinePoolReconciler struct {
	client.Client

	Context    context.Context
	Scheme     *runtime.Scheme
	Connection *sdk.Connection
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MachinePool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *MachinePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Context = ctx

	// create the request
	request, err := NewRequest(r, ctx, req)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return noRequeue(), fmt.Errorf("unable to create request - %w", err)
		}

		return noRequeue(), nil
	}

	// register the delete hooks
	if err := r.RegisterDeleteHooks(&request); err != nil {
		return noRequeue(), reconcilerError(
			request.ControllerRequest,
			"unable to register delete hooks",
			err,
		)
	}

	// run the reconciliation loop
	switch request.Trigger {
	case triggerCreate:
		return r.ReconcileCreateOrUpdate(&request)
	case triggerUpdate:
		return r.ReconcileCreateOrUpdate(&request)
	case triggerDelete:
		return r.ReconcileDelete(&request)
	default:
		return noRequeue(), reconcilerError(
			request.ControllerRequest,
			"unable to determine controller trigger",
			ErrTriggerUnknown,
		)
	}
}

func (r *MachinePoolReconciler) ReconcileCreateOrUpdate(request *MachinePoolRequest) (ctrl.Result, error) {
	// run through each phase of controller reconciliation
	for _, phase := range []MachinePoolPhaseFunc{
		r.Begin,
		r.GetDesiredState,
		r.GetCurrentState,
		r.CreateOrUpdate,
		r.WaitUntilReady,
		r.Complete,
	} {
		// run each phase function and return if we receive any errors
		result, err := phase(request)
		if err != nil || result.Requeue {
			return result, reconcilerError(
				request.ControllerRequest,
				"phase reconciliation error in create",
				err,
			)
		}
	}

	return noRequeue(), nil
}

func (r *MachinePoolReconciler) ReconcileDelete(request *MachinePoolRequest) (ctrl.Result, error) {
	return noRequeue(), nil
}

// RegisterDeleteHooks adds finializers to the machine pool so that the delete lifecycle can be run
// before the object is deleted.
func (r *MachinePoolReconciler) RegisterDeleteHooks(request *MachinePoolRequest) error {
	if request.Desired.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(request.Desired.GetFinalizers(), finalizerName(request.Desired)) {
			controllerutil.AddFinalizer(request.Desired, finalizerName(request.Desired))

			if err := r.Update(request.Context, request.Desired); err != nil {
				return fmt.Errorf("unable to register delete hook - %w", err)
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachinePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ocmv1alpha1.MachinePool{}).
		Complete(r)
}
