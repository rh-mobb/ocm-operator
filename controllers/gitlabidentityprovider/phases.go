package gitlabidentityprovider

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/events"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// GetCurrentState gets the current state of the GitLabIdentityProvider resource.  The current state of the GitLabIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// get the generic identity provider object from ocm
	request.OCMClient = ocm.NewIdentityProviderClient(
		request.Reconciler.Connection,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	idp, err := request.OCMClient.Get()
	if err != nil {
		return controllers.RequeueOnError(request, controllers.GetOCMError(request, err))
	}

	// return if there is no identity provider found
	if idp == nil {
		return controllers.NoRequeue(), nil
	}

	// store the current state
	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// request.Current.Spec.AccessTokenSecret = request.Desired.Spec.AccessTokenSecret
	request.Current = &ocmv1alpha1.GitLabIdentityProvider{}
	request.Current.Spec.ClusterName = request.Desired.Spec.ClusterName
	request.Current.Spec.DisplayName = request.Desired.Spec.DisplayName
	request.Current.Spec.ClientSecret.Name = request.Desired.Spec.ClientSecret.Name
	request.Current.Spec.CA.Name = request.Desired.Spec.CA.Name
	request.Current.Spec.MappingMethod = string(idp.MappingMethod())
	request.Current.CopyFrom(idp)

	return controllers.NoRequeue(), nil
}

// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
// // ApplyGitLab applies the state to a GitLab instance.  This includes creating and/or updating an application
// // with the appropriate oauth URL from OpenShift.
// func (r *Controller) ApplyGitLab(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
// 	// get the gitlab application from gitlab, using the display name as the name of the
// 	// application to search for
// 	application, err := request.GitLabClient.GetApplication(request.Desired.Spec.DisplayName)
// 	if err != nil {
// 		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
// 			"unable to retrieve application from gitlab - %w",
// 			err,
// 		)
// 	}

// 	// create the application if it does not exist
// 	request.Log.Info(fmt.Sprintf("creating oauth application in gitlab [%s]", request.Desired.Spec.DisplayName))
// 	if application == nil {
// 		application, err = request.GitLabClient.CreateApplication(request.Desired.Spec.DisplayName, request.Original.Status.CallbackURL)
// 		if err != nil {
// 			return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
// 				"unable to create oauth application in gitlab - %w",
// 				err,
// 			)
// 		}

// 		// set the client id and secret on the request
// 		request.ClientID = application.ApplicationID
// 		request.ClientSecret = application.Secret
// 	}

// 	// return if the application is already in the desired state
// 	if identityprovider.EqualGitLab(*application, *identityprovider.DesiredGitLab(
// 		request.Desired.Spec.DisplayName,
// 		application.ApplicationID,
// 		application.Secret,
// 		request.Original.Status.CallbackURL,
// 		true,
// 	)) {
// 		return controllers.NoRequeue(), nil
// 	}

// 	// return an error as we will not allow updates to the gitlab application
// 	return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), ErrGitLabApplicationDrift
// }

// ApplyIdentityProvider applies the GitLab identity provider state to OCM.  This includes creating and/or updating
// the identity provider based on the provided attributes from the custom resource.
func (r *Controller) ApplyIdentityProvider(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if request.desired() {
		request.Log.V(controllers.LogLevelDebug).Info(
			"gitlab identity provider already in desired state",
			controllers.LogValues(request)...,
		)

		return controllers.NoRequeue(), nil
	}

	builder := request.Desired.Builder(request.CA, request.ClientSecret)

	// create the identity provider if it does not exist
	if request.Current == nil {
		request.Log.Info("creating gitlab identity provider", controllers.LogValues(request)...)
		idp, err := request.OCMClient.Create(builder)
		if err != nil {
			return controllers.RequeueOnError(request, controllers.CreateOCMError(request, err))
		}

		// store the required provider data in the status
		original := request.Original.DeepCopy()
		request.Original.Status.ProviderID = idp.ID()

		if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
			return errUnableToUpdateStatusProviderID(request, idp.ID(), err)
		}

		// create an event indicating that the gitlab identity provider has been created
		events.RegisterAction(events.Created, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

		return controllers.NoRequeue(), nil
	}

	// update the identity provider if it does exist
	request.Log.Info("updating gitlab identity provider", controllers.LogValues(request)...)
	_, err := request.OCMClient.Update(builder)
	if err != nil {
		return controllers.RequeueOnError(request, controllers.UpdateOCMError(request, err))
	}

	// create an event indicating that the gitlab identity provider has been updated
	events.RegisterAction(events.Updated, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

	return controllers.NoRequeue(), nil
}

// Destroy will destroy an OpenShift Cluster Manager GitLab Identity Provider.
func (r *Controller) Destroy(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the gitlab identity provider
	if conditions.IsSet(conditions.IdentityProviderDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	// return if the cluster does not exist (has been deleted)
	_, exists, err := ocm.ClusterExists(request.Desired.Spec.ClusterName, request.Reconciler.Connection)
	if err != nil {
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), err
	}

	if !exists {
		return controllers.NoRequeue(), nil
	}

	ocmClient := ocm.NewIdentityProviderClient(
		request.Reconciler.Connection,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	// delete the object
	if err := ocmClient.Delete(request.Original.Status.ProviderID); err != nil {
		return controllers.RequeueOnError(request, controllers.DeleteOCMError(request, err))
	}

	// create an event indicating that the gitlab identity provider has been deleted
	events.RegisterAction(events.Deleted, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

	// set the deleted condition
	if err := conditions.Update(request.Context, request.Reconciler, request.Original, conditions.IdentityProviderDeleted()); err != nil {
		return controllers.RequeueOnError(request, controllers.UpdateDeletedConditionError(err))
	}

	return controllers.NoRequeue(), nil
}

// Complete will perform all actions required to successful complete a reconciliation request.  It will
// requeue after the interval value requested by the controller configuration to ensure that the
// object remains in its desired state at a specific interval.
func (r *Controller) Complete(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	if err := conditions.Update(
		request.Context,
		request.Reconciler,
		request.Original,
		conditions.Reconciled(request.Trigger),
	); err != nil {
		return controllers.RequeueOnError(request, controllers.UpdateReconcilingConditionError(err))
	}

	request.Log.Info("completed gitlab identity provider reconciliation", controllers.LogValues(request)...)
	request.Log.Info(fmt.Sprintf("reconciling again in %s", r.Interval.String()), controllers.LogValues(request)...)

	return controllers.RequeueAfter(r.Interval), nil
}

// CompleteDestroy will perform all actions required to successfully complete a delete reconciliation request.
func (r *Controller) CompleteDestroy(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	if err := controllers.RemoveFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueOnError(request, controllers.RemoveFinalizerError(err))
	}

	request.Log.Info("completed gitlab identity provider deletion", controllers.LogValues(request)...)

	return controllers.NoRequeue(), nil
}
