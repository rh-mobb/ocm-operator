/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

const (
	LDAPBindPasswordKey = "bindPassword"
	LDAPCAKey           = "ca.crt"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LDAPIdentityProviderSpec defines the desired state of LDAPIdentityProvider.
//
//nolint:lll
type LDAPIdentityProviderSpec struct {
	// url is an RFC 2255 URL which specifies the LDAP search parameters to use.
	// The syntax of the URL is:
	// ldap://host:port/basedn?attribute?scope?filter
	URL string `json:"url"`

	// bindDN is an optional DN to bind with during the search phase.
	// +optional
	BindDN string `json:"bindDN"`

	// bindPassword is an optional reference to a secret by name
	// containing a password to bind with during the search phase.
	// The key "bindPassword" is used to locate the data.
	// If specified and the secret or expected key is not found, the identity provider is not honored.
	// This should exist in the same namespace as the operator.
	// +optional
	BindPassword configv1.SecretNameReference `json:"bindPassword"`

	// insecure, if true, indicates the connection should not use TLS
	// WARNING: Should not be set to `true` with the URL scheme "ldaps://" as "ldaps://" URLs always
	//          attempt to connect using TLS, even when `insecure` is set to `true`
	// When `true`, "ldap://" URLS connect insecurely. When `false`, "ldap://" URLs are upgraded to
	// a TLS connection using StartTLS as specified in https://tools.ietf.org/html/rfc2830.
	// +kubebuilder:default=false
	Insecure bool `json:"insecure"`

	// ca is an optional reference to a config map by name containing the PEM-encoded CA bundle.
	// It is used as a trust anchor to validate the TLS certificate presented by the remote server.
	// The key "ca.crt" is used to locate the data.
	// If specified and the config map or expected key is not found, the identity provider is not honored.
	// If the specified ca data is not valid, the identity provider is not honored.
	// If empty, the default system roots are used.
	// The namespace for this config map is openshift-config.
	// +optional
	CA configv1.ConfigMapNameReference `json:"ca"`

	// attributes maps LDAP attributes to identities
	// +optional
	Attributes configv1.LDAPAttributeMapping `json:"attributes"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:message="clusterName is immutable",rule=(self == oldSelf)
	// Cluster ID in OpenShift Cluster Manager by which this should be managed for.  The cluster ID
	// can be obtained on the Clusters page for the individual cluster.  It may also be known as the
	// 'External ID' in some CLI clients.  It shows up in the format of 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
	// where the 'x' represents any alphanumeric character.
	ClusterName string `json:"clusterName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=4
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:XValidation:message="displayName is immutable",rule=(self == oldSelf)
	// Friendly display name as displayed in the OpenShift Cluster Manager
	// console.  If this is empty, the metadata.name field of the parent resource is used
	// to construct the display name.  This is limited to 15 characters as per the backend
	// API limitation.
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=claim
	// +kubebuilder:validation:Enum=claim;lookup;generate;add
	// Mapping method to use for the identity provider.
	// See https://docs.openshift.com/container-platform/latest/authentication/understanding-identity-provider.html#identity-provider-parameters_understanding-identity-provider
	// for a detailed description of what these mean.  Must be one of claim (default), lookup, generate, or add.
	MappingMethod string `json:"mappingMethod,omitempty"`
}

// LDAPIdentityProviderStatus defines the observed state of LDAPIdentityProvider.
type LDAPIdentityProviderStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.clusterID is immutable",rule=(self == oldSelf)
	// Represents the programmatic cluster ID of the cluster, as
	// determined during reconciliation.  This is used to reduce
	// the number of API calls to look up a cluster ID based on
	// the cluster name.
	ClusterID string `json:"clusterID,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.providerID is immutable",rule=(self == oldSelf)
	// Represents the programmatic identity provider ID of the IDP, as
	// determined during reconciliation.  This is used to reduce
	// the number of API calls to look up a cluster ID based on
	// the identity provider name.
	ProviderID string `json:"providerID,omitempty"`
}

// +kubebuilder:resource:categories=idps;identityproviders
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LDAPIdentityProvider is the Schema for the ldapidentityproviders API.
type LDAPIdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPIdentityProviderSpec   `json:"spec,omitempty"`
	Status LDAPIdentityProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPIdentityProviderList contains a list of LDAPIdentityProvider.
type LDAPIdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPIdentityProvider `json:"items"`
}

// GetConditions returns the status.conditions field from the object.  It is used to
// satisfy the Workload interface.
func (ldap *LDAPIdentityProvider) GetConditions() []metav1.Condition {
	return ldap.Status.Conditions
}

// SetConditions sets the status.conditions field from the object.  It is used to
// satisfy the Workload interface.
func (ldap *LDAPIdentityProvider) SetConditions(conditions []metav1.Condition) {
	ldap.Status.Conditions = conditions
}

// CopyFrom copies relevant fields from an LDAP Identity provider into an object that is able to be reconciled.
func (ldap *LDAPIdentityProvider) CopyFrom(source *clustersmgmtv1.LDAPIdentityProvider) {
	ldap.Spec.URL = source.URL()
	ldap.Spec.BindDN = source.BindDN()
	ldap.Spec.Insecure = source.Insecure()
	ldap.Spec.Attributes = LDAPAttributesToOpenShift(
		source.Attributes().ID(),
		source.Attributes().Name(),
		source.Attributes().Email(),
		source.Attributes().PreferredUsername(),
	)
}

// Builder returns the builder object from a reconciler object.  This object is used to
// pass into the OCM API for creating the object.
func (ldap *LDAPIdentityProvider) Builder(ca, bindPassword string) *clustersmgmtv1.IdentityProviderBuilder {
	builder := clustersmgmtv1.NewIdentityProvider().
		MappingMethod(clustersmgmtv1.IdentityProviderMappingMethod(ldap.Spec.MappingMethod)).
		Name(ldap.Spec.DisplayName).
		Type(clustersmgmtv1.IdentityProviderTypeLDAP)

	return builder.LDAP(
		clustersmgmtv1.NewLDAPIdentityProvider().
			URL(ldap.Spec.URL).
			BindDN(ldap.Spec.BindDN).
			Insecure(ldap.Spec.Insecure).
			BindPassword(bindPassword).
			Attributes(LDAPAttributesToOCM(
				ldap.Spec.Attributes.ID,
				ldap.Spec.Attributes.Name,
				ldap.Spec.Attributes.Email,
				ldap.Spec.Attributes.PreferredUsername,
			)).
			CA(ca),
	)
}

// LDAPAttributesToOpenShift copies fields from an OCM LDAP object into an OpenShift object.
func LDAPAttributesToOpenShift(id, name, email, username []string) configv1.LDAPAttributeMapping {
	return configv1.LDAPAttributeMapping{
		ID:                getLDAPAttributes(id, ocm.DefaultAttributeID),
		Name:              getLDAPAttributes(name, ocm.DefaultAttributeName),
		Email:             getLDAPAttributes(email, ocm.DefaultAttributeEmail),
		PreferredUsername: getLDAPAttributes(username, ocm.DefaultAttributeUsername),
	}
}

// LDAPAttributesToOCM converts fields from an OCM LDAP object into an OCM object.
func LDAPAttributesToOCM(id, name, email, username []string) *clustersmgmtv1.LDAPAttributesBuilder {
	return clustersmgmtv1.NewLDAPAttributes().
		ID(getLDAPAttributes(id, ocm.DefaultAttributeID)...).
		Name(getLDAPAttributes(name, ocm.DefaultAttributeName)...).
		Email(getLDAPAttributes(email, ocm.DefaultAttributeEmail)...).
		PreferredUsername(getLDAPAttributes(username, ocm.DefaultAttributeUsername)...)
}

// getLDAPAttributes will return the attributes with a default if attributes are not provided.
func getLDAPAttributes(attributes []string, def string) []string {
	if len(attributes) == 0 {
		return []string{def}
	}

	return attributes
}

func init() {
	SchemeBuilder.Register(&LDAPIdentityProvider{}, &LDAPIdentityProviderList{})
}
