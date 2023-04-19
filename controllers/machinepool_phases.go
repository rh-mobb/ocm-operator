package controllers

import (
	"fmt"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type MachinePoolPhaseFunc func(*MachinePoolRequest) (ctrl.Result, error)

// Begin begins the reconciliation state once we get the object (the desired state) from the cluster.
// It is mainly used to set conditions of the controller and to let anyone who is viewiing the
// custom resource know that we are currently reconciling.
func (r *MachinePoolReconciler) Begin(request *MachinePoolRequest) (ctrl.Result, error) {
	// set the reconciling conditions
	if err := r.updateReconcilingCondition(
		request,
		metav1.ConditionTrue,
		"controller requested reconciliation",
		"beginning controller reconciliation",
	); err != nil {
		return requeue(), err
	}

	return noRequeue(), nil
}

// GetDesiredState gets the the MachinePool resource from the cluster.  The desired state of the MachinePool resource
// is stored in a custom resource within the OpenShift cluster in which this controller is reconciling against.  It will
// be compared against the current state which exists in OpenShift Cluster Manager.
func (r *MachinePoolReconciler) GetDesiredState(request *MachinePoolRequest) (ctrl.Result, error) {
	// ensure the our managed labels do not conflict with what was submitted
	// to the cluster
	// TODO: move to validating/mutating webhook or CRD CEL language
	if request.Desired.HasManagedLabels() {
		return requeue(), fmt.Errorf(
			"invalid labels [%+v] - %w",
			request.Desired.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	// set the display name
	request.Desired.Spec.DisplayName = request.Desired.GetDisplayName()

	// set the managed labels on the desired state.  we do this because we expect
	// that the current state should have these labels.
	request.Desired.SetMachinePoolLabels()

	return noRequeue(), nil
}

// GetCurrentState gets the current state of the MachinePool resoruce.  The current state of the MachinePool resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *MachinePoolReconciler) GetCurrentState(request *MachinePoolRequest) (ctrl.Result, error) {
	// retrieve the cluster id
	clusterID := request.Desired.Status.ClusterID
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

		// store the cluster id in the status
		updatedObject := *request.Desired
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
	request.Client = ocm.NewMachinePoolClient(r.Connection, request.Desired.Spec.DisplayName, clusterID)
	if err := request.Client.Get(); err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"unable to retrieve machine pool from ocm [name=%s, clusterName=%s] - %w",
			request.Desired.Spec.DisplayName,
			request.Desired.Spec.ClusterName,
			err,
		)
	}

	// return if we did not find a machine pool.  this means that the machine pool does not
	// exist and must be created in the CreateOrUpdate phase.
	if !request.hasMachinePool() {
		return noRequeue(), nil
	}

	// copy the machine pool object from ocm into a
	// new object.
	if err := request.Current.CopyFrom(request.Client.MachinePool.Object, request.Desired.Spec.ClusterName); err != nil {
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

// CreateOrUpdate will create an OpenShift Cluster Manager machine pool if it does not exist,
// or update an OpenShift Cluster Manager machine pool if it does exist.
func (r *MachinePoolReconciler) CreateOrUpdate(request *MachinePoolRequest) (ctrl.Result, error) {
	builder := request.Desired.Builder()

	// if we have a current state, we need to compare it for equality
	if request.hasMachinePool() {
		// return if it is already in its desired state
		if request.desired() {
			request.Log.Info(machinePoolLog("machine pool already in desired state", request))

			return noRequeue(), nil
		}

		// update the object
		request.Log.Info(machinePoolLog("updating machine pool", request))
		if err := request.Client.Update(builder); err != nil {
			return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to update machine pool - %w", err)
		}

		return noRequeue(), nil
	}

	// create the object
	request.Log.Info(machinePoolLog("creating machine pool", request))
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
	nodes, err := kubernetes.GetLabeledNodes(request.Context, r, request.Desired.Labels)
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

	// set the nodes ready on the request
	request.NodesReady = true

	return noRequeue(), nil
}

// WaitUntilDeleted will requeue until the reconciler determines that the nodes
// no longer exist in the cluster.
func (r *MachinePoolReconciler) WaitUntilDeleted(request *MachinePoolRequest) (ctrl.Result, error) {
	nodes, err := kubernetes.GetLabeledNodes(request.Context, r, request.Desired.Labels)
	if err != nil {
		return requeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to get labeled nodes - %w", err)
	}

	// return if we cannot find any nodes
	if len(nodes.Items) > 0 {
		return requeueAfter(defaultMachinePoolRequeue), nil
	}

	return noRequeue(), nil
}

// Complete will perform all actions required to successful complete a reconciliation request.
func (r *MachinePoolReconciler) Complete(request *MachinePoolRequest) (ctrl.Result, error) {
	// set the reconciling conditions
	if err := r.updateReconcilingCondition(
		request,
		metav1.ConditionFalse,
		"controller finished reconciliation",
		"ending controller reconciliation",
	); err != nil {
		return requeue(), err
	}

	return noRequeue(), nil
}

func (r *MachinePoolReconciler) updateReconcilingCondition(
	request *MachinePoolRequest,
	status metav1.ConditionStatus,
	reason, message string,
) error {
	// set the reconciling conditions
	request.Desired.Status.Conditions = addCondition(
		request.Desired.Status.Conditions, conditionReconciling(
			status,
			"controller requested reconciliation",
			"beginning controller reconciliation",
		),
	)

	// update the reconciling conditions
	if err := r.Status().Update(request.Context, request.Desired); err != nil {
		return fmt.Errorf("unable to update status conditions - %w", err)
	}

	return nil
}

func machinePoolLog(message string, request *MachinePoolRequest) string {
	return fmt.Sprintf(
		"%s: name=%s, cluster=%s",
		message,
		request.Desired.GetDisplayName(),
		request.Desired.Spec.ClusterName,
	)
}
