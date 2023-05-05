package ocm

import (
	"errors"
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	callbackURLPrefix = "oauth"
)

var (
	ErrIdentityProviderMissing = errors.New("unable to find identity provider")
)

// IdentityProviderClient represents the client used to interact with a  Identity Provider API object.  Machine
// pools are associated with clusters that are not using hosted control plane.
type IdentityProviderClient struct {
	name       string
	connection *clustersmgmtv1.IdentityProvidersClient
}

func NewIdentityProviderClient(connection *sdk.Connection, name, clusterID string) *IdentityProviderClient {
	return &IdentityProviderClient{
		name:       name,
		connection: connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).IdentityProviders(),
	}
}

func (idpClient *IdentityProviderClient) For(id string) *clustersmgmtv1.IdentityProviderClient {
	return idpClient.connection.IdentityProvider(id)
}

func (idpClient *IdentityProviderClient) Get() (idp *clustersmgmtv1.IdentityProvider, err error) {
	// retrieve the identity provider from ocm
	response, err := idpClient.connection.List().Send()
	if err != nil {
		return idp, fmt.Errorf("error in get request - %w", err)
	}

	for _, idp := range response.Items().Slice() {
		if idp.Name() == idpClient.name {
			return idp, nil
		}
	}

	return idp, fmt.Errorf("missing identity provider with name [%s] - %w", idpClient.name, ErrIdentityProviderMissing)
}

func (idpClient *IdentityProviderClient) Create(builder *clustersmgmtv1.IdentityProviderBuilder) (gitLab *clustersmgmtv1.IdentityProvider, err error) {
	// build the object to create
	object, err := builder.Build()
	if err != nil {
		return gitLab, fmt.Errorf("unable to build object for identity provider creation - %w", err)
	}

	// create the identity provider in ocm
	response, err := idpClient.connection.Add().Body(object).Send()
	if err != nil {
		return gitLab, fmt.Errorf("error in create request - %w", err)
	}

	return response.Body(), nil
}

func (idpClient *IdentityProviderClient) Update(builder *clustersmgmtv1.IdentityProviderBuilder) (gitLab *clustersmgmtv1.IdentityProvider, err error) {
	// build the object to update
	object, err := builder.Build()
	if err != nil {
		return gitLab, fmt.Errorf("unable to build object for identity provider update - %w", err)
	}

	// update the identity provider in ocm
	response, err := idpClient.For(object.ID()).Update().Body(object).Send()
	if err != nil {
		return gitLab, fmt.Errorf("error in update request - %w", err)
	}

	return response.Body(), nil
}

func (idpClient *IdentityProviderClient) Delete(id string) error {
	// delete the identity provider in ocm
	response, err := idpClient.For(id).Delete().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error in delete request - %w", err)
	}

	return nil
}

func GetCallbackURL(cluster *clustersmgmtv1.Cluster, name string) string {
	return fmt.Sprintf("%s.%s.%s/oauth2callback/%s",
		callbackURLPrefix,
		cluster.Name(),
		cluster.DNS().BaseDomain(),
		name,
	)
}
