package conditions

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_addCondition(t *testing.T) {
	t.Parallel()

	testCondition := &metav1.Condition{
		Type:               "Test",
		Reason:             "TestReason",
		Status:             metav1.ConditionTrue,
		Message:            "test message",
		LastTransitionTime: metav1.Now(),
	}

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
				new:     testCondition,
			},
			want: []metav1.Condition{*testCondition},
		},
		{
			name: "ensure existing conditions is not added",
			args: args{
				current: []metav1.Condition{*testCondition},
				new:     testCondition,
			},
			want: []metav1.Condition{*testCondition},
		},
		{
			name: "ensure new condition is added",
			args: args{
				current: []metav1.Condition{*testCondition},
				new:     &metav1.Condition{Type: "Empty"},
			},
			want: []metav1.Condition{*testCondition, {Type: "Empty"}},
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
