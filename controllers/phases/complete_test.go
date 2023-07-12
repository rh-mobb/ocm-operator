package phases

import (
	"reflect"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/internal/factory"
)

func TestComplete(t *testing.T) {
	t.Parallel()

	type args struct {
		req        request.Request
		trigger    triggers.Trigger
		controller controllers.Controller
	}
	tests := []struct {
		name    string
		args    args
		want    ctrl.Result
		wantErr bool
	}{
		{
			name: "ensure bad request fails on create",
			args: args{
				trigger:    triggers.Create,
				controller: factory.NewTestController(),
				req:        factory.NewTestErrorRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: factory.DefaultRequeue},
			wantErr: true,
		},
		{
			name: "ensure good request succeeds on create",
			args: args{
				trigger:    triggers.Create,
				controller: factory.NewTestController(),
				req:        factory.NewTestRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: factory.DefaultRequeue},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Complete(tt.args.req, tt.args.trigger, tt.args.controller)
			if (err != nil) != tt.wantErr {
				t.Errorf("Complete() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Complete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompleteDestroy(t *testing.T) {
	t.Parallel()

	type args struct {
		req        request.Request
		controller controllers.Controller
	}
	tests := []struct {
		name    string
		args    args
		want    ctrl.Result
		wantErr bool
	}{
		{
			name: "ensure bad request fails",
			args: args{
				controller: factory.NewTestController(),
				req:        factory.NewTestErrorRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: factory.DefaultRequeue},
			wantErr: true,
		},
		{
			name: "ensure good request succeeds",
			args: args{
				controller: factory.NewTestController(),
				req:        factory.NewTestRequest(0, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: false},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := CompleteDestroy(tt.args.req, tt.args.controller)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteDestroy() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompleteDestroy() = %v, want %v", got, tt.want)
			}
		})
	}
}
