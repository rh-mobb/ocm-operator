package ocm

import (
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	LabelPrefixManaged = "ocm.mobb.redhat.com/managed"
	LabelPrefixName    = "ocm.mobb.redhat.com/name"
)

type MachinePoolClient struct {
	Name       string
	Connection *clustersmgmtv1.MachinePoolsClient
	Object     *clustersmgmtv1.MachinePool
}

func NewMachinePoolClient(connection *sdk.Connection, name, clusterID string) *MachinePoolClient {
	return &MachinePoolClient{
		Name:       name,
		Connection: connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).MachinePools(),
	}
}

func (mp *MachinePoolClient) For(machinePoolName string) *clustersmgmtv1.MachinePoolClient {
	return mp.Connection.MachinePool(mp.Name)
}

func (mp *MachinePoolClient) Get() error {
	// retrive the machine pool from ocm
	response, err := mp.For(mp.Name).Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("unable to retrieve machine pools from ocm - %w", err)
	}

	mp.Object = response.Body()

	return nil
}

func (mp *MachinePoolClient) Create(builder *clustersmgmtv1.MachinePoolBuilder) error {
	// build the object to create
	object, err := builder.Build()
	if err != nil {
		return fmt.Errorf("unable to build object for machine pool creation - %w", err)
	}

	// create the machine pool in ocm
	response, err := mp.Connection.Add().Body(object).Send()
	if err != nil {
		return fmt.Errorf("unable to create machine pool in ocm - %w", err)
	}

	mp.Object = response.Body()

	return nil
}
