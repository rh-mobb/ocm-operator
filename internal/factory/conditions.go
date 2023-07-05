package factory

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func TestCondition(at metav1.Time) *metav1.Condition {
	return &metav1.Condition{
		Type:               "TestCondition",
		Status:             metav1.ConditionTrue,
		Reason:             "Test",
		LastTransitionTime: at,
		Message:            "test message",
	}
}
