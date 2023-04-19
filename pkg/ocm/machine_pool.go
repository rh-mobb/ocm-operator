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
	Connection  *clustersmgmtv1.MachinePoolsClient
	MachinePool *machinePool
}

type machinePool struct {
	Name   string
	Object *clustersmgmtv1.MachinePool
}

func NewMachinePoolClient(connection *sdk.Connection, name, clusterID string) *MachinePoolClient {
	return &MachinePoolClient{
		Connection: connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).MachinePools(),
		MachinePool: &machinePool{
			Name: name,
		},
	}
}

func (mpc *MachinePoolClient) For(machinePoolName string) *clustersmgmtv1.MachinePoolClient {
	return mpc.Connection.MachinePool(machinePoolName)
}

func (mpc *MachinePoolClient) Get() error {
	// retrive the machine pool from ocm
	response, err := mpc.For(mpc.MachinePool.Name).Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("unable to retrieve machine pools from ocm - %w", err)
	}

	mpc.MachinePool.Object = response.Body()

	return nil
}

func (mpc *MachinePoolClient) Create(builder *clustersmgmtv1.MachinePoolBuilder) error {
	// build the object to create
	object, err := builder.Build()
	if err != nil {
		return fmt.Errorf("unable to build object for machine pool creation - %w", err)
	}

	// create the machine pool in ocm
	response, err := mpc.Connection.Add().Body(object).Send()
	if err != nil {
		return fmt.Errorf("unable to create machine pool in ocm - %w", err)
	}

	mpc.MachinePool.Object = response.Body()

	return nil
}

func (mpc *MachinePoolClient) Update(builder *clustersmgmtv1.MachinePoolBuilder) error {
	// build the object to update
	object, err := builder.Build()
	if err != nil {
		return fmt.Errorf("unable to build object for machine pool update - %w", err)
	}

	// update the machine pool in ocm
	response, err := mpc.For(object.ID()).Update().Send()
	if err != nil {
		return fmt.Errorf("unable to update machine pool in ocm - %w", err)
	}

	mpc.MachinePool.Object = response.Body()

	return nil
}
