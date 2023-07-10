package ocm

import (
	"fmt"

	"github.com/rh-mobb/ocm-operator/controllers/request"
)

// GetError returns an error indicating an object was unable to be retrieved from OCM.
func GetError(req request.Request, err error) error {
	return fmt.Errorf(
		"unable to get [%s] with name [%s] from ocm - %w",
		req.GetObject().GetObjectKind().GroupVersionKind().Kind,
		req.GetName(),
		err,
	)
}

// CreateError returns an error indicating an object was unable to be created in OCM.
func CreateError(req request.Request, err error) error {
	return fmt.Errorf(
		"unable to create [%s] with name [%s] in ocm - %w",
		req.GetObject().GetObjectKind().GroupVersionKind().Kind,
		req.GetName(),
		err,
	)
}

// UpdateError returns an error indicating an object was unable to be updated in OCM.
func UpdateError(req request.Request, err error) error {
	return fmt.Errorf(
		"unable to update [%s] with name [%s] in ocm - %w",
		req.GetObject().GetObjectKind().GroupVersionKind().Kind,
		req.GetName(),
		err,
	)
}

// DeleteError returns an error indicating an object was unable to be deleted from OCM.
func DeleteError(req request.Request, err error) error {
	return fmt.Errorf(
		"unable to delete [%s] with name [%s] from ocm - %w",
		req.GetObject().GetObjectKind().GroupVersionKind().Kind,
		req.GetName(),
		err,
	)
}
