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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nukleros/operator-builder-tools/pkg/controller/predicates"
	sdk "github.com/openshift-online/ocm-sdk-go"
	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
)

// Controller reconciles a LDAPIdentityProvider object
type Controller struct {
	client.Client

	Scheme     *runtime.Scheme
	Connection *sdk.Connection
	Recorder   record.EventRecorder
	Interval   time.Duration
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=ldapidentityproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=ldapidentityproviders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=ldapidentityproviders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
//nolint:wrapcheck
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return controllers.Reconcile(ctx, r, req)
}

// ReconcileCreate performs the reconciliation logic when a create event triggered
// the reconciliation.
func (r *Controller) ReconcileCreate(req controllers.Request) (ctrl.Result, error) {
	// type cast the request to a ldap identity provider request
	request, ok := req.(*LDAPIdentityProviderRequest)
	if !ok {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), ErrLDAPIdentityProviderRequestConvert
	}

	// add the finalizer
	if err := controllers.AddFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf("unable to register delete hooks - %w", err)
	}

	// execute the phases
	return request.execute([]Phase{
		{Name: "begin", Function: r.Begin},
		{Name: "getCurrentState", Function: r.GetCurrentState},
		{Name: "applyOCM", Function: r.ApplyIdentityProvider},
		{Name: "complete", Function: r.Complete},
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
	// type cast the request to a ldap identity provider request
	request, ok := req.(*LDAPIdentityProviderRequest)
	if !ok {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), ErrLDAPIdentityProviderRequestConvert
	}

	// execute the phases
	return request.execute([]Phase{
		{Name: "begin", Function: r.Begin},
		// {Name: "destroy", Function: r.Destroy},
		// {Name: "complete", Function: r.Complete},
	}...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicates.WorkloadPredicates()).
		For(&ocmv1alpha1.LDAPIdentityProvider{}).
		Complete(r)
}
