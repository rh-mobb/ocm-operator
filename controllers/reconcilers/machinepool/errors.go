package machinepool

import (
	"fmt"

	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

var (
	ErrMachinePoolNameLength    = fmt.Errorf("machine pool name exceeds maximum length of %d characters", maximumNameLength)
	ErrMachinePoolReservedLabel = fmt.Errorf("problem with system reserved labels: %+v", ocm.ManagedLabels())
)

// errMachinePoolCopy is an error indicating that the MachinePool object was unable to be
// copied.
func errMachinePoolCopy(request *MachinePoolRequest, err error) error {
	return fmt.Errorf("unable to copy ocm machine pool object [%s] - %w", request.GetName(), err)
}

// errMachinePoolManagedLabels is an error indicating that the MachinePool object does not
// have the appropriate managed labels.
func errMachinePoolManagedLabels(request *MachinePoolRequest, err error) error {
	return fmt.Errorf(
		"machine pool [%s] is missing managed labels [%+v] - %w",
		request.GetName(),
		request.Current.Spec.Labels,
		ErrMachinePoolReservedLabel,
	)
}

// errGetMachinePoolLabels is an error indicating that the MachinePool labels were unable to
// be retrieved.
func errGetMachinePoolLabels(request *MachinePoolRequest, err error) error {
	return fmt.Errorf("unable to get labeled nodes for machine pool [%s] - %w", request.GetName(), err)
}
