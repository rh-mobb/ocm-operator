package factory

import (
	"context"
	"time"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

const (
	DefaultRequeue = 30 * time.Second
)

type testRequest struct {
	now     metav1.Time
	object  workload.Workload
	requeue time.Duration
}

type testErrorRequest struct {
	now     metav1.Time
	object  workload.Workload
	requeue time.Duration
}

func NewTestRequest(requeue time.Duration, workload workload.Workload) *testRequest {
	return &testRequest{
		now:     metav1.Now(),
		object:  workload,
		requeue: requeue,
	}
}

func NewTestErrorRequest(requeue time.Duration, workload workload.Workload) *testErrorRequest {
	return &testErrorRequest{
		now:     metav1.Now(),
		object:  workload,
		requeue: requeue,
	}
}

func (t *testRequest) DefaultRequeue() time.Duration            { return t.requeue }
func (t *testRequest) GetObject() workload.Workload             { return t.object }
func (t *testRequest) GetName() string                          { return "test" }
func (t *testRequest) GetContext() context.Context              { return context.TODO() }
func (t *testRequest) GetReconciler() kubernetes.Client         { return &kubernetes.FakeClient{} }
func (t *testRequest) GetClusterName() string                   { return "test" }
func (t *testRequest) SetClusterStatus(*clustersmgmtv1.Cluster) {}

func (t *testErrorRequest) DefaultRequeue() time.Duration            { return t.requeue }
func (t *testErrorRequest) GetObject() workload.Workload             { return t.object }
func (t *testErrorRequest) GetName() string                          { return "test" }
func (t *testErrorRequest) GetContext() context.Context              { return context.TODO() }
func (t *testErrorRequest) GetReconciler() kubernetes.Client         { return &kubernetes.FakeErrorClient{} }
func (t *testErrorRequest) GetClusterName() string                   { return "test" }
func (t *testErrorRequest) SetClusterStatus(*clustersmgmtv1.Cluster) {}
