package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

const (
	machinePoolConditionTypeDeleted = "MachinePoolDeleted"
	machinePoolMessageDeleted       = "machine pool has been deleted from openshift cluster manager"
)

// MachinePoolDeleted return a condition indicating that the machine pool has
// been deleted from OpenShift Cluster Manager.
func MachinePoolDeleted() *metav1.Condition {
	return &metav1.Condition{
		Type:               machinePoolConditionTypeDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            machinePoolMessageDeleted,
	}
}
