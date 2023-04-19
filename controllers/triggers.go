package controllers

import (
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrTriggerUnknown = errors.New("unknown controller trigger")
)

type controllerTrigger int

const (
	triggerUnknown controllerTrigger = iota
	triggerCreate
	triggerUpdate
	triggerDelete
)

const (
	triggerUnknownString = "Unknown"
	triggerCreateString  = "Create"
	triggerUpdateString  = "Update"
	triggerDeleteString  = "Delete"
)

// trigger returns the trigger that caused the reconciliation event.
func trigger(object client.Object) controllerTrigger {
	if object.GetCreationTimestamp().Time.IsZero() {
		return triggerCreate
	}

	if object.GetDeletionTimestamp() == nil {
		return triggerUpdate
	}

	return triggerDelete
}

// String returns the string value of a controller trigger.
func (trigger controllerTrigger) String() string {
	return map[controllerTrigger]string{
		triggerUnknown: triggerUnknownString,
		triggerCreate:  triggerCreateString,
		triggerUpdate:  triggerUpdateString,
		triggerDelete:  triggerDeleteString,
	}[trigger]
}
