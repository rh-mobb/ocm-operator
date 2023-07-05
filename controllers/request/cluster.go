package request

import (
	"fmt"
	"net/http"
	"reflect"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
)

// Cluster is similar to a request, but represents a request that
// is a child request and the object being reconciled is related
// to an existing cluster.
type Cluster interface {
	Request

	GetClusterName() string
	SetClusterStatus(*clustersmgmtv1.Cluster)
}

// clusterFetcher is an interface that represents a client that
// fetches a cluster.  It is mostly used for testing purposes.
type clusterFetcher interface {
	Get() (*clustersmgmtv1.Cluster, error)
	For(string) *clustersmgmtv1.ClusterClient
}

// GetUpstreamCluster finds the actual cluster from OCM and sets the relevant cluster status fields on
// the request.
//
//nolint:nestif
func GetUpstreamCluster(request Cluster, client clusterFetcher) (cluster *clustersmgmtv1.Cluster, err error) {
	// retrieve the cluster
	if request.GetObject().GetClusterID() == "" {
		// retrieve the cluster from ocm
		cluster, err = client.Get()
		if err != nil {
			return cluster, fmt.Errorf("%s: [%s] - %w", errRetrieveClusterMessage, request.GetClusterName(), err)
		}

		if cluster == nil {
			return nil, nil
		}

		// if the cluster id is missing return an error
		if cluster.ID() == "" {
			return cluster, fmt.Errorf("%s: [%s] - %w", errRetrieveClusterMessage, request.GetClusterName(), ErrMissingClusterID)
		}
	} else {
		// retrieve the cluster from ocm by id
		response, err := client.For(request.GetObject().GetClusterID()).Get().Send()
		if err != nil {
			if response.Status() == http.StatusNotFound {
				return cluster, nil
			}
		}

		cluster = response.Body()
	}

	// keep track of the original object
	original := request.GetObject()

	// update the object
	request.SetClusterStatus(cluster)

	// if the original object and the current objects are equal, we do not require a
	// status update, so we can simply return the cluster.
	if reflect.DeepEqual(original, request.GetObject()) {
		return cluster, nil
	}

	// update the status containing the new values
	if err := kubernetes.PatchStatus(
		request.GetContext(),
		request.GetReconciler(),
		original,
		request.GetObject(),
	); err != nil {
		return cluster, fmt.Errorf(
			"unable to update status.clusterID=%s - %w",
			cluster.ID(),
			err,
		)
	}

	return cluster, nil
}
