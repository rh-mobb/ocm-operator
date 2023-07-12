package request

import (
	"fmt"
	"net/http"
	"reflect"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

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

// ClusterFetcher is an interface that represents a client that
// fetches a cluster.  It is mostly used for testing purposes.
type ClusterFetcher interface {
	Get() (*clustersmgmtv1.Cluster, error)
	For(string) *clustersmgmtv1.ClusterClient
}

// GetUpstreamCluster finds the actual cluster from OCM and sets the relevant cluster status fields on
// the request.
//
//nolint:nestif
func GetUpstreamCluster(request Cluster, client ClusterFetcher) (cluster *clustersmgmtv1.Cluster, err error) {
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
		var response *clustersmgmtv1.ClusterGetResponse

		// retrieve the cluster from ocm by id
		response, err = client.For(request.GetObject().GetClusterID()).Get().Send()
		if err != nil {
			if response.Status() == http.StatusNotFound {
				return cluster, nil
			}
		}

		cluster = response.Body()
	}

	// keep track of the original object
	object := request.GetObject().DeepCopyObject()

	// convert the client object to a runtime object.  to do this,
	// we convert to an unstructured object which satisfies both a client.Object
	// and runtime.Object interface.
	// TODO: this is a little sloppy and needs some attention.
	objectMap, objectErr := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if objectErr != nil {
		return cluster, fmt.Errorf("unable to convert client.Object to runtime.Object - %w", err)
	}
	original := &unstructured.Unstructured{Object: objectMap}

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
