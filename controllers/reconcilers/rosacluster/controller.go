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
	"fmt"
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
	"github.com/rh-mobb/ocm-operator/pkg/aws"
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
	Log        logr.Logger

	AWSClient *aws.Client
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=rosaclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=rosaclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=rosaclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, ctrlReq ctrl.Request) (ctrl.Result, error) {
	return controllers.Reconcile(ctx, r, ctrlReq)
}

// ReconcileCreate performs the reconciliation logic when a create event triggered
// the reconciliation.
func (r *Controller) ReconcileCreate(reconcileRequest request.Request) (ctrl.Result, error) {
	// run setup
	req, err := r.Setup(reconcileRequest)
	if err != nil {
		return requeue.OnError(req, fmt.Errorf("error executing setup method - %w", err))
	}

	// add the finalizer
	if err := controllers.AddFinalizer(req.Context, r, req.Original); err != nil {
		return requeue.OnError(req, controllers.AddFinalizerError(err))
	}

	// execute the phases
	return phases.NewHandler(req,
		phases.NewPhase("GetCurrentState", func() (ctrl.Result, error) { return r.GetCurrentState(req) }),
		phases.NewPhase("ApplyCluster", func() (ctrl.Result, error) { return r.ApplyCluster(req) }),
		phases.NewPhase("WaitUntilReady", func() (ctrl.Result, error) { return r.WaitUntilReady(req) }),
		phases.NewPhase("Complete", func() (ctrl.Result, error) { return phases.Complete(req, triggers.Create, r.Log) }),
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
	// run setup
	req, err := r.Setup(reconcileRequest)
	if err != nil {
		return requeue.OnError(req, fmt.Errorf("error executing setup method - %w", err))
	}

	// execute the phases
	return phases.NewHandler(req,
		phases.NewPhase("FindChildObjects", func() (ctrl.Result, error) { return r.FindChildObjects(req) }),
		phases.NewPhase("DestroyCluster", func() (ctrl.Result, error) { return r.DestroyCluster(req) }),
		phases.NewPhase("WaitUntilMissing", func() (ctrl.Result, error) { return r.WaitUntilMissing(req) }),
		phases.NewPhase("DestroyOperatorRoles", func() (ctrl.Result, error) { return r.DestroyOperatorRoles(req) }),
		phases.NewPhase("DestroyOIDC", func() (ctrl.Result, error) { return r.DestroyOIDC(req) }),
		phases.NewPhase("CompleteDestroy", func() (ctrl.Result, error) { return phases.CompleteDestroy(req, r.Log) }),
	).Execute()
}

// Setup runs the reconciliation process prior to executing the individual
// reconciliation phases.  It returns the request needed for the reconciliation
// process.
func (r *Controller) Setup(reconcileRequest request.Request) (*ROSAClusterRequest, error) {
	// type cast the req to a rosa cluster req
	req, ok := reconcileRequest.(*ROSAClusterRequest)
	if !ok {
		return &ROSAClusterRequest{}, request.TypeConvertError(&ROSAClusterRequest{})
	}

	// create the aws client used for interacting with aws services if it
	// is already not set.  this is set prior to reconciliation to avoid
	// having to create the client multiple times for each reconcile
	// request.
	if r.AWSClient == nil {
		awsClient, err := aws.NewClient(req.Desired.Spec.Region)
		if err != nil {
			return req, fmt.Errorf("unable to create aws client - %w", err)
		}

		r.AWSClient = awsClient
	}

	return req, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(workload.Predicates()).
		For(&ocmv1alpha1.ROSACluster{}).
		Complete(r)
}
