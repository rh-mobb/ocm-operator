package ocm

import (
	"errors"
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ErrClusterResponse = errors.New("invalid cluster response")
)

type ClusterClient struct {
	Name       string
	Connection *clustersmgmtv1.ClustersClient
}

func NewClusterClient(connection *sdk.Connection, name string) *ClusterClient {
	return &ClusterClient{
		Name:       name,
		Connection: connection.ClustersMgmt().V1().Clusters(),
	}
}

func (cc *ClusterClient) For(id string) *clustersmgmtv1.ClusterClient {
	return cc.Connection.Cluster(id)
}

func (cc *ClusterClient) Get() (cluster *clustersmgmtv1.Cluster, err error) {
	// retrieve the cluster from openshift cluster manager
	clusterList, err := cc.Connection.List().Search(fmt.Sprintf("name = '%s'", cc.Name)).Send()
	if err != nil {
		return cluster, fmt.Errorf("unable to retrieve cluster from openshift cluster manager - %w", err)
	}

	if len(clusterList.Items().Slice()) == 0 {
		return cluster, nil
	}

	return clusterList.Items().Slice()[0], nil
}

func (cc *ClusterClient) Create(
	builder *clustersmgmtv1.ClusterBuilder,
) (cluster *clustersmgmtv1.Cluster, err error) {
	// build the object to create
	object, err := builder.Build()
	if err != nil {
		return cluster, fmt.Errorf("unable to build object for cluster creation - %w", err)
	}

	// create the cluster in ocm
	response, err := cc.Connection.Add().Body(object).Send()
	if err != nil {
		return cluster, fmt.Errorf("error in create request - %w", err)
	}

	return response.Body(), nil
}

func (cc *ClusterClient) Update(
	builder *clustersmgmtv1.ClusterBuilder,
) (cluster *clustersmgmtv1.Cluster, err error) {
	// build the object to update
	object, err := builder.Build()
	if err != nil {
		return cluster, fmt.Errorf("unable to build object for cluster update - %w", err)
	}

	// update the cluster in ocm
	response, err := cc.For(object.ID()).Update().Body(object).Send()
	if err != nil {
		return cluster, fmt.Errorf("error in update request - %w", err)
	}

	return response.Body(), nil
}

func (cc *ClusterClient) Delete(id string) error {
	// delete the cluster in ocm
	response, err := cc.For(id).Delete().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error in delete request - %w", err)
	}

	return nil
}
