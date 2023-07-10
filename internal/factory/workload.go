package factory

import (
	"context"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	DefaultClusterID = "test"
)

type testWorkloadStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	ClusterID  string
}

type testWorkload struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   testWorkloadSpec
	Status testWorkloadStatus
}

type testWorkloadSpec struct{}

func NewTestWorkload(clusterID string) *testWorkload {
	condition := TestCondition(metav1.Now())

	return &testWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"test.workload/finalizer"},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Test",
			APIVersion: "workload/v1",
		},
		Status: testWorkloadStatus{
			ClusterID: clusterID,
			Conditions: []metav1.Condition{
				*condition,
			},
		},
	}
}

func (t *testWorkload) DeepCopyObject() runtime.Object              { return t }
func (t *testWorkload) GetClusterID() string                        { return t.Status.ClusterID }
func (t *testWorkload) GetConditions() []metav1.Condition           { return t.Status.Conditions }
func (t *testWorkload) SetConditions(conditions []metav1.Condition) { t.Status.Conditions = conditions }
func (t *testWorkload) ExistsForClusterID(context.Context, kubernetes.Client, string) (bool, error) {
	return true, nil
}
