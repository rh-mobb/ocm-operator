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

type machinePoolClient struct {
	name       string
	connection *clustersmgmtv1.MachinePoolsClient
}

func NewMachinePoolClient(connection *sdk.Connection, name, clusterID string) *machinePoolClient {
	return &machinePoolClient{
		name:       name,
		connection: connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).MachinePools(),
	}
}

func (mpc *machinePoolClient) For(machinePoolName string) *clustersmgmtv1.MachinePoolClient {
	return mpc.connection.MachinePool(machinePoolName)
}

func (mpc *machinePoolClient) Get() (machinePool *clustersmgmtv1.MachinePool, err error) {
	// retrieve the machine pool from ocm
	response, err := mpc.For(mpc.name).Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return machinePool, nil
		}

		return machinePool, fmt.Errorf("error in get request - %w", err)
	}

	return response.Body(), nil
}

func (mpc *machinePoolClient) Create(builder *clustersmgmtv1.MachinePoolBuilder) (machinePool *clustersmgmtv1.MachinePool, err error) {
	// build the object to create
	object, err := builder.Build()
	if err != nil {
		return machinePool, fmt.Errorf("unable to build object for machine pool creation - %w", err)
	}

	// create the machine pool in ocm
	response, err := mpc.connection.Add().Body(object).Send()
	if err != nil {
		return machinePool, fmt.Errorf("error in create request - %w", err)
	}

	return response.Body(), nil
}

func (mpc *machinePoolClient) Update(builder *clustersmgmtv1.MachinePoolBuilder) (machinePool *clustersmgmtv1.MachinePool, err error) {
	// build the object to update
	object, err := builder.Build()
	if err != nil {
		return machinePool, fmt.Errorf("unable to build object for machine pool update - %w", err)
	}

	// update the machine pool in ocm
	response, err := mpc.For(object.ID()).Update().Body(object).Send()
	if err != nil {
		return machinePool, fmt.Errorf("error in update request - %w", err)
	}

	return response.Body(), nil
}

func (mpc *machinePoolClient) Delete(id string) error {
	// delete the machine pool in ocm
	response, err := mpc.For(id).Delete().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error in delete request - %w", err)
	}

	return nil
}
