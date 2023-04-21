package ocm

import (
	"fmt"
	"net/http"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	LabelPrefixManaged = "ocm.mobb.redhat.com/managed"
	LabelPrefixName    = "ocm.mobb.redhat.com/name"

	createRetries  = 5
	createInterval = 5 * time.Second
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
	// retrive the machine pool from ocm
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

	// // create the machine pool in ocm
	// // NOTE: we run this on a loop because the ocm api occasionally returns a nil response
	// ticker := time.NewTicker(createInterval)
	// defer ticker.Stop()

	// var retries int

	// for {
	// 	select {
	// 	case <-ticker.C:
	// 		if retries == createRetries {
	// 			return machinePool, fmt.Errorf("exceeded create retries - %w", err)
	// 		}

	// 		response, err := mpc.connection.Add().Body(object).Send()
	// 		// retry if we have no response
	// 		if response == nil {
	// 			retries++

	// 			continue
	// 		}

	// 		if err != nil {
	// 			return machinePool, fmt.Errorf("error in create request - %w", err)
	// 		}

	// 		// return the response
	// 		return response.Body(), nil
	// 	default:
	// 		break
	// 	}
	// }
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
