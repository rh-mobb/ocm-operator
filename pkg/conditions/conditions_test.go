package conditions

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/workload"
)

func testConditionReconciling(at metav1.Time) *metav1.Condition {
	condition := Reconciling(triggers.Create)

	condition.LastTransitionTime = at

	return condition
}

func testConditionReconciled(at metav1.Time) *metav1.Condition {
	condition := Reconciled(triggers.Create)

	condition.LastTransitionTime = at

	return condition
}

func testObject(t metav1.Time) *ocmv1alpha1.MachinePool {
	return &ocmv1alpha1.MachinePool{
		Status: ocmv1alpha1.MachinePoolStatus{
			Conditions: []metav1.Condition{
				*testConditionReconciled(t),
			},
		},
	}
}

func Test_addCondition(t *testing.T) {
	t.Parallel()

	now := metav1.Now()

	type args struct {
		current []metav1.Condition
		new     *metav1.Condition
	}

	tests := []struct {
		name string
		args args
		want []metav1.Condition
	}{
		{
			name: "ensure empty conditions adds the new condition",
			args: args{
				current: []metav1.Condition{},
				new:     testConditionReconciling(now),
			},
			want: []metav1.Condition{*testConditionReconciling(now)},
		},
		{
			name: "ensure existing conditions is not added",
			args: args{
				current: []metav1.Condition{*testConditionReconciled(now)},
				new:     testConditionReconciled(now),
			},
			want: []metav1.Condition{*testConditionReconciled(now)},
		},
		{
			name: "ensure new condition is added",
			args: args{
				current: []metav1.Condition{*testConditionReconciled(now)},
				new:     testConditionReconciling(now),
			},
			want: []metav1.Condition{*testConditionReconciled(now), *testConditionReconciling(now)},
		},
		{
			name: "ensure differing condition is replaced",
			args: args{
				current: []metav1.Condition{*testConditionReconciling(now)},
				new:     testConditionReconciled(now),
			},
			want: []metav1.Condition{*testConditionReconciled(now)},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := addCondition(tt.args.current, tt.args.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	now := metav1.Now()

	ctx := context.TODO()

	type args struct {
		ctx        context.Context
		reconciler kubernetes.Client
		object     workload.Workload
		condition  *metav1.Condition
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ensure condition already set is not updated",
			args: args{
				ctx:        ctx,
				reconciler: &kubernetes.FakeClient{},
				object:     testObject(now),
				condition:  testConditionReconciled(now),
			},
			wantErr: false,
		},
		{
			name: "ensure condition not set is updated",
			args: args{
				ctx:        ctx,
				reconciler: &kubernetes.FakeClient{},
				object:     testObject(now),
				condition:  testConditionReconciling(now),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := Update(tt.args.ctx, tt.args.reconciler, tt.args.object, tt.args.condition); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
