package triggers

import (
	"errors"

	"github.com/rh-mobb/ocm-operator/controllers/request"
)

var (
	ErrTriggerUnknown = errors.New("unknown controller trigger")
)

type Trigger int

const (
	Unknown Trigger = iota
	Create
	Update
	Delete
	Requeue
)

const (
	triggerUnknownString = "Unknown"
	triggerCreateString  = "Create"
	triggerUpdateString  = "Update"
	triggerDeleteString  = "Delete"
	triggerRequeueString = "Requeue"
)

// GetTrigger returns the GetTrigger that caused the reconciliation event.
func GetTrigger(object request.Workload) Trigger {
	if object.GetCreationTimestamp().Time.IsZero() {
		return Create
	}

	if object.GetDeletionTimestamp() == nil {
		return Update
	}

	return Delete
}

// String returns the string value of a controller trigger.
func (trigger Trigger) String() string {
	return map[Trigger]string{
		Unknown: triggerUnknownString,
		Create:  triggerCreateString,
		Update:  triggerUpdateString,
		Delete:  triggerDeleteString,
		Requeue: triggerRequeueString,
	}[trigger]
}
