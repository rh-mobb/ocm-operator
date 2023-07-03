package ldapidentityprovider

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rh-mobb/ocm-operator/controllers"
)

// errUnableToUpdateStatus produces an error indicating the GitLab IDP was unable
// to be updated.
func errUnableToUpdateStatusProviderID(request *LDAPIdentityProviderRequest, id string, err error) (ctrl.Result, error) {
	return controllers.RequeueOnError(request, fmt.Errorf(
		"unable to update ldap identity provider [%s] status [providerID=%s] - %w",
		request.GetName(),
		id,
		err,
	))
}
