package ldapidentityprovider

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers/requeue"
)

// errUnableToUpdateStatusProviderID produces an error indicating the LDAP IDP was unable
// to be updated.
func errUnableToUpdateStatusProviderID(request *LDAPIdentityProviderRequest, id string, err error) (ctrl.Result, error) {
	return requeue.OnError(request, fmt.Errorf(
		"unable to update ldap identity provider [%s] status [providerID=%s] - %w",
		request.GetName(),
		id,
		err,
	))
}

// errGetBindPassword produces an error indicating that the bind password was unable
// to be retrieved for setting up the request.
func errGetBindPassword(from *ocmv1alpha1.LDAPIdentityProvider) error {
	return fmt.Errorf(
		"unable to retrieve bind password from secret [%s/%s] at key [%s] - %w",
		from.Namespace,
		from.Spec.BindPassword.Name,
		ocmv1alpha1.LDAPBindPasswordKey,
		ErrMissingBindPassword,
	)
}

// errGetCert produces an error indicating that the CA certificate was unable
// to be retrieved for setting up the request.
func errGetCert(from *ocmv1alpha1.LDAPIdentityProvider) error {
	return fmt.Errorf(
		"unable to retrieve ca cert from config map [%s/%s] at key [%s] - %w",
		from.Namespace,
		from.Spec.CA.Name,
		ocmv1alpha1.LDAPCAKey,
		ErrMissingCA,
	)
}
