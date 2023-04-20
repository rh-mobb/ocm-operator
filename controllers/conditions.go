package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	conditionTypeReconciling = "Reconciling"
	conditionTypeDeleted     = "MachinePoolDeleted"

	messageDeleted = "machine pool has been deleted from openshift cluster manager"

	optimisticLockErrorMessage = "the object has been modified; please apply your changes to the latest version and try again"
)

func conditionReconciling(status metav1.ConditionStatus, trigger controllerTrigger, message string) metav1.Condition {
	return metav1.Condition{
		Type:               conditionTypeReconciling,
		LastTransitionTime: metav1.Now(),
		Status:             status,
		Reason:             trigger.String(),
		Message:            message,
	}
}

func conditionDeleted() metav1.Condition {
	return metav1.Condition{
		Type:               conditionTypeDeleted,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             triggerDeleteString,
		Message:            messageDeleted,
	}
}

func addCondition(current []metav1.Condition, new metav1.Condition) []metav1.Condition {
	if len(current) < 1 {
		return []metav1.Condition{new}
	}

	for condition := range current {
		if current[condition].Type == new.Type {
			if equalCondition(current[condition], new) {
				return current
			}

			current[condition] = new

			return current
		}
	}

	return append(current, new)
}

func updateCondition(
	ctx context.Context,
	reconciler kubernetes.Client,
	object Workload,
	condition metav1.Condition,
) error {
	// return if we already have the condition set
	if hasCondition(object.GetConditions(), condition) {
		return nil
	}

	// create a copy of the original and convert to a client object
	original, ok := object.DeepCopyObject().(client.Object)
	if !ok {
		return fmt.Errorf("unable to convert object to client.Object")
	}

	// set the new condition
	object.SetConditions(addCondition(object.GetConditions(), condition))

	// run the patch
	if err := reconciler.Status().Patch(ctx, object, client.MergeFrom(original)); err != nil {
		if isOptimisticLockError(err) {
			return nil
		}

		return fmt.Errorf("unable to update status conditions - %w", err)
	}

	return nil
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
	// ignore the last transition time and observed generation
	current.LastTransitionTime = new.LastTransitionTime
	current.ObservedGeneration = new.ObservedGeneration

	return reflect.DeepEqual(current, new)
}

func isOptimisticLockError(err error) bool {
	return strings.Contains(err.Error(), optimisticLockErrorMessage)
}
