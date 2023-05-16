package aws

import "fmt"

// GetAvailabilityZonesBySubnet returns the availability zone ids for a list of
// subnet IDs.
func (awsClient *Client) GetAvailabilityZonesBySubnet(subnetIDs []string) ([]string, error) {
	availabilityZones := make([]string, len(subnetIDs))

	for i := range subnetIDs {
		availabilityZone, err := awsClient.Connection.GetSubnetAvailabilityZone(subnetIDs[i])
		if err != nil {
			return availabilityZones, fmt.Errorf(
				"unable to retrieve subnet id [%s] - %w",
				subnetIDs[i],
				err,
			)
		}

		availabilityZones[i] = availabilityZone
	}

	return availabilityZones, nil
}
