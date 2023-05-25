package ldapidentityprovider

import (
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/events"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	defaultLDAPIdentityProviderRequeue = 30 * time.Second
)

// GetCurrentState gets the current state of the LDAPIdentityProvider resoruce.  The current state of the LDAPIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(request *LDAPIdentityProviderRequest) (ctrl.Result, error) {
	// retrieve the cluster id
	clusterID := request.Original.Status.ClusterID
	if clusterID == "" {
		if err := request.updateStatusCluster(); err != nil {
			return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), err
		}

		clusterID = request.Original.Status.ClusterID
	}

	// get the generic identity provider object from ocm
	request.OCMClient = ocm.NewIdentityProviderClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName, clusterID)

	idp, err := request.OCMClient.Get()
	if err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf(
			"unable to retrieve identity provider from ocm - %w",
			err,
		)
	}

	// return if there is no identity provider found
	if idp == nil {
		return controllers.NoRequeue(), nil
	}

	// store the current state
	request.Current = &ocmv1alpha1.LDAPIdentityProvider{}
	request.Current.Spec.ClusterName = request.Desired.Spec.ClusterName
	request.Current.Spec.DisplayName = request.Desired.Spec.DisplayName
	request.Current.Spec.BindPassword.Name = request.Desired.Spec.BindPassword.Name
	request.Current.Spec.CA.Name = request.Desired.Spec.CA.Name
	request.Current.Spec.MappingMethod = string(idp.MappingMethod())
	request.Current.CopyFrom(idp.LDAP())

	return controllers.NoRequeue(), nil
}

// ApplyIdentityProvider applies the desired state of the LDAP identity provider to OCM.
func (r *Controller) ApplyIdentityProvider(request *LDAPIdentityProviderRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if request.desired() {
		request.Log.V(controllers.LogLevelDebug).Info(
			"ldap identity provider already in desired state",
			controllers.LogValues(request)...,
		)

		return controllers.NoRequeue(), nil
	}

	builder := request.Desired.Builder(request.DesiredCA, request.DesiredBindPassword)

	// create the identity provider if it does not exist
	if request.Current == nil {
		request.Log.Info("creating ldap identity provider", controllers.LogValues(request)...)
		idp, err := request.OCMClient.Create(builder)
		if err != nil {
			return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf(
				"unable to create ldap identity provider in ocm - %w",
				err,
			)
		}

		// store the required provider data in the status
		original := request.Original.DeepCopy()
		request.Original.Status.ProviderID = idp.ID()

		if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
			return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf(
				"unable to update status providerID=%s - %w",
				idp.ID(),
				err,
			)
		}

		// create an event indicating that the ldap identity provider has been created
		events.RegisterAction(events.Created, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

		return controllers.NoRequeue(), nil
	}

	// update the identity provider if it does exist
	request.Log.Info("updating ldap identity provider", controllers.LogValues(request)...)
	_, err := request.OCMClient.Update(builder)
	if err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf(
			"unable to update ldap identity provider in ocm - %w",
			err,
		)
	}

	// create an event indicating that the ldap identity provider has been updated
	events.RegisterAction(events.Updated, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

	return controllers.NoRequeue(), nil
}

// Destroy will destroy an OpenShift Cluster Manager LDAP Identity Provider.
func (r *Controller) Destroy(request *LDAPIdentityProviderRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the ldap identity provider
	if conditions.IsSet(conditions.IdentityProviderDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	ocmClient := ocm.NewIdentityProviderClient(
		request.Reconciler.Connection,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	// delete the object
	if err := ocmClient.Delete(request.Original.Status.ProviderID); err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf(
			"unable to create ldap identity provider in ocm - %w",
			err,
		)
	}

	// create an event indicating that the ldap identity provider has been deleted
	events.RegisterAction(events.Deleted, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

	// set the deleted condition
	if err := conditions.Update(request.Context, request.Reconciler, request.Original, conditions.IdentityProviderDeleted()); err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf("error updating deleted condition - %w", err)
	}

	return controllers.NoRequeue(), nil
}

// Complete will perform all actions required to successful complete a reconciliation request.  It will
// requeue after the interval value requested by the controller configuration to ensure that the
// object remains in its desired state at a specific interval.
func (r *Controller) Complete(request *LDAPIdentityProviderRequest) (ctrl.Result, error) {
	if err := conditions.Update(
		request.Context,
		request.Reconciler,
		request.Original,
		conditions.Reconciled(request.Trigger),
	); err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf("error updating reconciling condition - %w", err)
	}

	request.Log.Info("completed ldap identity provider reconciliation", controllers.LogValues(request)...)
	request.Log.Info(fmt.Sprintf("reconciling again in %s", r.Interval.String()), controllers.LogValues(request)...)

	return controllers.RequeueAfter(r.Interval), nil
}

// CompleteDestroy will perform all actions required to successful complete a reconciliation request.
func (r *Controller) CompleteDestroy(request *LDAPIdentityProviderRequest) (ctrl.Result, error) {
	if err := controllers.RemoveFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultLDAPIdentityProviderRequeue), fmt.Errorf("unable to remove finalizers - %w", err)
	}

	request.Log.Info("completed ldap identity provider deletion", controllers.LogValues(request)...)

	return controllers.NoRequeue(), nil
}
