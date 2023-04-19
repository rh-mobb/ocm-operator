package ocm

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	clusterContextPrefix = "clusterID"
)

type clusterClient struct {
	Name       string
	Connection *clustersmgmtv1.ClustersClient
	Object     *clustersmgmtv1.Cluster
}

func NewClusterClient(connection *sdk.Connection, name string) *clusterClient {
	return &clusterClient{
		Name:       name,
		Connection: connection.ClustersMgmt().V1().Clusters(),
	}
}

func (cc *clusterClient) Get() error {
	// retrieve the cluster from openshift cluster manager
	clusterList, err := cc.Connection.List().Search(fmt.Sprintf("name = '%s'", cc.Name)).Send()
	if err != nil {
		return fmt.Errorf("unable to retrieve cluster from openshift cluster manager - %w", err)
	}

	// return an error if we did not find exactly 1 cluster
	if len(clusterList.Items().Slice()) != 1 {
		return fmt.Errorf(
			"expected 1 cluster with name [%s] but found [%d]",
			cc.Name,
			len(clusterList.Items().Slice()),
		)
	}

	cc.Object = clusterList.Items().Slice()[0]

	return nil
}

// ClusterIDFromContext attempts to pull the cluster ID from a context to attempt to
// cut down on the number of API calls needed for OCM.
func ClusterIDFromContext(ctx context.Context, request ctrl.Request) string {
	value := ctx.Value(ClusterIDContextKey(request))

	if value == nil {
		return ""
	}

	return value.(string)
}

// ClusterIDContextKey returns the context key to use for a specific request.
func ClusterIDContextKey(request ctrl.Request) string {
	return fmt.Sprintf("%s.%s.%s", clusterContextPrefix, request.Namespace, request.Name)
}
