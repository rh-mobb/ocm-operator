package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/controllers/triggers"
)

const (
	identityProviderConditionTypeDeleted = "IdentityProviderDeleted"
	identityProviderMessageDeleted       = "identity provider has been deleted from openshift cluster manager"
)

// IdentityProviderDeleted return a condition indicating that the identity provider has
// been deleted from OpenShift Cluster Manager and the provider in which it is
// integrated with.
func IdentityProviderDeleted() *metav1.Condition {
	return &metav1.Condition{
		Type:               identityProviderConditionTypeDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            identityProviderMessageDeleted,
	}
}
