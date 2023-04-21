package conditions

import (
	"context"
	"errors"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

const (
	conditionTypeReconciling         = "Reconciling"
	conditionMessageReconcilingStart = "beginning reconciliation"
	conditionMessageReconcilingStop  = "ending reconciliation"
)

var (
	ErrConvertClientObject = errors.New("unable to convert object to client.Object")
)

// Reconciling returns a reconciling condition based up on a trigger.  This
// is the condition that is set upon entry to reconciliation.
func Reconciling(trigger triggers.Trigger) *metav1.Condition {
	return &metav1.Condition{
		Type:               conditionTypeReconciling,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             trigger.String(),
		Message:            conditionMessageReconcilingStart,
	}
}

// Reconciled returns a reconciled condition based up on a trigger.  This
// is the condition that is set upon exit of reconciliation.
func Reconciled(trigger triggers.Trigger) *metav1.Condition {
	return &metav1.Condition{
		Type:               conditionTypeReconciling,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionFalse,
		Reason:             triggers.Delete.String(),
		Message:            conditionMessageReconcilingStop,
	}
}

// Update updates the conditions on a workload.
func Update(
	ctx context.Context,
	reconciler kubernetes.Client,
	object request.Workload,
	condition *metav1.Condition,
) error {
	// return if we already have the condition set
	if IsSet(condition, object) {
		return nil
	}

	// create a copy of the original and convert to a client object
	original, ok := object.DeepCopyObject().(client.Object)
	if !ok {
		return ErrConvertClientObject
	}

	// set the new condition
	object.SetConditions(addCondition(object.GetConditions(), condition))

	// run the patch
	//nolint:wrapcheck
	return kubernetes.PatchStatus(ctx, reconciler, original, object)
}

// IsSet determines if a workload has a condition already set.
func IsSet(condition *metav1.Condition, on request.Workload) bool {
	for _, existing := range on.GetConditions() {
		if equalCondition(*condition, existing) {
			return true
		}
	}

	return false
}

func addCondition(existing []metav1.Condition, newCondition *metav1.Condition) []metav1.Condition {
	if len(existing) < 1 {
		return []metav1.Condition{*newCondition}
	}

	for condition := range existing {
		if existing[condition].Type == newCondition.Type {
			if equalCondition(existing[condition], *newCondition) {
				return existing
			}

			existing[condition] = *newCondition

			return existing
		}
	}

	return append(existing, *newCondition)
}

// equalCondition determines if two conditions are equal.
//
//nolint:gocritic
func equalCondition(existing, newCondition metav1.Condition) bool {
	// ignore the last transition time and observed generation
	existing.LastTransitionTime = newCondition.LastTransitionTime
	existing.ObservedGeneration = newCondition.ObservedGeneration

	return reflect.DeepEqual(existing, newCondition)
}
