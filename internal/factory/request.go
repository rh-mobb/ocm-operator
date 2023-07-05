package factory

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

type testRequest struct {
	now    metav1.Time
	object workload.Workload
}

func NewTestRequest() *testRequest {
	return &testRequest{
		now:    metav1.Now(),
		object: NewTestWorkload(),
	}
}

func (t *testRequest) DefaultRequeue() time.Duration    { return 30 * time.Second }
func (t *testRequest) GetObject() workload.Workload     { return t.object }
func (t *testRequest) GetName() string                  { return "test" }
func (t *testRequest) GetContext() context.Context      { return context.TODO() }
func (t *testRequest) GetReconciler() kubernetes.Client { return &kubernetes.FakeClient{} }
