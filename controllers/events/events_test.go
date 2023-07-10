package events

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestEvent_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "ensure unknown event returns correct string",
			event: Unknown,
			want:  UnknownString,
		},
		{
			name:  "ensure created event returns correct string",
			event: Created,
			want:  CreatedString,
		},
		{
			name:  "ensure updated event returns correct string",
			event: Updated,
			want:  UpdatedString,
		},
		{
			name:  "ensure deleted event returns correct string",
			event: Deleted,
			want:  DeletedString,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.event.String(); got != tt.want {
				t.Errorf("Event.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_Type(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "ensure unknown event returns correct type",
			event: Unknown,
			want:  UnknownString,
		},
		{
			name:  "ensure created event returns correct type",
			event: Created,
			want:  corev1.EventTypeNormal,
		},
		{
			name:  "ensure updated event returns correct type",
			event: Updated,
			want:  corev1.EventTypeNormal,
		},
		{
			name:  "ensure deleted event returns correct type",
			event: Deleted,
			want:  corev1.EventTypeNormal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.event.Type(); got != tt.want {
				t.Errorf("Event.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
