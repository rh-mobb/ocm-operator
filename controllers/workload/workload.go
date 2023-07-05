package workload

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

// Workload represents the actual object that is being reconciled.
type Workload interface {
	client.Object

	GetClusterID() string
	GetConditions() []metav1.Condition
	SetConditions([]metav1.Condition)
}

// ClusterChild is a specialized workload that has a parent cluster.
type ClusterChild interface {
	Workload

	ExistsForClusterID(context.Context, kubernetes.Client, string) (bool, error)
}
