package phases

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
)

const (
	defaultMissingUpstreamRequeue = 60 * time.Second
)

// HandleClusterPhase is the common phase that handles the upstream cluster for a child request.  It
// should be called with a wrapper function in order to satisfy the Phase.Function field.
func HandleClusterPhase(
	req request.Cluster,
	client request.ClusterFetcher,
	trigger triggers.Trigger,
	logger logr.Logger,
) (ctrl.Result, error) {
	cluster, err := request.GetUpstreamCluster(req, client)
	if err != nil {
		return requeue.After(defaultMissingUpstreamRequeue, fmt.Errorf(
			"unable to handle upstream cluster: [%s] - %w",
			req.GetClusterName(),
			err,
		))
	}

	clusterExists := (cluster != nil)

	// set condition for a missing or existing cluster
	if err := conditions.Update(
		req,
		conditions.UpstreamCluster(trigger, clusterExists),
	); err != nil {
		return requeue.After(defaultMissingUpstreamRequeue, fmt.Errorf(
			"unable to update status on cluster: [%s] - %w",
			req.GetClusterName(),
			err,
		))
	}

	// return if the cluster does not exist
	if !clusterExists {
		logger.Info(fmt.Sprintf("cluster [%s] does not exist...requeueing", req.GetClusterName()), request.LogValues(req)...)

		return requeue.After(defaultMissingUpstreamRequeue, nil)
	}

	// return if the cluster is not ready
	if cluster.State() != clustersmgmtv1.ClusterStateReady {
		logger.Info(
			fmt.Sprintf(
				"cluster [%s] with state [%s] is not ready...requeueing",
				req.GetClusterName(),
				cluster.State(),
			), request.LogValues(req)...)

		return requeue.After(defaultMissingUpstreamRequeue, nil)
	}

	return Next()
}
