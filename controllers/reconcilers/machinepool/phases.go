package machinepool

import (
	"reflect"
	"strings"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/events"
	"github.com/rh-mobb/ocm-operator/controllers/phases"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// GetCurrentState gets the current state of the MachinePool resource.  The current state of the MachinePool resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(req *MachinePoolRequest) (ctrl.Result, error) {
	// retrieve the machine pool (or node pool for hosted control plane clusters)
	var pool interface{}

	var err error

	if req.Original.Status.Hosted {
		poolClient := ocm.NewNodePoolClient(r.Connection, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)
		pool, err = poolClient.Get()
	} else {
		poolClient := ocm.NewMachinePoolClient(r.Connection, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)
		pool, err = poolClient.Get()
	}

	if err != nil {
		return requeue.OnError(req, ocm.GetError(req, err))
	}

	// return if we did not find a machine pool.  this means that the machine pool does not
	// exist and must be created in the CreateOrUpdate phase.
	if reflect.ValueOf(pool).IsNil() {
		return phases.Next()
	}

	// copy the machine pool object from ocm into a
	// new object.
	req.Current = &ocmv1alpha1.MachinePool{}
	if req.Original.Status.Hosted {
		nodePool, ok := pool.(*clustersmgmtv1.NodePool)
		if !ok {
			return requeue.OnError(req, ocm.ErrConvertNodePool)
		}

		err = req.Current.CopyFromNodePool(nodePool, req.Desired.Spec.ClusterName)
	} else {
		machinePool, ok := pool.(*clustersmgmtv1.MachinePool)
		if !ok {
			return requeue.OnError(req, ocm.ErrConvertMachinePool)
		}

		err = req.Current.CopyFromMachinePool(machinePool, req.Desired.Spec.ClusterName)
	}

	if err != nil {
		return requeue.OnError(req, errMachinePoolCopy(req, err))
	}

	// ensure that we have the required labels for the machine pool
	// we found.  we do this to ensure we are not managing something that
	// may have been created by another process.
	if !req.Current.HasManagedLabels() {
		return requeue.OnError(req, errMachinePoolManagedLabels(req, ErrMachinePoolReservedLabel))
	}

	return phases.Next()
}

// Apply will create an OpenShift Cluster Manager machine pool if it does not exist,
// or update an OpenShift Cluster Manager machine pool if it does exist.
//
//nolint:forcetypeassert
func (r *Controller) Apply(req *MachinePoolRequest) (ctrl.Result, error) {
	// return if it is already in its desired state
	if req.desired() {
		r.Log.V(controllers.LogLevelDebug).Info(
			"machine pool already in desired state",
			request.LogValues(req),
		)

		return phases.Next()
	}

	// get the client
	var poolClient interface{}

	if req.Original.Status.Hosted {
		poolClient = ocm.NewNodePoolClient(
			r.Connection,
			req.Desired.Spec.DisplayName,
			req.Original.Status.ClusterID,
		)
	} else {
		poolClient = ocm.NewMachinePoolClient(
			r.Connection,
			req.Desired.Spec.DisplayName,
			req.Original.Status.ClusterID,
		)
	}

	// build the request
	req.Desired.Status.AvailabilityZones = req.Original.Status.AvailabilityZones
	req.Desired.Status.Subnets = req.Original.Status.Subnets

	// if no machine pool exists, create it and return
	//nolint:nestif
	if req.Current == nil {
		var createErr error

		r.Log.Info("creating machine pool", request.LogValues(req)...)
		if req.Original.Status.Hosted {
			createErr = req.createNodePool(poolClient.(*ocm.NodePoolClient))
		} else {
			createErr = req.createMachinePool(poolClient.(*ocm.MachinePoolClient))
		}

		if createErr != nil {
			// if the cluster with same name is deleting, requeue without an error
			if strings.Contains(createErr.Error(), "is being deleted from cluster") {
				r.Log.Info(
					"machine pool with same name is deleting; requeueing",
					request.LogValues(req)...,
				)

				return requeue.Retry(req)
			}

			return requeue.OnError(req, ocm.CreateError(req, createErr))
		}

		// create an event indicating that the machine pool has been created
		events.RegisterAction(events.Created, req.Original, r.Recorder, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)

		return phases.Next()
	}

	// update the object
	var updateErr error

	r.Log.Info("updating machine pool", request.LogValues(req)...)
	if req.Original.Status.Hosted {
		updateErr = req.updateNodePool(poolClient.(*ocm.NodePoolClient))
	} else {
		updateErr = req.updateMachinePool(poolClient.(*ocm.MachinePoolClient))
	}

	if updateErr != nil {
		return requeue.OnError(req, ocm.UpdateError(req, updateErr))
	}

	// create an event indicating that the machine pool has been updated
	events.RegisterAction(events.Updated, req.Original, r.Recorder, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)

	return phases.Next()
}

// Destroy will destroy an OpenShift Cluster Manager machine pool.
//
//nolint:forcetypeassert
func (r *Controller) Destroy(req *MachinePoolRequest) (ctrl.Result, error) {
	// return immediately if we have already deleted the machine pool
	if conditions.IsSet(MachinePoolDeleted(), req.Original) {
		return phases.Next()
	}

	// return if the cluster does not exist (has been deleted)
	_, exists, err := ocm.ClusterExists(req.Desired.Spec.ClusterName, req.Reconciler.Connection)
	if err != nil {
		return requeue.OnError(req, err)
	}

	if !exists {
		return phases.Next()
	}

	// get the client
	var poolClient interface{}

	if req.Original.Status.Hosted {
		poolClient = ocm.NewNodePoolClient(
			r.Connection,
			req.Desired.Spec.DisplayName,
			req.Original.Status.ClusterID,
		)
	} else {
		poolClient = ocm.NewMachinePoolClient(
			r.Connection,
			req.Desired.Spec.DisplayName,
			req.Original.Status.ClusterID,
		)
	}

	// delete the object
	var deleteErr error

	r.Log.Info("deleting machine pool", request.LogValues(req)...)
	if req.Original.Status.Hosted {
		deleteErr = req.deleteNodePool(poolClient.(*ocm.NodePoolClient))
	} else {
		deleteErr = req.deleteMachinePool(poolClient.(*ocm.MachinePoolClient))
	}

	if deleteErr != nil {
		return requeue.OnError(req, ocm.DeleteError(req, deleteErr))
	}

	// create an event indicating that the machine pool has been deleted
	events.RegisterAction(events.Deleted, req.Original, r.Recorder, req.Desired.Spec.DisplayName, req.Original.Status.ClusterID)

	// set the deleted condition
	if err := conditions.Update(req, MachinePoolDeleted()); err != nil {
		return requeue.OnError(req, conditions.UpdateDeletedConditionError(err))
	}

	return phases.Next()
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
func (r *Controller) WaitUntilReady(req *MachinePoolRequest) (ctrl.Result, error) {
	// skip the wait check if we are not requesting to wait for readiness
	if !req.Original.Spec.Wait {
		return phases.Next()
	}

	nodes, err := kubernetes.GetLabeledNodes(req.Context, r, req.Desired.Spec.Labels)
	if err != nil {
		return requeue.OnError(req, (errGetMachinePoolLabels(req, err)))
	}

	// ensure all nodes are ready
	if !kubernetes.NodesAreReady(nodes.Items...) {
		return requeue.Retry(req)
	}

	r.Log.Info("nodes are ready", request.LogValues(req)...)

	return phases.Next()
}

// WaitUntilMissing will requeue until the reconciler determines that the nodes
// no longer exist in the cluster.
func (r *Controller) WaitUntilMissing(req *MachinePoolRequest) (ctrl.Result, error) {
	// skip the wait check if we are not requesting to wait for readiness
	if !req.Original.Spec.Wait {
		return phases.Next()
	}

	nodes, err := kubernetes.GetLabeledNodes(req.Context, r, req.Desired.Spec.Labels)
	if err != nil {
		return requeue.OnError(req, (errGetMachinePoolLabels(req, err)))
	}

	// requeue if we cannot find any nodes
	if len(nodes.Items) > 0 {
		return requeue.Retry(req)
	}

	r.Log.Info("nodes have been removed", request.LogValues(req)...)

	return phases.Next()
}
