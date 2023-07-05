package controllers

import (
	"fmt"
	"net/http"
	"reflect"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	errRetrieveClusterMessage = "unable to retrieve cluster from ocm"
)

// HandleUpstreamCluster finds the actual cluster from OCM and sets the relevant cluster status fields on
// the request.
//
//nolint:nestif
func HandleUpstreamCluster(req request.Cluster, client *ocm.ClusterClient) (cluster *clustersmgmtv1.Cluster, err error) {
	// retrieve the cluster
	if req.GetObject().GetClusterID() == "" {
		// retrieve the cluster from ocm
		cluster, err = client.Get()
		if err != nil {
			return cluster, fmt.Errorf("%s: [%s] - %w", errRetrieveClusterMessage, client.Name, err)
		}

		if cluster == nil {
			return nil, nil
		}

		// if the cluster id is missing return an error
		if cluster.ID() == "" {
			return cluster, fmt.Errorf("%s: [%s] - %w", errRetrieveClusterMessage, client.Name, request.ErrMissingClusterID)
		}
	} else {
		// retrieve the cluster from ocm by id
		response, err := client.For(req.GetObject().GetClusterID()).Get().Send()
		if err != nil {
			if response.Status() == http.StatusNotFound {
				return cluster, nil
			}
		}

		cluster = response.Body()
	}

	// keep track of the original object
	original := req.GetObject()

	// update the object
	req.SetClusterStatus(cluster)

	// if the original object and the current objects are equal, we do not require a
	// status update, so we can simply return the cluster.
	if reflect.DeepEqual(original, req.GetObject()) {
		return cluster, nil
	}

	// update the status containing the new values
	if err := kubernetes.PatchStatus(
		req.GetContext(),
		req.GetReconciler(),
		original,
		req.GetObject(),
	); err != nil {
		return cluster, fmt.Errorf(
			"unable to update status.clusterID=%s - %w",
			cluster.ID(),
			err,
		)
	}

	return cluster, nil
}

// LogValues returns a consistent set of values for a request.
func LogValues(request request.Request) []interface{} {
	object := request.GetObject()

	return []interface{}{
		"kind", object.GetObjectKind().GroupVersionKind().Kind,
		"resource", fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		"name", request.GetName(),
	}
}
