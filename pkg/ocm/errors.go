package ocm

import (
	"fmt"

	"github.com/rh-mobb/ocm-operator/controllers/request"
)

// GetError returns an error indicating an object was unable to be retrieved from OCM.
func GetError(request request.Request, err error) error {
	return fmt.Errorf(
		"unable to get [%s] with name [%s] from ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// CreateError returns an error indicating an object was unable to be created in OCM.
func CreateError(request request.Request, err error) error {
	return fmt.Errorf(
		"unable to create [%s] with name [%s] in ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// UpdateError returns an error indicating an object was unable to be updated in OCM.
func UpdateError(request request.Request, err error) error {
	return fmt.Errorf(
		"unable to update [%s] with name [%s] in ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}

// DeleteError returns an error indicating an object was unable to be deleted from OCM.
func DeleteError(request request.Request, err error) error {
	return fmt.Errorf(
		"unable to delete [%s] with name [%s] from ocm - %w",
		request.GetObject().GetObjectKind().GroupVersionKind().Kind,
		request.GetName(),
		err,
	)
}
