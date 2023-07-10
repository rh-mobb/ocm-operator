package phases

import (
	"errors"
	"reflect"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/internal/factory"
)

func Test_handler_Execute(t *testing.T) {
	t.Parallel()

	testRequest := factory.NewTestRequest(factory.DefaultRequeue, factory.NewTestWorkload(""))

	successPhase := NewPhase("success", Next)
	errorPhase := NewPhase("fail", func() (ctrl.Result, error) {
		return ctrl.Result{RequeueAfter: testRequest.DefaultRequeue()}, errors.New("fail")
	})

	errorHandler := NewHandler(testRequest, successPhase, errorPhase, successPhase)

	successHandler := NewHandler(testRequest, successPhase, successPhase, successPhase)

	type fields struct {
		Phases  []phase
		Request request.Request
	}
	tests := []struct {
		name    string
		fields  fields
		want    ctrl.Result
		wantErr bool
	}{
		{
			name: "ensure phase with error returns a requeue result with error",
			fields: fields{
				Request: errorHandler.Request,
				Phases:  errorHandler.Phases,
			},
			want:    ctrl.Result{Requeue: true, RequeueAfter: testRequest.DefaultRequeue()},
			wantErr: true,
		},
		{
			name: "ensure phase with no error returns a result without an error",
			fields: fields{
				Request: successHandler.Request,
				Phases:  successHandler.Phases,
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := &handler{
				Phases:  tt.fields.Phases,
				Request: tt.fields.Request,
			}
			got, err := handler.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("handler.Execute() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handler.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
