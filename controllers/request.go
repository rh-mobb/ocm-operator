package controllers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/workload"
)

const (
	errRetrieveClusterMessage = "unable to retrieve cluster from ocm"
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
}

// ClusterChildRequest is similar to a request, but represents a request that
// is a child request and relies upon an existing cluster.
type ClusterChildRequest interface {
	Request

	GetClusterName() string
	GetContext() context.Context
	GetReconciler() kubernetes.Client
	SetClusterStatus(*clustersmgmtv1.Cluster)
}

// HandleUpstreamCluster finds the actual cluster from OCM and sets the relevant cluster status fields on
// the request.
//
//nolint:nestif
func HandleUpstreamCluster(request ClusterChildRequest, client *ocm.ClusterClient) (cluster *clustersmgmtv1.Cluster, err error) {
	// retrieve the cluster
	if request.GetObject().GetClusterID() == "" {
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
			return cluster, fmt.Errorf("%s: [%s] - %w", errRetrieveClusterMessage, client.Name, ErrMissingClusterID)
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

// LogValues returns a consistent set of values for a request.
func LogValues(request Request) []interface{} {
	object := request.GetObject()

	return []interface{}{
		"kind", object.GetObjectKind().GroupVersionKind().Kind,
		"resource", fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		"name", request.GetName(),
	}
}
