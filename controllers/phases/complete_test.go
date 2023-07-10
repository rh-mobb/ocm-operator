package phases

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/internal/factory"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestComplete(t *testing.T) {
	t.Parallel()

	type args struct {
		req     request.Request
		trigger triggers.Trigger
		log     logr.Logger
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
				trigger: triggers.Create,
				log:     ctrl.Log.WithName("bad-request-create"),
				req:     factory.NewTestErrorRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: factory.DefaultRequeue},
			wantErr: true,
		},
		{
			name: "ensure good request succeeds on create",
			args: args{
				trigger: triggers.Create,
				log:     ctrl.Log.WithName("good-request-create"),
				req:     factory.NewTestRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: factory.DefaultRequeue},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Complete(tt.args.req, tt.args.trigger, tt.args.log)
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
		req request.Request
		log logr.Logger
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
				log: ctrl.Log.WithName("bad-request"),
				req: factory.NewTestErrorRequest(factory.DefaultRequeue, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: factory.DefaultRequeue},
			wantErr: true,
		},
		{
			name: "ensure good request succeeds",
			args: args{
				log: ctrl.Log.WithName("good-request"),
				req: factory.NewTestRequest(0, factory.NewTestWorkload("")),
			},
			want:    ctrl.Result{Requeue: false},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := CompleteDestroy(tt.args.req, tt.args.log)
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
