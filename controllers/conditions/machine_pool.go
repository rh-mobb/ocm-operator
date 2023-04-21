package conditions

import (
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	machinePoolConditionTypeDeleted = "MachinePoolDeleted"
	machinePoolMessageDeleted       = "machine pool has been deleted from openshift cluster manager"
)

// MachinePoolDeleted return a condition indicating that the machine pool has
// been deleted from OpenShift Cluster Manager.
func MachinePoolDeleted() metav1.Condition {
	return metav1.Condition{
		Type:               machinePoolConditionTypeDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            machinePoolMessageDeleted,
	}
}
