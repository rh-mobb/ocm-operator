package controllers

import (
	"context"
	"fmt"
	"reflect"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MachinePoolPhaseFunc func(*MachinePoolRequest) error

// GetDesiredState gets the the MachinePool resource from the cluster.  The desired state of the MachinePool resource
// is stored in a custom resource within the OpenShift cluster in which this controller is reconciling against.  It will
// be compared against the current state which exists in OpenShift Cluster Manager.
func (r *MachinePoolReconciler) GetDesiredState(request *MachinePoolRequest) error {
	desiredState := &ocmv1alpha1.MachinePool{}

	// get the resource from the cluster
	if err := r.Get(request.Context, request.ControllerRequest.NamespacedName, desiredState); err != nil {
		if !apierrs.IsNotFound(err) {
			return fmt.Errorf("unable to fetch machine pool from cluster - %w", err)
		}

		return err
	}

	// ensure the our managed labels do not conflict with what was submitted
	// to the cluster
	// TODO: move to validating/mutating webhook
	if desiredState.HasManagedLabels() {
		return fmt.Errorf("invalid labels [%+v] - %w", desiredState.Spec.Labels, ErrMachinePoolReservedLabel)
	}

	// set the display name
	desiredState.Spec.DisplayName = desiredState.GetDisplayName()

	// set the managed labels on the desired state.  we do this because we expect
	// that the current state should have these labels.
	desiredState.SetMachinePoolLabels()
	request.DesiredState = desiredState

	return nil
}

// GetCurrentState gets the current state of the MachinePool resoruce.  The current state of the MachinePool resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *MachinePoolReconciler) GetCurrentState(request *MachinePoolRequest) error {
	// retrieve the cluster id
	clusterID := ocm.ClusterIDFromContext(r.Context, request.ControllerRequest)
	if clusterID == "" {
		cluster := ocm.NewClusterClient(r.Connection, request.DesiredState.Spec.ClusterName)
		if err := cluster.Get(); err != nil {
			return fmt.Errorf(
				"unable to retrieve cluster from ocm [name=%s] - %w",
				request.DesiredState.Spec.ClusterName,
				err,
			)
		}

		clusterID = cluster.Object.ID()
		r.Context = context.WithValue(request.Context, ocm.ClusterIDContextKey(request.ControllerRequest), clusterID)
	}

	// retrieve the machine pool
	if r.ClientOCM == nil {
		machinePool := ocm.NewMachinePoolClient(r.Connection, request.DesiredState.Spec.DisplayName, clusterID)

		r.ClientOCM = machinePool
	}

	if err := r.ClientOCM.Get(); err != nil {
		return fmt.Errorf(
			"unable to retrieve machine pool from ocm [name=%s, clusterName=%s] - %w",
			request.DesiredState.Spec.DisplayName,
			request.DesiredState.Spec.ClusterName,
			err,
		)
	}

	// TODO: fix this logic
	if r.ClientOCM.Object == nil {
		return nil
	}

	request.MachinePool = r.ClientOCM.Object

	// copy the machine pool object from ocm into a
	// new object.
	currentState := &ocmv1alpha1.MachinePool{}
	if err := currentState.CopyFrom(r.ClientOCM.Object); err != nil {
		return fmt.Errorf("unable to copy ocm machine pool object - %w", err)
	}

	// ensure that we have the required labels for the machine pool
	// we found.  we do this to ensure we are not managing something that
	// may have been created by another process.
	// TODO: move to validating/mutating webhook
	if !currentState.HasManagedLabels() {
		return fmt.Errorf("missing managed labels [%+v] - %w", currentState.Spec.Labels, ErrMachinePoolReservedLabel)
	}

	request.CurrentState = currentState

	return nil
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
func (r *MachinePoolReconciler) WaitUntilReady(request *MachinePoolRequest) error {
	nodeList := corev1.NodeList{}

	labelSelector := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(request.DesiredState.Labels),
	}

	// list the nodes that have the appropriate labels
	if err := r.List(request.Context, &nodeList, labelSelector); err != nil {
		return fmt.Errorf("unable to list nodes in cluster - %w", err)
	}

	// set not ready if we cannot find any nodes
	if len(nodeList.Items) < 1 {
		request.Ready = false

		return nil
	}

	// ensure all nodes are ready
	for _, node := range nodeList.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeConditionType("Ready") && condition.Status == corev1.ConditionFalse {
				request.Ready = false

				return nil
			}
		}
	}

	// node is ready so we set the status and return
	request.Ready = true

	return nil
}

// CreateOrUpdate will create an OpenShift Cluster Manager machine pool if it does not exist,
// or update an OpenShift Cluster Manager machine pool if it does exist.
func (r *MachinePoolReconciler) CreateOrUpdate(request *MachinePoolRequest) error {
	builder := request.DesiredState.Builder()

	// if we have a current state, we need to compare it for equality
	if request.CurrentState != nil {
		// return if it is already in its desired state
		if Desired(request) {
			return nil
		}

		// update to the desired state
		// TODO:
	}

	// create the object
	if err := r.ClientOCM.Create(builder); err != nil {
		return fmt.Errorf("unable to create machine pool - %w", err)
	}

	return nil
}

// Desired will determine if the desired and current state of a resource are equal.
func Desired(request *MachinePoolRequest) bool {
	return reflect.DeepEqual(
		request.DesiredState.Spec,
		request.CurrentState.Spec,
	)
}
