package aws

import (
	"errors"
	"fmt"
	"io"

	rosa "github.com/openshift/rosa/pkg/aws"
	"github.com/sirupsen/logrus"
)

var (
	ErrConvertAWSClient = errors.New("unable to convert rosa client to internal client")
)

type Client struct {
	Connection rosa.Client
}

// NewClient returns a new instance of an AWS client.  This client is loaded
// from the rosa package to maintain consistency and supportability.
func NewClient() (*Client, error) {
	// create the client from the rosa package
	aws, err := rosa.NewClient().Logger(&logrus.Logger{Out: io.Discard}).Build()
	if err != nil {
		return &Client{}, fmt.Errorf("unable to create aws client - %w", err)
	}

	return &Client{Connection: aws}, nil
}
