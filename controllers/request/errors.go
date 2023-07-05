package request

import (
	"errors"
	"fmt"
)

const (
	errRetrieveClusterMessage = "unable to retrieve cluster from ocm"
)

var (
	ErrMissingClusterID = errors.New("unable to find cluster id")
)

// Error returns an error for the request.
func Error(request Request, err error) error {
	// return a nil error if we received a nil error
	if err == nil {
		return nil
	}

	return fmt.Errorf(
		"request=%s/%s - %w",
		request.GetObject().GetNamespace(),
		request.GetObject().GetName(),
		err,
	)
}

// TypeConvertError returns an error indicating a generic controllers.Request interface
// could not be converted to its underlying type.
func TypeConvertError(t interface{}) error {
	return fmt.Errorf("unable to convert request.Request interface to underlying request type [%T]", t)
}
