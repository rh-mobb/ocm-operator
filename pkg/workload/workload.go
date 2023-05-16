package workload

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Workload represents the actual object that is being reconciled.
type Workload interface {
	client.Object

	GetConditions() []metav1.Condition
	SetConditions([]metav1.Condition)
}
