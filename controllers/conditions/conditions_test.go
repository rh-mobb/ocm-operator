package conditions

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/internal/factory"
)

func testConditionCluster(at metav1.Time, exists bool) *metav1.Condition {
	condition := UpstreamCluster(triggers.Create, exists)

	condition.LastTransitionTime = at

	return condition
}

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
				new:     testConditionCluster(now, false),
			},
			want: []metav1.Condition{*testConditionCluster(now, false)},
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
				current: []metav1.Condition{*testConditionCluster(now, true)},
				new:     testConditionReconciling(now),
			},
			want: []metav1.Condition{*testConditionCluster(now, true), *testConditionReconciling(now)},
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

	type args struct {
		req       request.Request
		condition *metav1.Condition
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ensure condition already set is not updated",
			args: args{
				req:       factory.NewTestRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
				condition: factory.TestCondition(metav1.Now()),
			},
			wantErr: false,
		},
		{
			name: "ensure condition not set is updated",
			args: args{
				req:       factory.NewTestRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
				condition: testConditionReconciling(now),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := Update(tt.args.req, tt.args.condition); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
