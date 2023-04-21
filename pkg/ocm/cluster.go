package ocm

import (
	"errors"
	"fmt"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ErrClusterResponse = errors.New("invalid cluster response")
)

type clusterClient struct {
	Name       string
	Connection *clustersmgmtv1.ClustersClient
}

func NewClusterClient(connection *sdk.Connection, name string) *clusterClient {
	return &clusterClient{
		Name:       name,
		Connection: connection.ClustersMgmt().V1().Clusters(),
	}
}

func (cc *clusterClient) Get() (cluster *clustersmgmtv1.Cluster, err error) {
	// retrieve the cluster from openshift cluster manager
	clusterList, err := cc.Connection.List().Search(fmt.Sprintf("name = '%s'", cc.Name)).Send()
	if err != nil {
		return cluster, fmt.Errorf("unable to retrieve cluster from openshift cluster manager - %w", err)
	}

	// return an error if we did not find exactly 1 cluster
	if len(clusterList.Items().Slice()) != 1 {
		return cluster, fmt.Errorf(
			"expected 1 cluster with name [%s] but found [%d] - %w",
			cc.Name,
			len(clusterList.Items().Slice()),
			ErrClusterResponse,
		)
	}

	return clusterList.Items().Slice()[0], nil
}
