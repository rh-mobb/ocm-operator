package gitlabidentityprovider

import (
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/identityprovider"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

var (
	ErrGitLabApplicationDrift = errors.New("gitlab application is immutable but differs from the desired state configuration")
)

// GetCurrentState gets the current state of the GitLabIdentityProvider resource.  The current state of the GitLabIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// retrieve the cluster id
	clusterID := request.Original.Status.ClusterID
	if clusterID == "" {
		if err := request.updateStatusCluster(); err != nil {
			return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), err
		}

		clusterID = request.Original.Status.ClusterID
	}

	// get the gitlab identity provider from ocm
	request.OCMClient = ocm.NewGitLabIdentityProviderClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName, clusterID)

	idp, err := request.OCMClient.Get()
	if err != nil {
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
			"unable to retrieve gitlab identity provider from ocm - %w",
			err,
		)
	}

	// return if there is no identity provider found
	if idp == nil {
		return controllers.NoRequeue(), nil
	}

	// store the current state
	request.Current = &ocmv1alpha1.GitLabIdentityProvider{}
	request.Current.Spec.ClusterName = request.Desired.Spec.ClusterName
	request.Current.Spec.DisplayName = request.Desired.Spec.DisplayName
	request.Current.Spec.AccessTokenSecret = request.Desired.Spec.AccessTokenSecret
	request.Current.CopyFrom(idp)

	return controllers.NoRequeue(), nil
}

// ApplyGitLab applies the state to a GitLab instance.  This includes creating and/or updating an application
// with the appropriate oauth URL from OpenShift.
func (r *Controller) ApplyGitLab(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// get the gitlab application from gitlab, using the display name as the name of the
	// application to search for
	application, err := request.GitLabClient.GetApplication(request.Desired.Spec.DisplayName)
	if err != nil {
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
			"unable to retrieve application from gitlab - %w",
			err,
		)
	}

	// create the application if it does not exist
	request.Log.Info(fmt.Sprintf("creating oauth application in gitlab [%s]", request.Desired.Spec.DisplayName))
	if application == nil {
		application, err = request.GitLabClient.CreateApplication(request.Desired.Spec.DisplayName, request.Original.Status.CallbackURL)
		if err != nil {
			return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
				"unable to create oauth application in gitlab - %w",
				err,
			)
		}

		// set the client id and secret on the request
		request.ClientID = application.ApplicationID
		request.ClientSecret = application.Secret
	}

	// return if the application is already in the desired state
	if identityprovider.EqualGitLab(*application, *identityprovider.DesiredGitLab(
		request.Desired.Spec.DisplayName,
		application.ApplicationID,
		application.Secret,
		request.Original.Status.CallbackURL,
		true,
	)) {
		return controllers.NoRequeue(), nil
	}

	// return an error as we will not allow updates to the gitlab application
	return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), ErrGitLabApplicationDrift
}

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

	builder := request.Desired.Builder(request.Desired.Spec.CA, request.ClientSecret)

	// create the identity provider if it does not exist
	if request.Current == nil {
		_, err := request.OCMClient.Create(builder)
		if err != nil {
			return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
				"unable to create gitlab identity provider in ocm - %w",
				err,
			)
		}
	}

	// update the identity provider if it does exist
	_, err := request.OCMClient.Update(builder)
	if err != nil {
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
			"unable to update gitlab identity provider in ocm - %w",
			err,
		)
	}

	return controllers.NoRequeue(), nil
}

// Destroy will destroy an OpenShift Cluster Manager GitLab Identity Provider.
func (r *Controller) Destroy(request *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the gitlab identity provider
	if conditions.IsSet(conditions.IdentityProviderDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
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
		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf("error updating reconciling condition - %w", err)
	}

	request.Log.Info("completed gitlab identity provider reconciliation", controllers.LogValues(request)...)
	request.Log.Info(fmt.Sprintf("reconciling again in %s", r.Interval.String()), controllers.LogValues(request)...)

	return controllers.RequeueAfter(r.Interval), nil
}
