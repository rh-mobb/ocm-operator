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

// Phase defines an individual phase in the controller reconciliation process.
type Phase struct {
	Name     string
	Function func(*ROSAClusterRequest) (ctrl.Result, error)
}

// Begin begins the reconciliation state once we get the object (the desired state) from the cluster.
// It is mainly used to set conditions of the controller and to let anyone who is viewiing the
// custom resource know that we are currently reconciling.
func (r *Controller) Begin(request *ROSAClusterRequest) (ctrl.Result, error) {
	if err := request.updateCondition(conditions.Reconciling(request.Trigger)); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error updating reconciling condition - %w", err)
	}

	return controllers.NoRequeue(), nil
}

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

		// set the created condition
		if err := request.updateCondition(ClusterCreated()); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"error updating created condition - %w",
				err,
			)
		}

		return controllers.NoRequeue(), nil
	}

	// return and pass to the 'waiting' step if not ready
	if request.Cluster.State() != clustersmgmtv1.ClusterStateReady {
		return controllers.NoRequeue(), nil
	}

	// return if it is already in its desired state
	if request.desired() {
		request.Log.V(controllers.LogLevelDebug).Info("rosa cluster already in desired state", request.logValues()...)

		return controllers.NoRequeue(), nil
	}

	// update the existing rosa cluster
	if err := request.updateCluster(); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"error in updateCluster - %w",
			err,
		)
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
	request.Log.Info("deleting cluster", request.logValues()...)
	request.OCMClient = ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName)

	if err := request.OCMClient.Delete(request.Original.Status.ClusterID); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to delete cluster with id [%s] from ocm - %w",
			request.Original.Status.ClusterID,
			err,
		)
	}

	// create an event indicating that the cluster has been deleted
	events.RegisterAction(
		events.Deleted,
		request.Original,
		r.Recorder,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	// set the uninstalling condition
	if err := request.updateCondition(ClusterUninstalling()); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"error updating uninstalling condition - %w",
			err,
		)
	}

	return controllers.NoRequeue(), nil
}

// WaitUntilMissing will requeue until the reconciler determines that the cluster is missing.
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

	// return if we are still uninstalling
	if cluster.State() == clustersmgmtv1.ClusterStateUninstalling {
		request.Log.Info("cluster is still uninstalling", request.logValues()...)
		request.Log.Info(fmt.Sprintf("checking again in %s", request.provisionRequeueTime().String()), request.logValues()...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	}

	// set the deleted condition and return if we have no cluster
	if cluster == nil {
		if err := request.updateCondition(ClusterDeleted()); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"error updating deleted condition - %w",
				err,
			)
		}

		return controllers.NoRequeue(), nil
	}

	request.Log.Info("cluster has been deleted", request.logValues()...)

	return controllers.NoRequeue(), nil
}

// DestroyOperatorRoles destroys the operator roles in AWS.
func (r *Controller) DestroyOperatorRoles(request *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the operator roles
	if conditions.IsSet(OperatorRolesDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	request.Log.Info("deleting operator roles", request.logValues()...)
	if err := request.destroyOperatorRoles(); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to destroy operator roles - %w",
			err,
		)
	}

	// update the status indicating the operator roles have been deleted
	if err := request.updateCondition(OperatorRolesDeleted()); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"error updating operator roles deleted condition - %w",
			err,
		)
	}

	return controllers.NoRequeue(), nil
}

// DestroyOIDC destroys the OIDC configuration and provider in AWS.
func (r *Controller) DestroyOIDC(request *ROSAClusterRequest) (ctrl.Result, error) {
	// only destroy the oidc provider if we have not already done so
	if !conditions.IsSet(OIDCProviderDeleted(), request.Original) {
		request.Log.Info("deleting oidc provider", request.logValues()...)
		if err := ocm.NewOIDCConfigClient(
			request.Reconciler.Connection,
		).Delete(request.Original.Status.OIDCConfigID); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"unable to delete oidc provider - %w",
				err,
			)
		}

		// update the status indicating the oidc provider has been deleted
		if err := request.updateCondition(OIDCProviderDeleted()); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"error updating oidc provider deleted condition - %w",
				err,
			)
		}
	}

	// only destroy the oidc configuration if we have not already done so
	if !conditions.IsSet(OIDCConfigDeleted(), request.Original) {
		request.Log.Info("deleting oidc config", request.logValues()...)
		if err := request.AWSClient.DeleteOIDCProvider(request.Original.Status.OIDCProviderARN); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"unable to delete oidc config - %w",
				err,
			)
		}

		// update the status indicating the oidc config has been deleted
		if err := request.updateCondition(OIDCConfigDeleted()); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"error updating oidc config deleted condition - %w",
				err,
			)
		}
	}

	return controllers.NoRequeue(), nil
}

// WaitUntilReady will requeue until the reconciler determines that the current state of the
// resource in the cluster is ready.
//
//nolint:exhaustive,goerr113
func (r *Controller) WaitUntilReady(request *ROSAClusterRequest) (ctrl.Result, error) {
	switch request.Cluster.State() {
	case clustersmgmtv1.ClusterStateReady:
		request.Log.Info("cluster is ready", request.logValues()...)

		return controllers.NoRequeue(), nil
	case clustersmgmtv1.ClusterStateError:
		request.Log.Error(fmt.Errorf("cluster is in error state"), fmt.Sprintf(
			"checking again in %s", request.provisionRequeueTime().String(),
		), request.logValues()...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	default:
		request.Log.Info("cluster is not ready", request.logValues()...)
		request.Log.Info(fmt.Sprintf("checking again in %s", request.provisionRequeueTime().String()), request.logValues()...)

		return controllers.RequeueAfter(request.provisionRequeueTime()), nil
	}
}

// Complete will perform all actions required to successful complete a reconciliation request.  It will
// requeue after the interval value requested by the controller configuration to ensure that the
// object remains in its desired state at a specific interval.
func (r *Controller) Complete(request *ROSAClusterRequest) (ctrl.Result, error) {
	if err := request.updateCondition(conditions.Reconciled(request.Trigger)); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("error updating reconciled condition - %w", err)
	}

	request.Log.Info("completed rosa cluster reconciliation", request.logValues()...)
	request.Log.Info(fmt.Sprintf("reconciling again in %s", r.Interval.String()), request.logValues()...)

	return controllers.RequeueAfter(r.Interval), nil
}

// CompleteDestroy will perform all actions required to successful complete a reconciliation request.
func (r *Controller) CompleteDestroy(request *ROSAClusterRequest) (ctrl.Result, error) {
	if err := controllers.RemoveFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("unable to remove finalizers - %w", err)
	}

	request.Log.Info("completed rosa cluster deletion", request.logValues()...)

	return controllers.NoRequeue(), nil
}
