package ocm

import (
	"errors"
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ErrOIDCConfigResponse = errors.New("invalid cluster response")
)

type oidcConfigClient struct {
	connection *clustersmgmtv1.OidcConfigsClient
}

func NewOIDCConfigClient(connection *sdk.Connection) *oidcConfigClient {
	return &oidcConfigClient{
		connection: connection.ClustersMgmt().V1().OidcConfigs(),
	}
}

func (cfgClient *oidcConfigClient) For(id string) *clustersmgmtv1.OidcConfigClient {
	return cfgClient.connection.OidcConfig(id)
}

func (cfgClient *oidcConfigClient) Get(id string) (oidcConfig *clustersmgmtv1.OidcConfig, err error) {
	// retrieve the oidc config from openshift cluster manager
	response, err := cfgClient.For(id).Get().Send()
	if err != nil {
		return oidcConfig, fmt.Errorf("error in get request - %w", err)
	}

	return response.Body(), nil
}

func (cfgClient *oidcConfigClient) Create() (oidcConfig *clustersmgmtv1.OidcConfig, err error) {
	// build the object to create
	object, err := clustersmgmtv1.NewOidcConfig().Managed(true).Build()
	if err != nil {
		return oidcConfig, fmt.Errorf("unable to build oidc config - %w", err)
	}

	// create the oidc config provider
	response, err := cfgClient.connection.Add().Body(object).Send()
	if err != nil {
		return oidcConfig, fmt.Errorf("error in create request - %w", err)
	}

	return response.Body(), nil
}

func (cfgClient *oidcConfigClient) Delete(id string) error {
	// delete the identity provider in ocm
	response, err := cfgClient.For(id).Delete().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error in delete request - %w", err)
	}

	return nil
}
