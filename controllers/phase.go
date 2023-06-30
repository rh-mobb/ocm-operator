package controllers

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

const (
	defaultMissingUpstreamRequeue = 60 * time.Second
)

// Phase defines an individual phase in the controller reconciliation process.
type Phase struct {
	Name     string
	Function func() (ctrl.Result, error)
}

// Execute executes the phases of controller reconciliation.
func Execute(request Request, req reconcile.Request, phases ...Phase) (ctrl.Result, error) {
	for execute := range phases {
		// run each phase function and return if we receive any errors
		result, err := phases[execute].Function()
		if err != nil || result.Requeue {
			return result, ReconcileError(
				req,
				fmt.Sprintf("%s phase reconciliation error", phases[execute].Name),
				err,
			)
		}
	}

	return NoRequeue(), nil
}

// HandleClusterPhase is the common phase that handles the upstream cluster for a child request.  It
// should be called with a wrapper function in order to satisfy the Phase.Function field.
func HandleClusterPhase(
	request ClusterChildRequest,
	connection *sdk.Connection,
	trigger triggers.Trigger,
	logger logr.Logger,
) (ctrl.Result, error) {
	cluster, err := HandleUpstreamCluster(request, ocm.NewClusterClient(connection, request.GetClusterName()))
	if err != nil {
		return RequeueAfter(defaultMissingUpstreamRequeue), fmt.Errorf(
			"unable to handle upstream cluster: [%s] - %w",
			request.GetClusterName(),
			err,
		)
	}

	clusterExists := (cluster != nil)

	// set condition for a missing or existing cluster
	if err := conditions.Update(
		request.GetContext(),
		request.GetReconciler(),
		request.GetObject(),
		conditions.UpstreamCluster(trigger, clusterExists),
	); err != nil {
		return RequeueAfter(defaultMissingUpstreamRequeue), fmt.Errorf(
			"unable to update status on cluster: [%s] - %w",
			request.GetClusterName(),
			err,
		)
	}

	// return if the cluster exists
	if !clusterExists {
		logger.Info(fmt.Sprintf("cluster [%s] does not exist...requeueing", request.GetClusterName()), LogValues(request)...)

		return RequeueAfter(defaultMissingUpstreamRequeue), nil
	}

	// return if the cluster is not ready
	if cluster.State() != clustersmgmtv1.ClusterStateReady {
		logger.Info(
			fmt.Sprintf(
				"cluster [%s] with state [%s] is not ready...requeueing",
				request.GetClusterName(),
				cluster.State(),
			), LogValues(request)...)

		return RequeueAfter(defaultMissingUpstreamRequeue), nil
	}

	return NoRequeue(), nil
}
