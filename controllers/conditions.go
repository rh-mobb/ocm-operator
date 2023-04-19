package controllers

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	conditionTypeReconciling = "Reconciling"
)

func conditionReconciling(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               conditionTypeReconciling,
		LastTransitionTime: metav1.Now(),
		Status:             status,
		Reason:             reason,
		Message:            message,
	}
}

func addCondition(current []metav1.Condition, new metav1.Condition) []metav1.Condition {
	for condition := range current {
		if current[condition].Type == new.Type {
			if equalCondition(current[condition], new) {
				continue
			}

			current[condition] = new
		}
	}

	return current
}

func hasCondition(current []metav1.Condition, new metav1.Condition) bool {
	for condition := range current {
		if equalCondition(current[condition], new) {
			return true
		}
	}

	return false
}

func equalCondition(current metav1.Condition, new metav1.Condition) bool {
	// ignore the last transition time
	current.LastTransitionTime = new.LastTransitionTime

	return reflect.DeepEqual(current, new)
}
