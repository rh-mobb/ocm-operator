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

package machinepool

import (
	"context"
	"fmt"
	"time"

	"github.com/nukleros/operator-builder-tools/pkg/controller/predicates"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
)

const (
	defaultMachinePoolRequeue = 30 * time.Second
)

// Controller reconciles a MachinePool object.
type Controller struct {
	client.Client

	Scheme     *runtime.Scheme
	Connection *sdk.Connection
	Recorder   record.EventRecorder
	Interval   time.Duration
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=machinepools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return controllers.Reconcile(ctx, r, req)
}

// ReconcileCreate performs the reconciliation logic when a create event triggered
// the reconciliation.
func (r *Controller) ReconcileCreate(req controllers.Request) (ctrl.Result, error) {
	// type cast the request to a machine pool request
	request, ok := req.(*MachinePoolRequest)
	if !ok {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), ErrMachinePoolRequestConvert
	}

	// add the finalizer
	if err := controllers.AddFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to register delete hooks - %w", err)
	}

	// execute the phases
	return controllers.Execute(request, request.ControllerRequest, []controllers.Phase{
		{Name: "GetCurrentState", Function: func() (ctrl.Result, error) { return r.GetCurrentState(request) }},
		{Name: "Apply", Function: func() (ctrl.Result, error) { return r.Apply(request) }},
		{Name: "WaitUntilReady", Function: func() (ctrl.Result, error) { return r.WaitUntilReady(request) }},
		{Name: "Complete", Function: func() (ctrl.Result, error) { return r.Complete(request) }},
	}...)
}

// ReconcileUpdate performs the reconciliation logic when an update event triggered
// the reconciliation.  In this instance, create and update share identical logic
// so we are simply calling the ReconcileCreate method.
func (r *Controller) ReconcileUpdate(req controllers.Request) (ctrl.Result, error) {
	return r.ReconcileCreate(req)
}

// ReconcileDelete performs the reconciliation logic when a delete event triggered
// the reconciliation.
func (r *Controller) ReconcileDelete(req controllers.Request) (ctrl.Result, error) {
	// type cast the request to a machine pool request
	request, ok := req.(*MachinePoolRequest)
	if !ok {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), ErrMachinePoolRequestConvert
	}

	// execute the phases
	return controllers.Execute(request, request.ControllerRequest, []controllers.Phase{
		{Name: "Destroy", Function: func() (ctrl.Result, error) { return r.Destroy(request) }},
		{Name: "WaitUntilMissing", Function: func() (ctrl.Result, error) { return r.WaitUntilMissing(request) }},
		{Name: "CompleteDestroy", Function: func() (ctrl.Result, error) { return r.CompleteDestroy(request) }},
	}...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicates.WorkloadPredicates()).
		For(&ocmv1alpha1.MachinePool{}).
		Complete(r)
}
