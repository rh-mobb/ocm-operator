package rosacluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

const (
	rosaConditionTypeCreated      = "ROSAClusterCreated"
	rosaConditionTypeUpdated      = "ROSAClusterUpdated"
	rosaConditionTypeUninstalling = "ROSAClusterUninstalling"
	rosaConditionTypeDeleted      = "ROSAClusterDeleted"
	rosaMessageCreated            = "rosa cluster has been created"
	rosaMessageUpdated            = "rosa cluster has been updated"
	rosaMessageUninstalling       = "rosa cluster has been deleted from openshift cluster manager and is uninstalling"
	rosaMessageDeleted            = "rosa infrastructure has been deleted"

	awsConditionTypeOperatorRolesDeleted = "ROSAOperatorRolesDeleted"
	awsMessageOperatorRolesDeleted       = "operator roles have been deleted from aws"

	oidcConditionTypeConfigDeleted   = "OIDCConfigDeleted"
	oidcConditionTypeProviderDeleted = "OIDCProviderDeleted"
	oidcMessageConfigDeleted         = "oidc config has been deleted from ocm"
	oidcMessageProviderDeleted       = "oidc provider has been deleted from aws"
)

// ClusterCreated return a condition indicating that the ROSA Cluster has
// been created.
func ClusterCreated() *metav1.Condition {
	return &metav1.Condition{
		Type:               rosaConditionTypeCreated,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Create.String(),
		Message:            rosaMessageCreated,
	}
}

// ClusterUpdated return a condition indicating that the ROSA Cluster has
// been updated.
func ClusterUpdated() *metav1.Condition {
	return &metav1.Condition{
		Type:               rosaConditionTypeUpdated,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Update.String(),
		Message:            rosaMessageUpdated,
	}
}

// ClusterUninstalling return a condition indicating that the ROSA Cluster has
// been deleted from OpenShift Cluster Manager and is uninstalling.
func ClusterUninstalling() *metav1.Condition {
	return &metav1.Condition{
		Type:               rosaConditionTypeUninstalling,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            rosaMessageUninstalling,
	}
}

// ClusterDeleted return a condition indicating that the ROSA Cluster has
// been deleted entirely.
func ClusterDeleted() *metav1.Condition {
	return &metav1.Condition{
		Type:               rosaConditionTypeDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            rosaMessageDeleted,
	}
}

// OperatorRolesDeleted return a condition indicating that the ROSA operator
// roles have been deleted from AWS.
func OperatorRolesDeleted() *metav1.Condition {
	return &metav1.Condition{
		Type:               awsConditionTypeOperatorRolesDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            awsMessageOperatorRolesDeleted,
	}
}

// OIDCProviderDeleted return a condition indicating that the OIDC Provider
// has been deleted from AWS.
func OIDCProviderDeleted() *metav1.Condition {
	return &metav1.Condition{
		Type:               oidcConditionTypeProviderDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            oidcMessageProviderDeleted,
	}
}

// OIDCConfigDeleted return a condition indicating that the OIDC Configuration
// has been deleted from OCM.
func OIDCConfigDeleted() *metav1.Condition {
	return &metav1.Condition{
		Type:               oidcConditionTypeConfigDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggers.Delete.String(),
		Message:            oidcMessageConfigDeleted,
	}
}
