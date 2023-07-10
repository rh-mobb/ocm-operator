package rosacluster

import (
	"fmt"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/events"
	"github.com/rh-mobb/ocm-operator/controllers/phases"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// GetCurrentState gets the current state of the LDAPIdentityProvider resource.  The current state of the LDAPIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(req *ROSAClusterRequest) (ctrl.Result, error) {
	// retrieve the cluster
	req.OCMClient = ocm.NewClusterClient(req.Reconciler.Connection, req.Desired.Spec.DisplayName)

	cluster, err := req.OCMClient.Get()
	if err != nil {
		return requeue.OnError(req, fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			req.Desired.Spec.DisplayName,
			err,
		))
	}

	// return immediately if we have no cluster
	if cluster == nil {
		return phases.Next()
	}

	// store the current state
	req.Current = &ocmv1alpha1.ROSACluster{}
	req.Current.Spec.DisplayName = req.Desired.Spec.DisplayName
	req.Current.CopyFrom(cluster)
	req.Cluster = cluster

	return phases.Next()
}

// ApplyCluster applies the desired state of the LDAP rosa cluster to OCM.
func (r *Controller) ApplyCluster(req *ROSAClusterRequest) (ctrl.Result, error) {
	// create the rosa cluster if it does not exist
	if req.Current == nil {
		// return immediately if we have already created the cluster
		if conditions.IsSet(ClusterCreated(), req.Original) {
			return phases.Next()
		}

		if err := req.createCluster(); err != nil {
			return requeue.OnError(req, fmt.Errorf(
				"error in createCluster - %w",
				err,
			))
		}

		// send a notification that the cluster has been created
		if err := req.notify(events.Created, ClusterCreated(), rosaConditionTypeCreated); err != nil {
			return requeue.OnError(req, fmt.Errorf("error sending cluster created notification - %w", err))
		}

		return phases.Next()
	}

	// return and pass to the 'waiting' step if not ready
	if req.Cluster.State() != clustersmgmtv1.ClusterStateReady {
		return phases.Next()
	}

	// return if it is already in its desired state
	if req.desired() {
		req.Log.V(controllers.LogLevelDebug).Info("rosa cluster already in desired state", request.LogValues(req)...)

		return phases.Next()
	}

	// update the existing rosa cluster
	if err := req.updateCluster(); err != nil {
		return requeue.OnError(req, fmt.Errorf(
			"error in updateCluster - %w",
			err,
		))
	}

	// send a notification that the cluster has been updated
	if err := req.notify(events.Updated, ClusterUpdated(), rosaConditionTypeUpdated); err != nil {
		return requeue.OnError(req, fmt.Errorf("error sending cluster updated notification - %w", err))
	}

	return phases.Next()
}

// FindChildObjects finds all of the child objects related to this cluster.  This is intended to run during the delete
// workflow and will return a requeue if any child objects are found.  This is to prevent deletion of the cluster while
// objects are still attached, which leaves the controller spamming error messages.
func (r *Controller) FindChildObjects(req *ROSAClusterRequest) (ctrl.Result, error) {
	// loop through each of our children types and ensure we have no remaining objects based on the
	// status of the cluster id
	for _, object := range []workload.ClusterChild{
		&ocmv1alpha1.GitLabIdentityProvider{},
		&ocmv1alpha1.LDAPIdentityProvider{},
		&ocmv1alpha1.MachinePool{},
	} {
		exists, err := object.ExistsForClusterID(req.Context, req.Reconciler, req.Original.Status.ClusterID)
		if err != nil {
			return requeue.OnError(req, fmt.Errorf(
				"unable to determine if object [%T] has a parent cluster with ID [%s] - %w",
				object,
				req.Original.Status.ClusterID,
				err,
			))
		}

		// if our object type still exists, we need to requeue until
		// we do not have any objects to prevent deleting the cluster while we have
		// related objects.
		if exists {
			req.Log.Info(fmt.Sprintf("cluster [%s/%s] still has child objects of type [%T]...skipping deletion",
				req.Original.Namespace,
				req.Original.Name,
				object,
			), request.LogValues(req)...)

			return requeue.Retry(req)
		}
	}

	return phases.Next()
}

// DestroyCluster deletes the cluster from OCM.
func (r *Controller) DestroyCluster(req *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already uninstalled the cluster or if
	// we have not created the cluster
	if conditions.IsSet(ClusterUninstalling(), req.Original) || !conditions.IsSet(ClusterCreated(), req.Original) {
		return phases.Next()
	}

	// delete the cluster
	req.Log.Info("deleting cluster", request.LogValues(req)...)
	req.OCMClient = ocm.NewClusterClient(req.Reconciler.Connection, req.Desired.Spec.DisplayName)

	if err := req.OCMClient.Delete(req.Original.Status.ClusterID); err != nil {
		return requeue.OnError(req, fmt.Errorf(
			"unable to delete cluster with id [%s] from ocm - %w",
			req.Original.Status.ClusterID,
			err,
		))
	}

	// send a notification that the cluster has been deleted
	if err := req.notify(events.Deleted, ClusterUninstalling(), rosaConditionTypeUninstalling); err != nil {
		return requeue.OnError(req, fmt.Errorf("error sending cluster deleted notification - %w", err))
	}

	return phases.Next()
}

// WaitUntilMissing will requeue until the reconciler determines that the cluster is missing.
//
//nolint:exhaustive
func (r *Controller) WaitUntilMissing(req *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the cluster
	if conditions.IsSet(ClusterDeleted(), req.Original) {
		return phases.Next()
	}

	// retrieve the cluster and return if it does not exist (has been deleted)
	cluster, exists, err := ocm.ClusterExists(req.Desired.Spec.DisplayName, req.Reconciler.Connection)
	if err != nil {
		return requeue.OnError(req, fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			req.Desired.Spec.DisplayName,
			err,
		))
	}

	if !exists {
		return phases.Next()
	}

	// set the deleted condition and return if we have no cluster
	if cluster == nil {
		if err := conditions.Update(req, ClusterDeleted()); err != nil {
			return requeue.OnError(req, fmt.Errorf("error updating deleted condition - %w", err))
		}

		req.Log.Info("cluster has been deleted", request.LogValues(req)...)

		return phases.Next()
	}

	// return if we are still uninstalling
	switch cluster.State() {
	case clustersmgmtv1.ClusterStateUninstalling:
		req.Log.Info("cluster is still uninstalling", request.LogValues(req)...)
		req.Log.Info(fmt.Sprintf("checking again in %s", req.provisionRequeueTime().String()), request.LogValues(req)...)

		return requeue.After(req.provisionRequeueTime(), nil)
	case clustersmgmtv1.ClusterStateError:
		req.Log.Error(fmt.Errorf("cluster uninstalling is in error state"), fmt.Sprintf(
			"checking again in %s", req.provisionRequeueTime().String(),
		), request.LogValues(req)...)

		return requeue.After(req.provisionRequeueTime(), nil)
	default:
		req.Log.Error(fmt.Errorf("cluster uninstalling is in unknown state"), fmt.Sprintf(
			"checking again in %s", req.provisionRequeueTime().String(),
		), request.LogValues(req)...)

		return requeue.After(req.provisionRequeueTime(), nil)
	}
}

// DestroyOperatorRoles destroys the operator roles in AWS.
func (r *Controller) DestroyOperatorRoles(req *ROSAClusterRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the operator roles
	if conditions.IsSet(OperatorRolesDeleted(), req.Original) {
		return phases.Next()
	}

	req.Log.Info("deleting operator roles", request.LogValues(req)...)
	if err := req.destroyOperatorRoles(); err != nil {
		return requeue.OnError(req, fmt.Errorf(
			"unable to destroy operator roles - %w",
			err,
		))
	}

	// send a notification that the operator roles have been deleted
	if err := req.notify(events.Deleted, OperatorRolesDeleted(), awsConditionTypeOperatorRolesDeleted); err != nil {
		return requeue.OnError(req, fmt.Errorf("error sending operator roles deleted notification - %w", err))
	}

	return phases.Next()
}

// DestroyOIDC destroys the OIDC configuration and provider in AWS.
func (r *Controller) DestroyOIDC(req *ROSAClusterRequest) (ctrl.Result, error) {
	// only destroy the oidc provider if we have not already done so
	if !conditions.IsSet(OIDCProviderDeleted(), req.Original) {
		req.Log.Info("deleting oidc provider", request.LogValues(req)...)
		if err := ocm.NewOIDCConfigClient(
			req.Reconciler.Connection,
		).Delete(req.Original.Status.OIDCConfigID); err != nil {
			return requeue.OnError(req, fmt.Errorf(
				"unable to delete oidc provider - %w",
				err,
			))
		}

		// send a notification that the oidc provider has been deleted
		if err := req.notify(events.Deleted, OIDCProviderDeleted(), oidcConditionTypeProviderDeleted); err != nil {
			return requeue.OnError(req, fmt.Errorf("error sending oidc provider deleted notification - %w", err))
		}
	}

	// only destroy the oidc configuration if we have not already done so
	if !conditions.IsSet(OIDCConfigDeleted(), req.Original) {
		req.Log.Info("deleting oidc config", request.LogValues(req)...)
		if err := req.Reconciler.AWSClient.DeleteOIDCProvider(req.Original.Status.OIDCProviderARN); err != nil {
			return requeue.OnError(req, fmt.Errorf(
				"unable to delete oidc config - %w",
				err,
			))
		}

		// send a notification that the oidc config has been deleted
		if err := req.notify(events.Deleted, OIDCConfigDeleted(), oidcConditionTypeConfigDeleted); err != nil {
			return requeue.OnError(req, fmt.Errorf("error sending oidc config deleted notification - %w", err))
		}
	}

	return phases.Next()
}

// WaitUntilReady will requeue until the reconciler determines that the current state of the
// resource in the cluster is ready.
//
//nolint:exhaustive
func (r *Controller) WaitUntilReady(req *ROSAClusterRequest) (ctrl.Result, error) {
	switch req.Cluster.State() {
	case clustersmgmtv1.ClusterStateReady:
		req.Log.Info("cluster is ready", request.LogValues(req)...)

		return phases.Next()
	case clustersmgmtv1.ClusterStateError:
		req.Log.Error(fmt.Errorf("cluster is in error state"), fmt.Sprintf(
			"checking again in %s", req.provisionRequeueTime().String(),
		), request.LogValues(req)...)

		return requeue.After(req.provisionRequeueTime(), nil)
	default:
		req.Log.Info("cluster is not ready", request.LogValues(req)...)
		req.Log.Info(fmt.Sprintf("checking again in %s", req.provisionRequeueTime().String()), request.LogValues(req)...)

		return requeue.After(req.provisionRequeueTime(), nil)
	}
}
