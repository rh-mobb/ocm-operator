package request

import (
	"context"
	"time"

	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

// Request represents a request that was sent to the controller that
// caused reconciliation.  It is used to track the status during the steps of
// controller reconciliation and pass information.  It should be able to
// return back the original object, in its pure form, that was discovered
// when the request was triggered.
type Request interface {
	DefaultRequeue() time.Duration
	GetObject() workload.Workload
	GetName() string
	GetContext() context.Context
	GetReconciler() kubernetes.Client
}
