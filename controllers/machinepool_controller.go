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

	"github.com/nukleros/operator-builder-tools/pkg/controller/predicates"
	sdk "github.com/openshift-online/ocm-sdk-go"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/utils"
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

	Scheme     *runtime.Scheme
	Connection *sdk.Connection
	Recorder   record.EventRecorder
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MachinePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// create the request
	request, err := NewRequest(r, ctx, req)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return noRequeue(), fmt.Errorf("unable to create request - %w", err)
		}

		return noRequeue(), nil
	}

	// run the reconciliation loop based on the type of request
	switch request.Trigger {
	case triggers.Create:
		return r.ReconcileCreateOrUpdate(&request)
	case triggers.Update:
		return r.ReconcileCreateOrUpdate(&request)
	case triggers.Delete:
		return r.ReconcileDelete(&request)
	default:
		return noRequeue(), reconcilerError(
			request.ControllerRequest,
			"unable to determine controller trigger",
			triggers.ErrTriggerUnknown,
		)
	}
}

func (r *MachinePoolReconciler) ReconcileCreateOrUpdate(request *MachinePoolRequest) (ctrl.Result, error) {
	// register the delete hooks
	if err := r.RegisterDeleteHooks(request); err != nil {
		return noRequeue(), fmt.Errorf("unable to register delete hooks - %w", err)
	}

	// run through each phase of controller reconciliation
	for name, execute := range map[string]PhaseFunction{
		"begin":           r.Begin,
		"getCurrentState": r.GetCurrentState,
		"applyState":      r.Apply,
		"waitUntilReady":  r.WaitUntilReady,
		"complete":        r.Complete,
	} {
		// run each phase function and return if we receive any errors
		result, err := execute(request)
		if err != nil || result.Requeue {
			return result, reconcilerError(
				request.ControllerRequest,
				fmt.Sprintf("%s phase reconciliation error in create or update", name),
				err,
			)
		}
	}

	return noRequeue(), nil
}

func (r *MachinePoolReconciler) ReconcileDelete(request *MachinePoolRequest) (ctrl.Result, error) {
	// run through each phase of controller reconciliation
	for name, execute := range map[string]PhaseFunction{
		"begin":            r.Begin,
		"destroy":          r.Destroy,
		"waitUntilMissing": r.WaitUntilMissing,
		"complete":         r.CompleteDestroy,
	} {
		// run each phase function and return if we receive any errors
		result, err := execute(request)
		if err != nil || result.Requeue {
			return result, reconcilerError(
				request.ControllerRequest,
				fmt.Sprintf("%s phase reconciliation error in delete", name),
				err,
			)
		}
	}

	return noRequeue(), nil
}

// RegisterDeleteHooks adds finializers to the machine pool so that the delete lifecycle can be run
// before the object is deleted.
func (r *MachinePoolReconciler) RegisterDeleteHooks(request *MachinePoolRequest) error {
	if request.Original.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !utils.ContainsString(request.Original.GetFinalizers(), finalizerName(request.Original)) {
			original := request.Original.DeepCopy()

			controllerutil.AddFinalizer(request.Original, finalizerName(request.Original))

			if err := r.Patch(request.Context, request.Original, client.MergeFrom(original)); err != nil {
				return fmt.Errorf("unable to register delete hook - %w", err)
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachinePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicates.WorkloadPredicates()).
		For(&ocmv1alpha1.MachinePool{}).
		Complete(r)
}
