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

package gitlabidentityprovider

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nukleros/operator-builder-tools/pkg/controller/predicates"
	sdk "github.com/openshift-online/ocm-sdk-go"
	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/utils"
)

const (
	defaultGitLabIdentityProviderRequeue = 30 * time.Second
)

// Controller reconciles a GitLabIdentityProvider object
type Controller struct {
	client.Client

	Scheme     *runtime.Scheme
	Connection *sdk.Connection
	Recorder   record.EventRecorder
	Interval   time.Duration
}

//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=gitlabidentityproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=gitlabidentityproviders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocm.mobb.redhat.com,resources=gitlabidentityproviders/finalizers,verbs=update

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
	// type cast the request to a machine pool request
	request, ok := req.(*GitLabIdentityProviderRequest)
	if !ok {
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), ErrGitLabIdentityProviderRequestConvert
	}

	// register the delete hooks
	if err := r.RegisterDeleteHooks(request); err != nil {
		return controllers.NoRequeue(), fmt.Errorf("unable to register delete hooks - %w", err)
	}

	// execute the phases
	return request.execute([]Phase{
		{Name: "begin", Function: r.Begin},
		{Name: "getCurrentState", Function: r.GetCurrentState},
		{Name: "applyGitLab", Function: r.ApplyGitLab},
		{Name: "applyIdentityProvider", Function: r.ApplyIdentityProvider},
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
	// type cast the request to a machine pool request
	request, ok := req.(*GitLabIdentityProviderRequest)
	if !ok {
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), ErrGitLabIdentityProviderRequestConvert
	}

	// execute the phases
	return request.execute([]Phase{
		{Name: "begin", Function: r.Begin},
		{Name: "destroy", Function: r.Destroy},
		{Name: "complete", Function: r.Complete},
	}...)
}

// RegisterDeleteHooks adds finializers to the machine pool so that the delete lifecycle can be run
// before the object is deleted.
// TODO: centralize this into controllers package.
func (r *Controller) RegisterDeleteHooks(request *GitLabIdentityProviderRequest) error {
	if request.Original.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !utils.ContainsString(request.Original.GetFinalizers(), controllers.FinalizerName(request.Original)) {
			original := request.Original.DeepCopy()

			controllerutil.AddFinalizer(request.Original, controllers.FinalizerName(request.Original))

			if err := r.Patch(request.Context, request.Original, client.MergeFrom(original)); err != nil {
				return fmt.Errorf("unable to register delete hook - %w", err)
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicates.WorkloadPredicates()).
		For(&ocmv1alpha1.GitLabIdentityProvider{}).
		Complete(r)
}
