package machinepool

import (
	"context"
	"fmt"
	"reflect"
	"time"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	maximumNameLength = 15
)

// MachinePoolRequest is an object that is unique to each reconciliation
// req.
//
// TODO: make this a generic request to be used across a variety of controllers
// which will likely require some fields to become interfaces.
type MachinePoolRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.MachinePool
	Original          *ocmv1alpha1.MachinePool
	Desired           *ocmv1alpha1.MachinePool
	Trigger           triggers.Trigger
	Reconciler        *Controller
}

func (r *Controller) NewRequest(ctx context.Context, ctrlReq ctrl.Request) (request.Request, error) {
	original := &ocmv1alpha1.MachinePool{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, ctrlReq.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &MachinePoolRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &MachinePoolRequest{}, err
	}

	// ensure the our managed labels do not conflict with what was submitted
	// to the cluster
	//
	// NOTE: this is implemented via CRD CEL validations, however leaving in
	// place for clusters that may not have this feature gate enabled as CEL
	// is in beta currently.
	if original.HasManagedLabels() {
		return &MachinePoolRequest{}, fmt.Errorf(
			"spec.labels cannot contain reserved labels [%+v] - %w",
			original.Spec.Labels,
			ErrMachinePoolReservedLabel,
		)
	}

	// ensure the name is less than 15 characters.  this is due to a limitation in the downstream
	// API.
	//
	// NOTE: this should be limited by CRD CEL language or some other validation in the CRD
	// but we can leave this in place as a secondary check.
	//
	// See https://github.com/rh-mobb/ocm-operator/issues/3
	desired := original.DesiredState()

	if len(desired.Spec.DisplayName) > maximumNameLength {
		return &MachinePoolRequest{}, fmt.Errorf(
			"requested name [%s] is invalid - %w",
			desired.Spec.DisplayName,
			ErrMachinePoolNameLength,
		)
	}

	// if we have a hosted control plane, ensure that we ignore the aws
	// configuration as it is invalid for a hosted control plane.  this is
	// only relevant so that the desired state does not drift and constantly
	// require an update.
	if desired.Status.Hosted {
		desired.Spec.AWS = ocmv1alpha1.MachinePoolProviderAWS{}
	}

	return &MachinePoolRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: ctrlReq,
		Context:           ctx,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
	}, nil
}

// DefaultRequeue returns the default requeue time for a request.
func (req *MachinePoolRequest) DefaultRequeue() time.Duration {
	return defaultMachinePoolRequeue
}

// GetObject returns the original object to satisfy the request.Request interface.
func (req *MachinePoolRequest) GetObject() workload.Workload {
	return req.Original
}

// GetName returns the name as it should appear in OCM.
func (req *MachinePoolRequest) GetName() string {
	return req.Desired.Spec.DisplayName
}

// GetClusterName returns the cluster name that this object belongs to.
func (req *MachinePoolRequest) GetClusterName() string {
	return req.Desired.Spec.ClusterName
}

// GetContext returns the context of the request.
func (req *MachinePoolRequest) GetContext() context.Context {
	return req.Context
}

// GetReconciler returns the context of the request.
func (req *MachinePoolRequest) GetReconciler() kubernetes.Client {
	return req.Reconciler
}

// SetClusterStatus sets the relevant cluster fields in the status.  It is used
// to satisfy the request.Request interface.
func (req *MachinePoolRequest) SetClusterStatus(cluster *clustersmgmtv1.Cluster) {
	if req.Original.Status.ClusterID == "" {
		req.Original.Status.ClusterID = cluster.ID()
	}

	if len(req.Original.Status.AvailabilityZones) == 0 {
		req.Original.Status.AvailabilityZones = cluster.Nodes().AvailabilityZones()
	}

	if len(req.Original.Status.Subnets) == 0 {
		req.Original.Status.Subnets = cluster.AWS().SubnetIDs()
	}

	req.Original.Status.Hosted = cluster.Hypershift().Enabled()
}

func (req *MachinePoolRequest) desired() bool {
	if req.Desired == nil || req.Current == nil {
		return false
	}

	// ignore the wait field as it is an internal field to the controller
	// and does not represent the desired state of the machine pool
	req.Current.Spec.Wait = req.Desired.Spec.Wait

	return reflect.DeepEqual(
		req.Desired.Spec,
		req.Current.Spec,
	)
}

// createMachinePool creates a machine pool object in OCM.
func (req *MachinePoolRequest) createMachinePool(poolClient *ocm.MachinePoolClient) error {
	if _, err := poolClient.Create(req.Desired.MachinePoolBuilder()); err != nil {
		return fmt.Errorf("unable to create machine pool - %w", err)
	}

	return nil
}

// createNodePool creates a node pool object in OCM (hosted control plane).
func (req *MachinePoolRequest) createNodePool(poolClient *ocm.NodePoolClient) error {
	if _, err := poolClient.Create(req.Desired.NodePoolBuilder()); err != nil {
		return fmt.Errorf("unable to create node pool - %w", err)
	}

	return nil
}

// updateMachinePool updates a machine pool object in OCM.
func (req *MachinePoolRequest) updateMachinePool(poolClient *ocm.MachinePoolClient) error {
	if _, err := poolClient.Update(req.Desired.MachinePoolBuilder()); err != nil {
		return fmt.Errorf("unable to update machine pool - %w", err)
	}

	return nil
}

// updateNodePool updates a node pool object in OCM.
func (req *MachinePoolRequest) updateNodePool(poolClient *ocm.NodePoolClient) error {
	if _, err := poolClient.Update(req.Desired.NodePoolBuilder()); err != nil {
		return fmt.Errorf("unable to update node pool - %w", err)
	}

	return nil
}

// deleteMachinePool deletes a machine pool object in OCM.
func (req *MachinePoolRequest) deleteMachinePool(poolClient *ocm.MachinePoolClient) error {
	if err := poolClient.Delete(req.Desired.Spec.DisplayName); err != nil {
		return fmt.Errorf("unable to update machine pool - %w", err)
	}

	return nil
}

// deleteNodePool deletes a node pool object in OCM.
func (req *MachinePoolRequest) deleteNodePool(poolClient *ocm.NodePoolClient) error {
	if err := poolClient.Delete(req.Desired.Spec.DisplayName); err != nil {
		return fmt.Errorf("unable to delete node pool - %w", err)
	}

	return nil
}
