package ocm

import (
	"errors"
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ErrConvertGitLabIdentityProvider = errors.New("error converting to gitlab identity provider object")
)

// GitLabIdentityProviderClient represents the client used to interact with a Machine Pool API object.  Machine
// pools are associated with clusters that are not using hosted control plane.
type GitLabIdentityProviderClient struct {
	name       string
	connection *clustersmgmtv1.IdentityProvidersClient
}

func NewGitLabIdentityProviderClient(connection *sdk.Connection, name, clusterID string) *GitLabIdentityProviderClient {
	return &GitLabIdentityProviderClient{
		name:       name,
		connection: connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).IdentityProviders(),
	}
}

func (glc *GitLabIdentityProviderClient) For(gitLabName string) *clustersmgmtv1.IdentityProviderClient {
	return glc.connection.IdentityProvider(gitLabName)
}

func (glc *GitLabIdentityProviderClient) Get() (gitLab *clustersmgmtv1.GitlabIdentityProvider, err error) {
	// retrieve the gitlab identity provider from ocm
	response, err := glc.For(glc.name).Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return gitLab, nil
		}

		return gitLab, fmt.Errorf("error in get request - %w", err)
	}

	return response.Body().Gitlab(), nil
}

func (glc *GitLabIdentityProviderClient) Create(builder *clustersmgmtv1.GitlabIdentityProviderBuilder) (gitLab *clustersmgmtv1.GitlabIdentityProvider, err error) {
	body := clustersmgmtv1.NewIdentityProvider().Gitlab(builder)

	// build the object to create
	object, err := body.Build()
	if err != nil {
		return gitLab, fmt.Errorf("unable to build object for gitlab identity provider creation - %w", err)
	}

	// create the gitlab identity provider in ocm
	response, err := glc.connection.Add().Body(object).Send()
	if err != nil {
		return gitLab, fmt.Errorf("error in create request - %w", err)
	}

	return response.Body().Gitlab(), nil
}

func (glc *GitLabIdentityProviderClient) Update(builder *clustersmgmtv1.GitlabIdentityProviderBuilder) (gitLab *clustersmgmtv1.GitlabIdentityProvider, err error) {
	body := clustersmgmtv1.NewIdentityProvider().Gitlab(builder)

	// build the object to update
	object, err := body.Build()
	if err != nil {
		return gitLab, fmt.Errorf("unable to build object for gitlab identity provider update - %w", err)
	}

	// update the gitlab identity provider in ocm
	response, err := glc.For(object.ID()).Update().Body(object).Send()
	if err != nil {
		return gitLab, fmt.Errorf("error in update request - %w", err)
	}

	return response.Body().Gitlab(), nil
}

func (glc *GitLabIdentityProviderClient) Delete(id string) error {
	// delete the gitlab identity provider in ocm
	response, err := glc.For(id).Delete().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error in delete request - %w", err)
	}

	return nil
}
