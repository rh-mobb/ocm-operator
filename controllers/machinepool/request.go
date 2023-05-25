package machinepool

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/workload"
)

const (
	maximumNameLength = 15
)

var (
	ErrMissingClusterID          = errors.New("unable to find cluster id")
	ErrMachinePoolRequestConvert = errors.New("unable to convert generic request to machine pool request")
	ErrMachinePoolNameLength     = fmt.Errorf("machine pool name exceeds maximum length of %d characters", maximumNameLength)
	ErrMachinePoolReservedLabel  = fmt.Errorf(
		"problem with system reserved labels: %s, %s",
		ocm.LabelPrefixManaged,
		ocm.LabelPrefixName,
	)
)

// MachinePoolRequest is an object that is unique to each reconciliation
// request.
//
// TODO: make this a generic request to be used across a variety of controllers
// which will likely require some fields to become interfaces.
type MachinePoolRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.MachinePool
	Original          *ocmv1alpha1.MachinePool
	Desired           *ocmv1alpha1.MachinePool
	Log               logr.Logger
	Trigger           triggers.Trigger
	Reconciler        *Controller
}

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
	original := &ocmv1alpha1.MachinePool{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
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
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.Log,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
	}, nil
}

// GetObject returns the original object to satisfy the controllers.Request interface.
func (request *MachinePoolRequest) GetObject() workload.Workload {
	return request.Original
}

// GetName returns the name as it should appear in OCM.
func (request *MachinePoolRequest) GetName() string {
	return request.Desired.Spec.DisplayName
}

func (request *MachinePoolRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
	)
}

// updateStatusCluster updates fields related to the cluster in which the machine pool resides in.
func (request *MachinePoolRequest) updateStatusCluster() error {
	// retrieve the cluster id
	clusterClient := ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.ClusterName)
	cluster, err := clusterClient.Get()
	if err != nil || cluster == nil {
		return fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			request.Desired.Spec.ClusterName,
			err,
		)
	}

	// if the cluster id is missing return an error
	if cluster.ID() == "" {
		return fmt.Errorf("missing cluster id in response - %w", ErrMissingClusterID)
	}

	// keep track of the original object
	original := request.Original.DeepCopy()
	request.Original.Status.ClusterID = cluster.ID()
	request.Original.Status.AvailabilityZones = cluster.Nodes().AvailabilityZones()
	request.Original.Status.Subnets = cluster.AWS().SubnetIDs()
	request.Original.Status.Hosted = cluster.Hypershift().Enabled()

	// store the cluster id in the status
	if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
		return fmt.Errorf(
			"unable to update status.clusterID=%s - %w",
			cluster.ID(),
			err,
		)
	}

	return nil
}

// createMachinePool creates a machine pool object in OCM.
func (request *MachinePoolRequest) createMachinePool(poolClient *ocm.MachinePoolClient) error {
	if _, err := poolClient.Create(request.Desired.MachinePoolBuilder()); err != nil {
		return fmt.Errorf("unable to create machine pool - %w", err)
	}

	return nil
}

// createNodePool creates a node pool object in OCM (hosted control plane).
func (request *MachinePoolRequest) createNodePool(poolClient *ocm.NodePoolClient) error {
	if _, err := poolClient.Create(request.Desired.NodePoolBuilder()); err != nil {
		return fmt.Errorf("unable to create node pool - %w", err)
	}

	return nil
}

// updateMachinePool updates a machine pool object in OCM.
func (request *MachinePoolRequest) updateMachinePool(poolClient *ocm.MachinePoolClient) error {
	if _, err := poolClient.Update(request.Desired.MachinePoolBuilder()); err != nil {
		return fmt.Errorf("unable to update machine pool - %w", err)
	}

	return nil
}

// updateNodePool updates a node pool object in OCM.
func (request *MachinePoolRequest) updateNodePool(poolClient *ocm.NodePoolClient) error {
	if _, err := poolClient.Update(request.Desired.NodePoolBuilder()); err != nil {
		return fmt.Errorf("unable to update node pool - %w", err)
	}

	return nil
}

// deleteMachinePool deletes a machine pool object in OCM.
func (request *MachinePoolRequest) deleteMachinePool(poolClient *ocm.MachinePoolClient) error {
	if err := poolClient.Delete(request.Desired.Spec.DisplayName); err != nil {
		return fmt.Errorf("unable to update machine pool - %w", err)
	}

	return nil
}

// deleteNodePool deletes a node pool object in OCM.
func (request *MachinePoolRequest) deleteNodePool(poolClient *ocm.NodePoolClient) error {
	if err := poolClient.Delete(request.Desired.Spec.DisplayName); err != nil {
		return fmt.Errorf("unable to delete node pool - %w", err)
	}

	return nil
}
