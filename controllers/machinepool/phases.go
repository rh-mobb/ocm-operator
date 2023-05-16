package machinepool

import (
	"fmt"
	"reflect"
	"strings"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/events"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// Phase defines an individual phase in the controller reconciliation process.
type Phase struct {
	Name     string
	Function func(*MachinePoolRequest) (ctrl.Result, error)
}

// GetCurrentState gets the current state of the MachinePool resoruce.  The current state of the MachinePool resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
// TODO: needs refactor
//
//nolint:cyclop
func (r *Controller) GetCurrentState(request *MachinePoolRequest) (ctrl.Result, error) {
	// retrieve the cluster id
	clusterID := request.Original.Status.ClusterID
	if clusterID == "" {
		if err := request.updateStatusCluster(); err != nil {
			return controllers.RequeueAfter(defaultMachinePoolRequeue), err
		}

		clusterID = request.Original.Status.ClusterID
	}

	// retrieve the machine pool (or node pool for hosted control plane clusters)
	var pool interface{}

	var err error

	if request.Original.Status.Hosted {
		poolClient := ocm.NewNodePoolClient(r.Connection, request.Desired.Spec.DisplayName, clusterID)
		pool, err = poolClient.Get()
	} else {
		poolClient := ocm.NewMachinePoolClient(r.Connection, request.Desired.Spec.DisplayName, clusterID)
		pool, err = poolClient.Get()
	}

	if err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"unable to retrieve machine pool from ocm [name=%s, clusterName=%s] - %w",
			request.Desired.Spec.DisplayName,
			request.Desired.Spec.ClusterName,
			err,
		)
	}

	// return if we did not find a machine pool.  this means that the machine pool does not
	// exist and must be created in the CreateOrUpdate phase.
	if reflect.ValueOf(pool).IsNil() {
		return controllers.NoRequeue(), nil
	}

	// copy the machine pool object from ocm into a
	// new object.
	request.Current = &ocmv1alpha1.MachinePool{}
	if request.Original.Status.Hosted {
		nodePool, ok := pool.(*clustersmgmtv1.NodePool)
		if !ok {
			return controllers.RequeueAfter(defaultMachinePoolRequeue), ocm.ErrConvertNodePool
		}

		err = request.Current.CopyFromNodePool(nodePool, request.Desired.Spec.ClusterName)
	} else {
		machinePool, ok := pool.(*clustersmgmtv1.MachinePool)
		if !ok {
			return controllers.RequeueAfter(defaultMachinePoolRequeue), ocm.ErrConvertMachinePool
		}

		err = request.Current.CopyFromMachinePool(machinePool, request.Desired.Spec.ClusterName)
	}

	if err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to copy ocm machine pool object - %w", err)
	}

	// ensure that we have the required labels for the machine pool
	// we found.  we do this to ensure we are not managing something that
	// may have been created by another process.
	if !request.Current.HasManagedLabels() {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf(
			"missing managed labels [%+v] - %w",
			request.Current.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	return controllers.NoRequeue(), nil
}

// Apply will create an OpenShift Cluster Manager machine pool if it does not exist,
// or update an OpenShift Cluster Manager machine pool if it does exist.
//
//nolint:forcetypeassert
func (r *Controller) Apply(request *MachinePoolRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if request.desired() {
		request.Log.V(controllers.LogLevelDebug).Info("machine pool already in desired state", request.logValues()...)

		return controllers.NoRequeue(), nil
	}

	// get the client
	var poolClient interface{}

	if request.Original.Status.Hosted {
		poolClient = ocm.NewNodePoolClient(
			r.Connection,
			request.Desired.Spec.DisplayName,
			request.Original.Status.ClusterID,
		)
	} else {
		poolClient = ocm.NewMachinePoolClient(
			r.Connection,
			request.Desired.Spec.DisplayName,
			request.Original.Status.ClusterID,
		)
	}

	// build the request
	request.Desired.Status.AvailabilityZones = request.Original.Status.AvailabilityZones
	request.Desired.Status.Subnets = request.Original.Status.Subnets

	// if no machine pool exists, create it and return
	//nolint:nestif
	if request.Current == nil {
		var createErr error

		request.Log.Info("creating machine pool", request.logValues()...)
		if request.Original.Status.Hosted {
			createErr = request.createNodePool(poolClient.(*ocm.NodePoolClient))
		} else {
			createErr = request.createMachinePool(poolClient.(*ocm.MachinePoolClient))
		}

		if createErr != nil {
			// if the cluster with same name is deleting, requeue without an error
			if strings.Contains(createErr.Error(), "is being deleted from cluster") {
				request.Log.Info(
					"machine pool with same name is deleting; requeueing",
					request.logValues()...,
				)

				return controllers.RequeueAfter(defaultMachinePoolRequeue), nil
			}

			return controllers.RequeueAfter(defaultMachinePoolRequeue), createErr
		}

		// create an event indicating that the machine pool has been created
		events.RegisterAction(events.Created, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

		return controllers.NoRequeue(), nil
	}

	// update the object
	var updateErr error

	request.Log.Info("updating machine pool", request.logValues()...)
	if request.Original.Status.Hosted {
		updateErr = request.updateNodePool(poolClient.(*ocm.NodePoolClient))
	} else {
		updateErr = request.updateMachinePool(poolClient.(*ocm.MachinePoolClient))
	}

	if updateErr != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), updateErr
	}

	// create an event indicating that the machine pool has been updated
	events.RegisterAction(events.Updated, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

	return controllers.NoRequeue(), nil
}

// Destroy will destroy an OpenShift Cluster Manager machine pool.
//
//nolint:forcetypeassert
func (r *Controller) Destroy(request *MachinePoolRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the machine pool
	if conditions.IsSet(MachinePoolDeleted(), request.Original) {
		return controllers.NoRequeue(), nil
	}

	// get the client
	var poolClient interface{}

	if request.Original.Status.Hosted {
		poolClient = ocm.NewNodePoolClient(
			r.Connection,
			request.Desired.Spec.DisplayName,
			request.Original.Status.ClusterID,
		)
	} else {
		poolClient = ocm.NewMachinePoolClient(
			r.Connection,
			request.Desired.Spec.DisplayName,
			request.Original.Status.ClusterID,
		)
	}

	// delete the object
	var deleteErr error

	request.Log.Info("deleting machine pool", request.logValues()...)
	if request.Original.Status.Hosted {
		deleteErr = request.deleteNodePool(poolClient.(*ocm.NodePoolClient))
	} else {
		deleteErr = request.deleteMachinePool(poolClient.(*ocm.MachinePoolClient))
	}

	if deleteErr != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), deleteErr
	}

	// create an event indicating that the machine pool has been deleted
	events.RegisterAction(events.Deleted, request.Original, r.Recorder, request.Desired.Spec.DisplayName, request.Original.Status.ClusterID)

	// set the deleted condition
	if err := conditions.Update(request.Context, request.Reconciler, request.Original, MachinePoolDeleted()); err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("error updating deleted condition - %w", err)
	}

	return controllers.NoRequeue(), nil
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
func (r *Controller) WaitUntilReady(request *MachinePoolRequest) (ctrl.Result, error) {
	// skip the wait check if we are not requesting to wait for readiness
	if !request.Original.Spec.Wait {
		return controllers.NoRequeue(), nil
	}

	nodes, err := kubernetes.GetLabeledNodes(request.Context, r, request.Desired.Spec.Labels)
	if err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to get labeled nodes - %w", err)
	}

	// return if we cannot find any nodes
	if len(nodes.Items) < 1 {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), nil
	}

	// ensure all nodes are ready
	if !kubernetes.NodesAreReady(nodes.Items...) {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), nil
	}

	request.Log.Info("nodes are ready", request.logValues()...)

	return controllers.NoRequeue(), nil
}

// WaitUntilMissing will requeue until the reconciler determines that the nodes
// no longer exist in the cluster.
func (r *Controller) WaitUntilMissing(request *MachinePoolRequest) (ctrl.Result, error) {
	// skip the wait check if we are not requesting to wait for readiness
	if !request.Original.Spec.Wait {
		return controllers.NoRequeue(), nil
	}

	nodes, err := kubernetes.GetLabeledNodes(request.Context, r, request.Desired.Spec.Labels)
	if err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to get labeled nodes - %w", err)
	}

	// return if we cannot find any nodes
	if len(nodes.Items) > 0 {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), nil
	}

	request.Log.Info("nodes have been removed", request.logValues()...)

	return controllers.NoRequeue(), nil
}

// Complete will perform all actions required to successful complete a reconciliation request.  It will
// requeue after the interval value requested by the controller configuration to ensure that the
// object remains in its desired state at a specific interval.
func (r *Controller) Complete(request *MachinePoolRequest) (ctrl.Result, error) {
	if err := conditions.Update(
		request.Context,
		request.Reconciler,
		request.Original,
		conditions.Reconciled(request.Trigger),
	); err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("error updating reconciling condition - %w", err)
	}

	request.Log.Info("completed machine pool reconciliation", request.logValues()...)
	request.Log.Info(fmt.Sprintf("reconciling again in %s", r.Interval.String()), request.logValues()...)

	return controllers.RequeueAfter(r.Interval), nil
}

// CompleteDestroy will perform all actions required to successful complete a reconciliation request.
func (r *Controller) CompleteDestroy(request *MachinePoolRequest) (ctrl.Result, error) {
	if err := controllers.RemoveFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultMachinePoolRequeue), fmt.Errorf("unable to remove finalizers - %w", err)
	}

	request.Log.Info("completed machine pool deletion", request.logValues()...)

	return controllers.NoRequeue(), nil
}
