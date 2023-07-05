package conditions

import (
	"errors"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

const (
	conditionTypeReconciling               = "Reconciling"
	conditionTypeUpstreamClusterExists     = "UpstreamClusterExists"
	conditionMessageReconcilingStart       = "beginning reconciliation"
	conditionMessageReconcilingStop        = "ending reconciliation"
	conditionMessageUpstreamClusterExists  = "upstream cluster exists"
	conditionMessageUpstreamClusterMissing = "upstream cluster is missing"
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
		Reason:             trigger.String(),
		Message:            conditionMessageReconcilingStop,
	}
}

// UpstreamCluster returns a condition that gives the status of the upstream cluster.
func UpstreamCluster(trigger triggers.Trigger, exists bool) *metav1.Condition {
	var message string

	var status metav1.ConditionStatus

	if exists {
		message = conditionMessageUpstreamClusterExists
		status = metav1.ConditionTrue
	} else {
		message = conditionMessageUpstreamClusterMissing
		status = metav1.ConditionFalse
	}

	return &metav1.Condition{
		Type:               conditionTypeUpstreamClusterExists,
		LastTransitionTime: metav1.Now(),
		Status:             status,
		Reason:             trigger.String(),
		Message:            message,
	}
}

// Update updates the conditions on a workload.
func Update(req request.Request, condition *metav1.Condition) error {
	// return if we already have the condition set
	if IsSet(condition, req.GetObject()) {
		return nil
	}

	// create a copy of the original and convert to a client object
	original, ok := req.GetObject().DeepCopyObject().(client.Object)
	if !ok {
		return ErrConvertClientObject
	}

	// set the new condition
	req.GetObject().SetConditions(addCondition(req.GetObject().GetConditions(), condition))

	// run the patch
	return kubernetes.PatchStatus(req.GetContext(), req.GetReconciler(), original, req.GetObject())
}

// IsSet determines if a workload has a condition already set.
func IsSet(condition *metav1.Condition, on workload.Workload) bool {
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
