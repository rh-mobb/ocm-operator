package machinepool

import (
	corev1 "k8s.io/api/core/v1"
)

type Event int

const (
	EventUnknown Event = iota
	EventCreated
	EventUpdated
	EventDeleted
)

const (
	UnknownString = "Unknown"
	CreatedString = "Created"
	UpdatedString = "Updated"
	DeletedString = "Deleted"
)

// String returns the string value of a machine pool event.
func (event Event) String() string {
	return map[Event]string{
		EventUnknown: UnknownString,
		EventCreated: CreatedString,
		EventUpdated: UpdatedString,
		EventDeleted: DeletedString,
	}[event]
}

// Type returns the type of machine pool event.
func (event Event) Type() string {
	return map[Event]string{
		EventUnknown: UnknownString,
		EventCreated: corev1.EventTypeNormal,
		EventUpdated: corev1.EventTypeNormal,
		EventDeleted: corev1.EventTypeNormal,
	}[event]
}
