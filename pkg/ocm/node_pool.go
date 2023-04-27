package ocm

import (
	"errors"
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ErrConvertNodePool = errors.New("error converting to node pool object")
)

// NodePoolClient represents the client used to interact with a Node Pool API object.  Node
// pools are associated with clusters that are using hosted control plane.
type NodePoolClient struct {
	name       string
	connection *clustersmgmtv1.NodePoolsClient
}

func NewNodePoolClient(connection *sdk.Connection, name, clusterID string) *NodePoolClient {
	return &NodePoolClient{
		name:       name,
		connection: connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).NodePools(),
	}
}

func (npc *NodePoolClient) For(nodePoolName string) *clustersmgmtv1.NodePoolClient {
	return npc.connection.NodePool(nodePoolName)
}

func (npc *NodePoolClient) Get() (nodePool *clustersmgmtv1.NodePool, err error) {
	// retrieve the node pool from ocm
	response, err := npc.For(npc.name).Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nodePool, nil
		}

		return nodePool, fmt.Errorf("error in get request - %w", err)
	}

	return response.Body(), nil
}

func (npc *NodePoolClient) Create(builder *clustersmgmtv1.NodePoolBuilder) (nodePool *clustersmgmtv1.NodePool, err error) {
	// build the object to create
	object, err := builder.Build()
	if err != nil {
		return nodePool, fmt.Errorf("unable to build object for node pool creation - %w", err)
	}

	// create the node pool in ocm
	response, err := npc.connection.Add().Body(object).Send()
	if err != nil {
		return nodePool, fmt.Errorf("error in create request - %w", err)
	}

	return response.Body(), nil
}

func (npc *NodePoolClient) Update(builder *clustersmgmtv1.NodePoolBuilder) (nodePool *clustersmgmtv1.NodePool, err error) {
	// build the object to update
	object, err := builder.Build()
	if err != nil {
		return nodePool, fmt.Errorf("unable to build object for node pool update - %w", err)
	}

	// update the node pool in ocm
	response, err := npc.For(object.ID()).Update().Body(object).Send()
	if err != nil {
		return nodePool, fmt.Errorf("error in update request - %w", err)
	}

	return response.Body(), nil
}

func (npc *NodePoolClient) Delete(id string) error {
	// delete the node pool in ocm
	response, err := npc.For(id).Delete().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error in delete request - %w", err)
	}

	return nil
}
