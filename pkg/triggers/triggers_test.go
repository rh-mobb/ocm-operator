package triggers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
)

func TestGetTrigger(t *testing.T) {
	t.Parallel()

	now := metav1.Now()

	type args struct {
		object client.Object
	}

	tests := []struct {
		name string
		args args
		want Trigger
	}{
		{
			name: "ensure create trigger returns appropriately",
			args: args{
				object: &ocmv1alpha1.MachinePool{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
			want: Create,
		},
		{
			name: "ensure delete trigger returns appropriately",
			args: args{
				object: &ocmv1alpha1.MachinePool{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: now,
						DeletionTimestamp: &now,
					},
				},
			},
			want: Delete,
		},
		{
			name: "ensure update trigger returns appropriately",
			args: args{
				object: &ocmv1alpha1.MachinePool{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: now,
					},
				},
			},
			want: Update,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GetTrigger(tt.args.object); got != tt.want {
				t.Errorf("GetTrigger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrigger_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{
			name:    "ensure create string",
			trigger: Create,
			want:    CreateString,
		},
		{
			name:    "ensure update string",
			trigger: Update,
			want:    UpdateString,
		},
		{
			name:    "ensure delete string",
			trigger: Delete,
			want:    DeleteString,
		},
		{
			name:    "ensure unknown string",
			trigger: Unknown,
			want:    UnknownString,
		},
		{
			name:    "ensure requeue string",
			trigger: Requeue,
			want:    RequeueString,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.trigger.String(); got != tt.want {
				t.Errorf("Trigger.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
