package factory

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

type testController struct {
	kubernetes.FakeClient

	logger logr.Logger
}

func NewTestController() *testController {
	return &testController{
		logger: ctrl.Log.WithName("test-controller"),
	}
}

func (t *testController) Log() logr.Logger                        { return t.logger }
func (t *testController) ReconcileInterval() time.Duration        { return DefaultRequeue }
func (t *testController) SetupWithManager(mgr ctrl.Manager) error { return nil }

func (t *testController) NewRequest(context.Context, ctrl.Request) (request.Request, error) {
	return nil, nil
}

func (t *testController) Reconcile(context.Context, ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (t *testController) ReconcileCreate(req request.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (t *testController) ReconcileUpdate(req request.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (t *testController) ReconcileDelete(req request.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
