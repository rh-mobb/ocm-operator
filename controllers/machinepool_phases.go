package controllers

import (
	"fmt"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Begin begins the reconciliation state once we get the object (the desired state) from the cluster.
// It is mainly used to set conditions of the controller and to let anyone who is viewiing the
// custom resource know that we are currently reconciling.
func (r *MachinePoolReconciler) Begin(request *MachinePoolRequest) (ctrl.Result, error) {
	if err := updateCondition(
		request.Context,
		r,
		request.Original,
		conditionReconciling(metav1.ConditionTrue, request.Trigger, "beginning reconciliation"),
	); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("error updating reconciling condition - %w", err)
	}

	return noRequeue(), nil
}

// GetCurrentState gets the current state of the MachinePool resoruce.  The current state of the MachinePool resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *MachinePoolReconciler) GetCurrentState(request *MachinePoolRequest) (ctrl.Result, error) {
	// retrieve the cluster id
	clusterID := request.Original.Status.ClusterID
	if clusterID == "" {
		cluster := ocm.NewClusterClient(r.Connection, request.Desired.Spec.ClusterName)
		if err := cluster.Get(); err != nil {
			return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
				"unable to retrieve cluster from ocm [name=%s] - %w",
				request.Desired.Spec.ClusterName,
				err,
			)
		}

		clusterID = cluster.Object.ID()
		request.Original.Status.ClusterID = clusterID

		// store the cluster id in the status
		// TODO: refactor and switch to patch over update
		updatedObject := *request.Original
		updatedObject.Status.ClusterID = clusterID
		if err := r.Status().Update(request.Context, &updatedObject); err != nil {
			return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
				"unable to update status.clusterID=%s - %w",
				clusterID,
				err,
			)
		}
	}

	// retrieve the machine pool
	poolClient := ocm.NewMachinePoolClient(r.Connection, request.Desired.Spec.DisplayName, clusterID)
	machinePool, err := poolClient.Get()
	if err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"unable to retrieve machine pool from ocm [name=%s, clusterName=%s] - %w",
			request.Desired.Spec.DisplayName,
			request.Desired.Spec.ClusterName,
			err,
		)
	}

	// return if we did not find a machine pool.  this means that the machine pool does not
	// exist and must be created in the CreateOrUpdate phase.
	if machinePool == nil {
		return noRequeue(), nil
	}

	// copy the machine pool object from ocm into a
	// new object.
	request.Current = &ocmv1alpha1.MachinePool{}
	if err := request.Current.CopyFrom(machinePool, request.Desired.Spec.ClusterName); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to copy ocm machine pool object - %w", err)
	}

	// ensure that we have the required labels for the machine pool
	// we found.  we do this to ensure we are not managing something that
	// may have been created by another process.
	if !request.Current.HasManagedLabels() {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"missing managed labels [%+v] - %w",
			request.Current.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	return noRequeue(), nil
}

// Apply will create an OpenShift Cluster Manager machine pool if it does not exist,
// or update an OpenShift Cluster Manager machine pool if it does exist.
func (r *MachinePoolReconciler) Apply(request *MachinePoolRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if request.desired() {
		request.Log.Info(machinePoolLog("machine pool already in desired state", request))

		return noRequeue(), nil
	}

	// get the machine pool client
	poolClient := ocm.NewMachinePoolClient(
		r.Connection,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	// build the request
	builder := request.Desired.Builder()

	// if no machine pool exists, create it and return
	if request.Current == nil {
		request.Log.Info(machinePoolLog("creating machine pool", request))
		if _, err := poolClient.Create(builder); err != nil {
			return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to create machine pool - %w", err)
		}

		return noRequeue(), nil
	}

	// update the object
	request.Log.Info(machinePoolLog("updating machine pool", request))
	if _, err := poolClient.Update(builder); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to update machine pool - %w", err)
	}

	return noRequeue(), nil
}

// Destroy will destroy an OpenShift Cluster Manager machine pool.
func (r *MachinePoolReconciler) Destroy(request *MachinePoolRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the machine pool
	if hasCondition(request.Original.Status.Conditions, conditionDeleted()) {
		return noRequeue(), nil
	}

	// get the machine pool client
	poolClient := ocm.NewMachinePoolClient(
		r.Connection,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	// delete the object
	request.Log.Info(machinePoolLog("deleting machine pool", request))
	if err := poolClient.Delete(request.Desired.Spec.DisplayName); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to delete machine pool - %w", err)
	}

	// set the deleted condition
	if err := updateCondition(
		request.Context,
		r,
		request.Original,
		conditionDeleted(),
	); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("error updating deleted condition - %w", err)
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
	nodes, err := kubernetes.GetLabeledNodes(request.Context, r, request.Desired.Spec.Labels)
	if err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to get labeled nodes - %w", err)
	}

	// return if we cannot find any nodes
	if len(nodes.Items) < 1 {
		return requeueAfter(defaultMachinePoolRequeue), nil
	}

	// ensure all nodes are ready
	if !kubernetes.NodesAreReady(nodes.Items...) {
		return requeueAfter(defaultMachinePoolRequeue), nil
	}

	return noRequeue(), nil
}

// WaitUntilMissing will requeue until the reconciler determines that the nodes
// no longer exist in the cluster.
func (r *MachinePoolReconciler) WaitUntilMissing(request *MachinePoolRequest) (ctrl.Result, error) {
	nodes, err := kubernetes.GetLabeledNodes(request.Context, r, request.Desired.Spec.Labels)
	if err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to get labeled nodes - %w", err)
	}

	// return if we cannot find any nodes
	if len(nodes.Items) > 0 {
		return requeueAfter(defaultMachinePoolRequeue), nil
	}

	return noRequeue(), nil
}

// CompleteDestroy will perform all actions required to successful complete a reconciliation request.
func (r *MachinePoolReconciler) CompleteDestroy(request *MachinePoolRequest) (ctrl.Result, error) {
	if containsString(request.Original.GetFinalizers(), finalizerName(request.Original)) {
		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(request.Original, finalizerName(request.Original))

		if err := r.Update(request.Context, request.Original); err != nil {
			return noRequeue(), fmt.Errorf("unable to remove finalizer - %w", err)
		}
	}

	return noRequeue(), nil
}

// Complete will perform all actions required to successful complete a reconciliation request.
func (r *MachinePoolReconciler) Complete(request *MachinePoolRequest) (ctrl.Result, error) {
	if err := updateCondition(
		request.Context,
		r,
		request.Original,
		conditionReconciling(metav1.ConditionFalse, request.Trigger, "ending reconciliation"),
	); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("error updating reconciling condition - %w", err)
	}

	return noRequeue(), nil
}

func machinePoolLog(message string, request *MachinePoolRequest) string {
	return fmt.Sprintf(
		"%s: name=%s, cluster=%s",
		message,
		request.Desired.Spec.DisplayName,
		request.Desired.Spec.ClusterName,
	)
}
