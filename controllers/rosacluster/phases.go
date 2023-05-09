package rosacluster

import (
	"fmt"

	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
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

	return controllers.NoRequeue(), nil
}

// ApplyCluster applies the desired state of the LDAP rosa cluster to OCM.
func (r *Controller) ApplyCluster(request *ROSAClusterRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if request.desired() {
		request.Log.V(controllers.LogLevelDebug).Info("rosa cluster already in desired state", request.logValues()...)

		return controllers.NoRequeue(), nil
	}

	// create the rosa cluster if it does not exist
	if request.Current == nil {
		if err := request.createCluster(); err != nil {
			return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
				"error in createCluster - %w",
				err,
			)
		}

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
