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

package rosacluster

import (
	"context"
	"errors"
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

var (
	ErrMissingClusterID      = errors.New("unable to find cluster id")
	ErrClusterRequestConvert = errors.New("unable to convert generic request to cluster request")
)

const (
	defaultClusterRequeue                     = 30 * time.Second
	defaultClusterRequeueHostedPostProvision  = 60 * time.Second
	defaultClusterRequeueClassicPostProvision = 300 * time.Second
)

// Controller reconciles a Cluster object.
type Controller struct {
	client.Client

	Scheme     *runtime.Scheme
	Connection *sdk.Connection
	Recorder   record.EventRecorder
	Interval   time.Duration
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=rosaclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=rosaclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=rosaclusters/finalizers,verbs=update

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
	request, ok := req.(*ROSAClusterRequest)
	if !ok {
		return controllers.RequeueAfter(defaultClusterRequeue), ErrClusterRequestConvert
	}

	// add the finalizer
	if err := controllers.AddFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("unable to register delete hooks - %w", err)
	}

	// execute the phases
	return request.execute([]Phase{
		{Name: "begin", Function: r.Begin},
		{Name: "getCurrentState", Function: r.GetCurrentState},
		{Name: "applyOCM", Function: r.ApplyCluster},
		{Name: "waitUntilReady", Function: r.WaitUntilReady},
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
	// type cast the request to an ldap identity provider request
	request, ok := req.(*ROSAClusterRequest)
	if !ok {
		return controllers.RequeueAfter(defaultClusterRequeue), ErrClusterRequestConvert
	}

	// execute the phases
	return request.execute([]Phase{
		{Name: "begin", Function: r.Begin},
		{Name: "destroy", Function: r.DestroyCluster},
		{Name: "destroy", Function: r.WaitUntilMissing},
		{Name: "destroy", Function: r.DestroyOperatorRoles},
		{Name: "destroy", Function: r.DestroyOIDC},
		{Name: "complete", Function: r.CompleteDestroy},
	}...)
}

// SetupWithManager sets up the controller with the Manager.
//
//nolint:wrapcheck
func (r *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicates.WorkloadPredicates()).
		For(&ocmv1alpha1.ROSACluster{}).
		Complete(r)
}
