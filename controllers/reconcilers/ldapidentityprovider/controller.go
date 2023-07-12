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

package ldapidentityprovider

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/phases"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	defaultLDAPIdentityProviderRequeue = 30 * time.Second
)

// Controller reconciles a LDAPIdentityProvider object.
type Controller struct {
	client.Client

	Scheme     *runtime.Scheme
	Connection *sdk.Connection
	Recorder   record.EventRecorder
	Interval   time.Duration
	Logger     logr.Logger
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=ldapidentityproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=ldapidentityproviders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=ldapidentityproviders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, ctrlReq ctrl.Request) (ctrl.Result, error) {
	return controllers.Reconcile(ctx, r, ctrlReq)
}

// ReconcileCreate performs the reconciliation logic when a create event triggered
// the reconciliation.
func (r *Controller) ReconcileCreate(reconcileRequest request.Request) (ctrl.Result, error) {
	// type cast the request to an ldap identity provider request
	req, ok := reconcileRequest.(*LDAPIdentityProviderRequest)
	if !ok {
		return requeue.OnError(req, request.TypeConvertError(&LDAPIdentityProviderRequest{}))
	}

	// add the finalizer
	if err := controllers.AddFinalizer(req.Context, r, req.Original); err != nil {
		return requeue.OnError(req, controllers.AddFinalizerError(err))
	}

	// execute the phases
	return phases.NewHandler(req,
		phases.NewPhase("HandleUpstreamCluster", func() (ctrl.Result, error) {
			return phases.HandleClusterPhase(
				req,
				ocm.NewClusterClient(req.Reconciler.Connection, req.GetClusterName()),
				triggers.Create,
				r.Logger,
			)
		}),
		phases.NewPhase("GetCurrentState", func() (ctrl.Result, error) { return r.GetCurrentState(req) }),
		phases.NewPhase("ApplyIdentityProvider", func() (ctrl.Result, error) { return r.ApplyIdentityProvider(req) }),
		phases.NewPhase("Complete", func() (ctrl.Result, error) { return phases.Complete(req, triggers.Create, r) }),
	).Execute()
}

// ReconcileUpdate performs the reconciliation logic when an update event triggered
// the reconciliation.  In this instance, create and update share identical logic
// so we are simply calling the ReconcileCreate method.
func (r *Controller) ReconcileUpdate(reconcileRequest request.Request) (ctrl.Result, error) {
	return r.ReconcileCreate(reconcileRequest)
}

// ReconcileDelete performs the reconciliation logic when a delete event triggered
// the reconciliation.
func (r *Controller) ReconcileDelete(reconcileRequest request.Request) (ctrl.Result, error) {
	// type cast the request to a ldap identity provider req
	req, ok := reconcileRequest.(*LDAPIdentityProviderRequest)
	if !ok {
		return requeue.OnError(req, request.TypeConvertError(&LDAPIdentityProviderRequest{}))
	}

	// execute the phases
	return phases.NewHandler(req,
		phases.NewPhase("Destroy", func() (ctrl.Result, error) { return r.Destroy(req) }),
		phases.NewPhase("CompleteDestroy", func() (ctrl.Result, error) { return phases.CompleteDestroy(req, r) }),
	).Execute()
}

// ReconcileInterval returns the requeue interval for the controller.  It is used to
// satisfy the Controller interface.
func (r *Controller) ReconcileInterval() time.Duration {
	return r.Interval
}

// Log returns the controller logger.  It is used to satisfy the Controller interface.
func (r *Controller) Log() logr.Logger {
	return r.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (r *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(workload.Predicates()).
		For(&ocmv1alpha1.LDAPIdentityProvider{}).
		Complete(r)
}
