package controllers

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/phases"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	defaultMissingUpstreamRequeue = 60 * time.Second
)

// HandleClusterPhase is the common phase that handles the upstream cluster for a child request.  It
// should be called with a wrapper function in order to satisfy the Phase.Function field.
func HandleClusterPhase(
	request request.Cluster,
	connection *sdk.Connection,
	trigger triggers.Trigger,
	logger logr.Logger,
) (ctrl.Result, error) {
	cluster, err := HandleUpstreamCluster(request, ocm.NewClusterClient(connection, request.GetClusterName()))
	if err != nil {
		return requeue.After(defaultMissingUpstreamRequeue, fmt.Errorf(
			"unable to handle upstream cluster: [%s] - %w",
			request.GetClusterName(),
			err,
		))
	}

	clusterExists := (cluster != nil)

	// set condition for a missing or existing cluster
	if err := conditions.Update(
		request,
		conditions.UpstreamCluster(trigger, clusterExists),
	); err != nil {
		return requeue.After(defaultMissingUpstreamRequeue, fmt.Errorf(
			"unable to update status on cluster: [%s] - %w",
			request.GetClusterName(),
			err,
		))
	}

	// return if the cluster exists
	if !clusterExists {
		logger.Info(fmt.Sprintf("cluster [%s] does not exist...requeueing", request.GetClusterName()), LogValues(request)...)

		return requeue.After(defaultMissingUpstreamRequeue, nil)
	}

	// return if the cluster is not ready
	if cluster.State() != clustersmgmtv1.ClusterStateReady {
		logger.Info(
			fmt.Sprintf(
				"cluster [%s] with state [%s] is not ready...requeueing",
				request.GetClusterName(),
				cluster.State(),
			), LogValues(request)...)

		return requeue.After(defaultMissingUpstreamRequeue, nil)
	}

	return phases.Next()
}
