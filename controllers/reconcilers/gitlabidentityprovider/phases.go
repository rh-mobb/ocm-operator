package gitlabidentityprovider

import (
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/events"
	"github.com/rh-mobb/ocm-operator/controllers/phases"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// GetCurrentState gets the current state of the GitLabIdentityProvider resource.  The current state of the GitLabIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(req *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// get the generic identity provider object from ocm
	req.OCMClient = ocm.NewIdentityProviderClient(
		req.Reconciler.Connection,
		req.Desired.Spec.DisplayName,
		req.Original.Status.ClusterID,
	)

	idp, err := req.OCMClient.Get()
	if err != nil {
		return requeue.OnError(req, ocm.GetError(req, err))
	}

	// return if there is no identity provider found
	if idp == nil {
		return phases.Next()
	}

	// store the current state
	// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
	// req.Current.Spec.AccessTokenSecret = req.Desired.Spec.AccessTokenSecret
	req.Current = &ocmv1alpha1.GitLabIdentityProvider{}
	req.Current.Spec.ClusterName = req.Desired.Spec.ClusterName
	req.Current.Spec.DisplayName = req.Desired.Spec.DisplayName
	req.Current.Spec.ClientSecret.Name = req.Desired.Spec.ClientSecret.Name
	req.Current.Spec.CA.Name = req.Desired.Spec.CA.Name
	req.Current.Spec.MappingMethod = string(idp.MappingMethod())
	req.Current.CopyFrom(idp)

	return phases.Next()
}

// TODO: see TODO in api/v1alpha1/gitlabidentityprovider_types.go file for explanation.
// // ApplyGitLab applies the state to a GitLab instance.  This includes creating and/or updating an application
// // with the appropriate oauth URL from OpenShift.
// func (r *Controller) ApplyGitLab(req *GitLabIdentityProviderRequest) (ctrl.Result, error) {
// 	// get the gitlab application from gitlab, using the display name as the name of the
// 	// application to search for
// 	application, err := req.GitLabClient.GetApplication(req.Desired.Spec.DisplayName)
// 	if err != nil {
// 		return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
// 			"unable to retrieve application from gitlab - %w",
// 			err,
// 		)
// 	}

// 	// create the application if it does not exist
// 	req.Log.Info(fmt.Sprintf("creating oauth application in gitlab [%s]", req.Desired.Spec.DisplayName))
// 	if application == nil {
// 		application, err = req.GitLabClient.CreateApplication(req.Desired.Spec.DisplayName, req.Original.Status.CallbackURL)
// 		if err != nil {
// 			return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), fmt.Errorf(
// 				"unable to create oauth application in gitlab - %w",
// 				err,
// 			)
// 		}

// 		// set the client id and secret on the request
// 		req.ClientID = application.ApplicationID
// 		req.ClientSecret = application.Secret
// 	}

// 	// return if the application is already in the desired state
// 	if identityprovider.EqualGitLab(*application, *identityprovider.DesiredGitLab(
// 		req.Desired.Spec.DisplayName,
// 		application.ApplicationID,
// 		application.Secret,
// 		req.Original.Status.CallbackURL,
// 		true,
// 	)) {
// 		return controllers.NoRequeue(), nil
// 	}

// 	// return an error as we will not allow updates to the gitlab application
// 	return controllers.RequeueAfter(defaultGitLabIdentityProviderRequeue), ErrGitLabApplicationDrift
// }

// ApplyIdentityProvider applies the GitLab identity provider state to OCM.  This includes creating and/or updating
// the identity provider based on the provided attributes from the custom resource.
func (r *Controller) ApplyIdentityProvider(req *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if req.desired() {
		r.Log.V(controllers.LogLevelDebug).Info(
			"gitlab identity provider already in desired state",
			controllers.LogValues(req)...,
		)

		return phases.Next()
	}

	builder := req.Desired.Builder(req.CA, req.ClientSecret)

	// create the identity provider if it does not exist
	if req.Current == nil {
		r.Log.Info("creating gitlab identity provider", controllers.LogValues(req)...)
		idp, err := req.OCMClient.Create(builder)
		if err != nil {
			return requeue.OnError(req, ocm.CreateError(req, err))
		}

		// store the required provider data in the status
		original := req.Original.DeepCopy()
		req.Original.Status.ProviderID = idp.ID()

		if err := kubernetes.PatchStatus(req.Context, req.Reconciler, original, req.Original); err != nil {
			return errUnableToUpdateStatusProviderID(req, idp.ID(), err)
		}

		// create an event indicating that the gitlab identity provider has been created
		events.RegisterAction(events.Created, req.Original, r.Recorder, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)

		return phases.Next()
	}

	// update the identity provider if it does exist
	r.Log.Info("updating gitlab identity provider", controllers.LogValues(req)...)
	_, err := req.OCMClient.Update(builder)
	if err != nil {
		return requeue.OnError(req, ocm.UpdateError(req, err))
	}

	// create an event indicating that the gitlab identity provider has been updated
	events.RegisterAction(events.Updated, req.Original, r.Recorder, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)

	return phases.Next()
}

// Destroy will destroy an OpenShift Cluster Manager GitLab Identity Provider.
func (r *Controller) Destroy(req *GitLabIdentityProviderRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the gitlab identity provider
	if conditions.IsSet(conditions.IdentityProviderDeleted(), req.Original) {
		return phases.Next()
	}

	// return if the cluster does not exist (has been deleted)
	_, exists, err := ocm.ClusterExists(req.Desired.Spec.ClusterName, req.Reconciler.Connection)
	if err != nil {
		return requeue.OnError(req, err)
	}

	if !exists {
		return phases.Next()
	}

	ocmClient := ocm.NewIdentityProviderClient(
		req.Reconciler.Connection,
		req.Desired.Spec.DisplayName,
		req.Original.Status.ClusterID,
	)

	// delete the object
	if err := ocmClient.Delete(req.Original.Status.ProviderID); err != nil {
		return requeue.OnError(req, ocm.DeleteError(req, err))
	}

	// create an event indicating that the gitlab identity provider has been deleted
	events.RegisterAction(events.Deleted, req.Original, r.Recorder, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)

	// set the deleted condition
	if err := conditions.Update(req, conditions.IdentityProviderDeleted()); err != nil {
		return requeue.OnError(req, conditions.UpdateDeletedConditionError(err))
	}

	return phases.Next()
}
