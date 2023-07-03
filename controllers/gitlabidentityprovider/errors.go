package gitlabidentityprovider

import (
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers"
)

var (
	// TODO: this is unused for now because GitLab applications cannot be managed by non-admin users via the API.  The
	//       intent is that this will eventually be used if that ever becomes a possibility.
	ErrGitLabApplicationDrift = errors.New("gitlab application is immutable but differs from the desired state configuration")
)

// errUnableToUpdateStatus produces an error indicating the GitLab IDP was unable
// to be updated.
func errUnableToUpdateStatusProviderID(request *GitLabIdentityProviderRequest, id string, err error) (ctrl.Result, error) {
	return controllers.RequeueOnError(request, fmt.Errorf(
		"unable to update gitlab identity provider [%s] status [providerID=%s] - %w",
		request.GetName(),
		id,
		err,
	))
}
