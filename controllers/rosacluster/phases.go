package rosacluster

import (
	"fmt"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/events"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// GetCurrentState gets the current state of the LDAPIdentityProvider resoruce.  The current state of the LDAPIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(request *ROSAClusterRequest) (ctrl.Result, error) {
	// retrieve the cluster
	request.OCMClient = ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName)

	cluster, err := request.OCMClient.Get()
	if err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			request.Desired.Spec.DisplayName,
			err,
		)
	}

	// return immediately if we have no cluster
	if cluster == nil {
		return controllers.NoRequeue(), nil
	}

	// store the current state
	request.Current = &ocmv1alpha1.ROSACluster{}
	request.Current.Spec.DisplayName = request.Desired.Spec.DisplayName
	request.Current.CopyFrom(cluster)
	request.Cluster = cluster

	return controllers.NoRequeue(), nil
}

// ApplyCluster applies the desired state of the LDAP rosa cluster to OCM.
func (r *Controller) ApplyCluster(request *ROSAClusterRequest) (ctrl.Result, error) {
	// create the rosa cluster if it does not exist
	if request.Current == nil {
		// return immediately if we have already created the cluster
		if conditions.IsSet(ClusterCreated(), request.Original) {
			return controllers.NoRequeue(), nil
		}

		if err := request.createCluster(); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"error in createCluster - %w",
				err,
			)
		}

		// send a notification that the cluster has been created
		if err := request.notify(events.Created, ClusterCreated(), rosaConditionTypeCreated); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error sending cluster created notification - %w", err)
		}

		return controllers.NoRequeue(), nil
	}

	// return and pass to the 'waiting' step if not ready
	if request.Cluster.State() != clustersmgmtv1.ClusterStateReady {
		return controllers.NoRequeue(), nil
	}

	// return if it is already in its desired state
	if request.desired() {
		request.Log.V(controllers.LogLevelDebug).Info("rosa cluster already in desired state", controllers.LogValues(request)...)

		return controllers.NoRequeue(), nil
	}

	// update the existing rosa cluster
	if err := request.updateCluster(); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"error in updateCluster - %w",
			err,
		)
	}

	// send a notification that the cluster has been updated
	if err := request.notify(events.Updated, ClusterUpdated(), rosaConditionTypeUpdated); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error sending cluster updated notification - %w", err)
	}

	return controllers.NoRequeue(), nil
}

// DestroyCluster deletes the cluster from OCM.
func (r *Controller) DestroyCluster(request *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already uninstalled the cluster or if
	// we have not created the cluster
	if conditions.IsSet(ClusterUninstalling(), request.Original) || !conditions.IsSet(ClusterCreated(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	// delete the cluster
	request.Log.Info("deleting cluster", controllers.LogValues(request)...)
	request.OCMClient = ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName)

	if err := request.OCMClient.Delete(request.Original.Status.ClusterID); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to delete cluster with id [%s] from ocm - %w",
			request.Original.Status.ClusterID,
			err,
		)
	}

	// send a notification that the cluster has been deleted
	if err := request.notify(events.Deleted, ClusterDeleted(), rosaConditionTypeDeleted); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error sending cluster deleted notification - %w", err)
	}

	return controllers.NoRequeue(), nil
}

// WaitUntilMissing will requeue until the reconciler determines that the cluster is missing.
//
//nolint:exhaustive
func (r *Controller) WaitUntilMissing(request *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the cluster
	if conditions.IsSet(ClusterDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	// retrieve the cluster
	request.OCMClient = ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName)

	cluster, err := request.OCMClient.Get()
	if err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			request.Desired.Spec.DisplayName,
			err,
		)
	}

	// set the deleted condition and return if we have no cluster
	if cluster == nil {
		if err := conditions.Update(request.Context, request.Reconciler, request.Original, ClusterDeleted()); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error updating deleted condition - %w", err)
		}

		request.Log.Info("cluster has been deleted", controllers.LogValues(request)...)

		return controllers.NoRequeue(), nil
	}

	// return if we are still uninstalling
	switch cluster.State() {
	case clustersmgmtv1.ClusterStateUninstalling:
		request.Log.Info("cluster is still uninstalling", controllers.LogValues(request)...)
		request.Log.Info(fmt.Sprintf("checking again in %s", request.provisionRequeueTime().String()), controllers.LogValues(request)...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	case clustersmgmtv1.ClusterStateError:
		request.Log.Error(fmt.Errorf("cluster uninstalling is in error state"), fmt.Sprintf(
			"checking again in %s", request.provisionRequeueTime().String(),
		), controllers.LogValues(request)...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	default:
		request.Log.Error(fmt.Errorf("cluster uninstalling is in unknown state"), fmt.Sprintf(
			"checking again in %s", request.provisionRequeueTime().String(),
		), controllers.LogValues(request)...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	}
}

// DestroyOperatorRoles destroys the operator roles in AWS.
func (r *Controller) DestroyOperatorRoles(request *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the operator roles
	if conditions.IsSet(OperatorRolesDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	request.Log.Info("deleting operator roles", controllers.LogValues(request)...)
	if err := request.destroyOperatorRoles(); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to destroy operator roles - %w",
			err,
		)
	}

	// send a notification that the operator roles have been deleted
	if err := request.notify(events.Deleted, OperatorRolesDeleted(), awsConditionTypeOperatorRolesDeleted); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error sending operator roles deleted notification - %w", err)
	}

	return controllers.NoRequeue(), nil
}

// DestroyOIDC destroys the OIDC configuration and provider in AWS.
func (r *Controller) DestroyOIDC(request *ROSAClusterRequest) (ctrl.Result, error) {
	// only destroy the oidc provider if we have not already done so
	if !conditions.IsSet(OIDCProviderDeleted(), request.Original) {
		request.Log.Info("deleting oidc provider", controllers.LogValues(request)...)
		if err := ocm.NewOIDCConfigClient(
			request.Reconciler.Connection,
		).Delete(request.Original.Status.OIDCConfigID); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"unable to delete oidc provider - %w",
				err,
			)
		}

		// send a notification that the oidc provider has been deleted
		if err := request.notify(events.Deleted, OIDCProviderDeleted(), oidcConditionTypeProviderDeleted); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error sending oidc provider deleted notification - %w", err)
		}
	}

	// only destroy the oidc configuration if we have not already done so
	if !conditions.IsSet(OIDCConfigDeleted(), request.Original) {
		request.Log.Info("deleting oidc config", controllers.LogValues(request)...)
		if err := request.AWSClient.DeleteOIDCProvider(request.Original.Status.OIDCProviderARN); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"unable to delete oidc config - %w",
				err,
			)
		}

		// send a notification that the oidc config has been deleted
		if err := request.notify(events.Deleted, OIDCConfigDeleted(), oidcConditionTypeConfigDeleted); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error sending oidc config deleted notification - %w", err)
		}
	}

	return controllers.NoRequeue(), nil
}

// WaitUntilReady will requeue until the reconciler determines that the current state of the
// resource in the cluster is ready.
//
//nolint:exhaustive
func (r *Controller) WaitUntilReady(request *ROSAClusterRequest) (ctrl.Result, error) {
	switch request.Cluster.State() {
	case clustersmgmtv1.ClusterStateReady:
		request.Log.Info("cluster is ready", controllers.LogValues(request)...)

		return controllers.NoRequeue(), nil
	case clustersmgmtv1.ClusterStateError:
		request.Log.Error(fmt.Errorf("cluster is in error state"), fmt.Sprintf(
			"checking again in %s", request.provisionRequeueTime().String(),
		), controllers.LogValues(request)...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	default:
		request.Log.Info("cluster is not ready", controllers.LogValues(request)...)
		request.Log.Info(fmt.Sprintf("checking again in %s", request.provisionRequeueTime().String()), controllers.LogValues(request)...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	}
}

// Complete will perform all actions required to successful complete a reconciliation request.  It will
// requeue after the interval value requested by the controller configuration to ensure that the
// object remains in its desired state at a specific interval.
func (r *Controller) Complete(request *ROSAClusterRequest) (ctrl.Result, error) {
	if err := conditions.Update(request.Context, request.Reconciler, request.Original, conditions.Reconciled(request.Trigger)); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error updating reconciled condition - %w", err)
	}

	request.Log.Info("completed rosa cluster reconciliation", controllers.LogValues(request)...)
	request.Log.Info(fmt.Sprintf("reconciling again in %s", r.Interval.String()), controllers.LogValues(request)...)

	return controllers.RequeueAfter(r.Interval), nil
}

// CompleteDestroy will perform all actions required to successful complete a reconciliation request.
func (r *Controller) CompleteDestroy(request *ROSAClusterRequest) (ctrl.Result, error) {
	if err := controllers.RemoveFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("unable to remove finalizers - %w", err)
	}

	request.Log.Info("completed rosa cluster deletion", controllers.LogValues(request)...)

	return controllers.NoRequeue(), nil
}
