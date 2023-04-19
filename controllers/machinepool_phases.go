package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MachinePoolPhaseFunc func(*MachinePoolRequest) (ctrl.Result, error)

// GetDesiredState gets the the MachinePool resource from the cluster.  The desired state of the MachinePool resource
// is stored in a custom resource within the OpenShift cluster in which this controller is reconciling against.  It will
// be compared against the current state which exists in OpenShift Cluster Manager.
func (r *MachinePoolReconciler) GetDesiredState(request *MachinePoolRequest) (ctrl.Result, error) {
	// get the resource from the cluster
	if err := r.Get(request.Context, request.ControllerRequest.NamespacedName, request.DesiredState); err != nil {
		return requeue(), fmt.Errorf("unable to fetch machine pool from cluster - %w", err)
	}

	// ensure the our managed labels do not conflict with what was submitted
	// to the cluster
	// TODO: move to validating/mutating webhook or CRD CEL language
	if request.DesiredState.HasManagedLabels() {
		return requeue(), fmt.Errorf(
			"invalid labels [%+v] - %w",
			request.DesiredState.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	// set the display name
	request.DesiredState.Spec.DisplayName = request.DesiredState.GetDisplayName()

	// set the managed labels on the desired state.  we do this because we expect
	// that the current state should have these labels.
	request.DesiredState.SetMachinePoolLabels()

	return noRequeue(), nil
}

// GetCurrentState gets the current state of the MachinePool resoruce.  The current state of the MachinePool resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *MachinePoolReconciler) GetCurrentState(request *MachinePoolRequest) (ctrl.Result, error) {
	// retrieve the cluster id
	clusterID := ocm.ClusterIDFromContext(r.Context, request.ControllerRequest)
	if clusterID == "" {
		cluster := ocm.NewClusterClient(r.Connection, request.DesiredState.Spec.ClusterName)
		if err := cluster.Get(); err != nil {
			return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
				"unable to retrieve cluster from ocm [name=%s] - %w",
				request.DesiredState.Spec.ClusterName,
				err,
			)
		}

		clusterID = cluster.Object.ID()
		r.Context = context.WithValue(request.Context, ocm.ClusterIDContextKey(request.ControllerRequest), clusterID)
	}

	// retrieve the machine pool
	request.Client = ocm.NewMachinePoolClient(r.Connection, request.DesiredState.Spec.DisplayName, clusterID)
	if err := request.Client.Get(); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"unable to retrieve machine pool from ocm [name=%s, clusterName=%s] - %w",
			request.DesiredState.Spec.DisplayName,
			request.DesiredState.Spec.ClusterName,
			err,
		)
	}

	// return if we did not find a machine pool.  this means that the machine pool does not
	// exist and must be created in the CreateOrUpdate phase.
	if request.Client.MachinePool.Object == nil {
		return noRequeue(), nil
	}

	// copy the machine pool object from ocm into a
	// new object.
	if err := request.CurrentState.CopyFrom(request.Client.MachinePool.Object); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to copy ocm machine pool object - %w", err)
	}

	// ensure that we have the required labels for the machine pool
	// we found.  we do this to ensure we are not managing something that
	// may have been created by another process.
	if !request.CurrentState.HasManagedLabels() {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"missing managed labels [%+v] - %w",
			request.CurrentState.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	return noRequeue(), nil
}

// CreateOrUpdate will create an OpenShift Cluster Manager machine pool if it does not exist,
// or update an OpenShift Cluster Manager machine pool if it does exist.
func (r *MachinePoolReconciler) CreateOrUpdate(request *MachinePoolRequest) (ctrl.Result, error) {
	builder := request.DesiredState.Builder()

	// if we have a current state, we need to compare it for equality
	if request.CurrentState != nil {
		// return if it is already in its desired state
		if desired(request) {
			return noRequeue(), nil
		}

		// update the object
		if err := request.Client.Update(builder); err != nil {
			return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to update machine pool - %w", err)
		}
	}

	// create the object
	if err := request.Client.Create(builder); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to create machine pool - %w", err)
	}

	return noRequeue(), nil
}

//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;list;watch

// The above allows us to retrieve the node status to see when the MachinePool is
// ready.  This limits the controller to only running in the cluster in which
// it is reconciling against and limits a centralized management solution for
// MachinePools.
//
// See https://github.com/rh-mobb/ocm-operator/issues/1

// WaitUntilReady will requeue until the reconciler determines that the current state of the
// resource in the cluster is ready.
func (r *MachinePoolReconciler) WaitUntilReady(request *MachinePoolRequest) (ctrl.Result, error) {
	nodeList := corev1.NodeList{}

	// list the nodes that have the appropriate labels. this ensures that we only find
	// nodes with the proper labels to include our own managed labels
	if err := r.List(
		request.Context,
		&nodeList,
		&client.ListOptions{LabelSelector: labels.SelectorFromSet(request.DesiredState.Labels)},
	); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to list nodes in cluster - %w", err)
	}

	// return if we cannot find any nodes
	if len(nodeList.Items) < 1 {
		return requeueAfter(defaultMachinePoolRequeue), nil
	}

	// ensure all nodes are ready
	for _, node := range nodeList.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeConditionType("Ready") && condition.Status == corev1.ConditionFalse {
				return requeueAfter(defaultMachinePoolRequeue), nil
			}
		}
	}

	// set the nodes ready on the request
	request.NodesReady = true

	return noRequeue(), nil
}

// Complete will perform all actions required to successful complete a reconciliation request.
func (r *MachinePoolReconciler) Complete(request *MachinePoolRequest) error {
	return nil
}

// desired will determine if the desired and current state of a resource are equal.
func desired(request *MachinePoolRequest) bool {
	return reflect.DeepEqual(
		request.DesiredState.Spec,
		request.CurrentState.Spec,
	)
}
