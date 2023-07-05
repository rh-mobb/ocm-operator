package phases

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers/requeue"
)

// NewPhase returns a new instance of a phase.
func NewPhase(name string, f func() (ctrl.Result, error)) phase {
	return phase{
		Name:     name,
		Function: f,
	}
}

// Phase defines an individual phase in the controller reconciliation process.
type phase struct {
	Name     string
	Function func() (ctrl.Result, error)
}

// // HandleClusterPhase is the common phase that handles the upstream cluster for a child request.  It
// // should be called with a wrapper function in order to satisfy the Phase.Function field.
// func HandleClusterPhase(
// 	request ClusterChildRequest,
// 	connection *sdk.Connection,
// 	trigger triggers.Trigger,
// 	logger logr.Logger,
// ) (ctrl.Result, error) {
// 	cluster, err := HandleUpstreamCluster(request, ocm.NewClusterClient(connection, request.GetClusterName()))
// 	if err != nil {
// 		return RequeueAfter(defaultMissingUpstreamRequeue), fmt.Errorf(
// 			"unable to handle upstream cluster: [%s] - %w",
// 			request.GetClusterName(),
// 			err,
// 		)
// 	}

// 	clusterExists := (cluster != nil)

// 	// set condition for a missing or existing cluster
// 	if err := conditions.Update(
// 		request.GetContext(),
// 		request.GetReconciler(),
// 		request.GetObject(),
// 		conditions.UpstreamCluster(trigger, clusterExists),
// 	); err != nil {
// 		return RequeueAfter(defaultMissingUpstreamRequeue), fmt.Errorf(
// 			"unable to update status on cluster: [%s] - %w",
// 			request.GetClusterName(),
// 			err,
// 		)
// 	}

// 	// return if the cluster exists
// 	if !clusterExists {
// 		logger.Info(fmt.Sprintf("cluster [%s] does not exist...requeueing", request.GetClusterName()), LogValues(request)...)

// 		return RequeueAfter(defaultMissingUpstreamRequeue), nil
// 	}

// 	// return if the cluster is not ready
// 	if cluster.State() != clustersmgmtv1.ClusterStateReady {
// 		logger.Info(
// 			fmt.Sprintf(
// 				"cluster [%s] with state [%s] is not ready...requeueing",
// 				request.GetClusterName(),
// 				cluster.State(),
// 			), LogValues(request)...)

// 		return RequeueAfter(defaultMissingUpstreamRequeue), nil
// 	}

// 	return NoRequeue(), nil
// }

// Next is a helper function for code readability to proceed to the next phase.
func Next() (ctrl.Result, error) {
	return requeue.Skip(nil)
}
