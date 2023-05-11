package ocm

import (
	"errors"
	"fmt"

	sdk "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	rosa "github.com/openshift/rosa/pkg/aws"
	rosatags "github.com/openshift/rosa/pkg/aws/tags"
	rosahelper "github.com/openshift/rosa/pkg/helper"

	"github.com/rh-mobb/ocm-operator/pkg/aws"
)

const (
	operatorRolesPolicyType = "OperatorRole"
)

var (
	ErrCredentialRequestsEmpty = errors.New("unable to retrieve credential requests; empty response")
	ErrPolicyARNEmpty          = errors.New("unable to find policy arn")
)

type STSClient struct {
	Prefix             string
	AccountID          string
	CredentialRequest  *clustersmgmtv1.STSCredentialRequestsInquiryListRequest
	PolicyRequest      *clustersmgmtv1.AWSSTSPoliciesInquiryListRequest
	HostedControlPlane bool
	OIDCEndpointURL    string
	ManagedPolicies    bool
}

type STSCredentialRequest struct {
	ID        string
	Namespace string
	Operator  *clustersmgmtv1.STSOperator
	Role      *clustersmgmtv1.OperatorIAMRole
}

func NewSTSClient(
	connection *sdk.Connection,
	hostedControlPlane, managedPolicies bool,
	prefix, accountID, oidcEndpointURL string,
) *STSClient {
	return &STSClient{
		Prefix:          prefix,
		AccountID:       accountID,
		OIDCEndpointURL: oidcEndpointURL,
		CredentialRequest: connection.ClustersMgmt().
			V1().
			AWSInquiries().
			STSCredentialRequests().
			List().
			Parameter("is_hypershift", hostedControlPlane),
		PolicyRequest: connection.ClustersMgmt().
			V1().
			AWSInquiries().
			STSPolicies().
			List().
			Search(fmt.Sprintf("policy_type = '%s'", operatorRolesPolicyType)),
		HostedControlPlane: hostedControlPlane,
		ManagedPolicies:    managedPolicies,
	}
}

func (stsClient *STSClient) GetCredentialRequests() ([]*STSCredentialRequest, error) {
	stsCredentialResponse, err := stsClient.CredentialRequest.Send()
	if err != nil {
		return []*STSCredentialRequest{}, fmt.Errorf("error retrieving sts credential requests - %w", err)
	}

	requests := make([]*STSCredentialRequest, len(stsCredentialResponse.Items().Slice()))

	// return an error if we found no items in the response
	if len(stsCredentialResponse.Items().Slice()) == 0 {
		return requests, ErrCredentialRequestsEmpty
	}

	// append the request for each response item
	for i, req := range stsCredentialResponse.Items().Slice() {
		request := &STSCredentialRequest{
			ID:        req.Name(),
			Namespace: req.Operator().Namespace(),
			Operator:  req.Operator(),
		}

		role, err := clustersmgmtv1.NewOperatorIAMRole().
			Name(req.Name()).
			Namespace(req.Operator().Namespace()).
			RoleARN(aws.GetOperatorRoleArn(request.Operator.Name(), request.Namespace, stsClient.AccountID, stsClient.Prefix)).
			Build()
		if err != nil {
			return requests, fmt.Errorf("unable to build iam operator roles - %w", err)
		}

		request.Role = role

		requests[i] = request
	}

	return requests, nil
}

// CreateOperatorRoles creates the operator roles given a specific version and a set of
// credential requests obtained from OCM.
//
//nolint:cyclop
func (stsClient *STSClient) CreateOperatorRoles(
	awsClient *aws.Client,
	ver *clustersmgmtv1.Version,
	requests ...*STSCredentialRequest,
) error {
	// get the list of policies
	policyResponse, err := stsClient.PolicyRequest.Send()
	if err != nil {
		return fmt.Errorf("unable to retrieve sts policies - %w", err)
	}
	policies := policyResponse.Items().Slice()

	// get the version in a format compatible with sts roles/policies
	version := MajorMinorVersion(ver)

	for i := range requests {
		// retrieve the role name for this request
		roleName, err := rosa.GetResourceIdFromARN(requests[i].Role.RoleARN())
		if err != nil || roleName == "" {
			return fmt.Errorf("unable to find role name from role arn [%s] - %w", requests[i].Role.RoleARN(), err)
		}

		// retrieve the policy arn for this request
		policyID := rosa.GetOperatorPolicyKey(requests[i].ID, stsClient.HostedControlPlane)

		// set the tags
		tagsList := map[string]string{
			rosatags.OperatorNamespace: requests[i].Namespace,
			rosatags.OperatorName:      requests[i].Operator.Name(),
			rosatags.RedHatManaged:     rosahelper.True,
			rosatags.RolePrefix:        stsClient.Prefix,
			rosatags.OpenShiftVersion:  version,
		}

		if stsClient.ManagedPolicies {
			tagsList[rosatags.ManagedPolicies] = rosahelper.True
		}

		if stsClient.HostedControlPlane {
			tagsList[rosatags.HypershiftPolicies] = rosahelper.True
		}

		var policyARN string

		if stsClient.ManagedPolicies {
			policyARN = getPolicyARNByID(policyID, policies...)
			if policyARN == "" {
				return fmt.Errorf("error retrieving policy id [%s] - %w", policyID, ErrPolicyARNEmpty)
			}
		} else {
			policyARN = rosa.GetOperatorPolicyARN(
				stsClient.AccountID,
				stsClient.Prefix,
				requests[i].Namespace,
				requests[i].ID,
				"",
			)

			// ensure the policy exists
			_, err = awsClient.Connection.EnsurePolicy(policyARN, getPolicyDetails(policyID, policies...), version, tagsList, "")
			if err != nil {
				return fmt.Errorf("unable to create policy [%s] - %w", policyID, err)
			}
		}

		// ensure the role exists
		policy, err := rosa.GenerateOperatorRolePolicyDocByOidcEndpointUrl(
			stsClient.OIDCEndpointURL,
			stsClient.AccountID,
			requests[i].Operator,
			getPolicyDetails("operator_iam_role_policy", policies...),
		)
		if err != nil {
			return fmt.Errorf("error retrieving iam role policy details - %w", err)
		}

		_, err = awsClient.Connection.EnsureRole(roleName, policy, "", "", tagsList, "", stsClient.ManagedPolicies)
		if err != nil {
			return fmt.Errorf("unable to create aws iam role [%s] - %w", roleName, err)
		}

		// attach the policy to the role
		if err := awsClient.Connection.AttachRolePolicy(roleName, policyARN); err != nil {
			return fmt.Errorf("unable to attach iam policy [%s] to iam role [%s] - %w", policyARN, roleName, err)
		}
	}

	return nil
}

// DeleteOperatorRoles deletes the operator roles given a specific version and a set of
// credential requests obtained from OCM.
func (stsClient *STSClient) DeleteOperatorRoles(awsClient *aws.Client, requests ...*STSCredentialRequest) error {
	// turn our requests into a format understood by the underlying library
	requestsMap := make(map[string]*clustersmgmtv1.STSOperator)

	for i := range requests {
		requestsMap[requests[i].ID] = requests[i].Operator
	}

	// get the operator roles
	operatorRoles, err := awsClient.Connection.GetOperatorRolesFromAccountByPrefix(stsClient.Prefix, requestsMap)
	if err != nil {
		return fmt.Errorf("unable to retrieve operator roles - %w", err)
	}

	// return if there are no roles to delete
	if len(operatorRoles) == 0 {
		return nil
	}

	// delete the operator roles
	for _, role := range operatorRoles {
		if err := awsClient.Connection.DeleteOperatorRole(role, stsClient.ManagedPolicies); err != nil {
			return fmt.Errorf("unable to delete role [%s] - %w", role, err)
		}
	}

	return nil
}

func getPolicyARNByID(id string, existing ...*clustersmgmtv1.AWSSTSPolicy) string {
	for policy := range existing {
		if existing[policy].ID() == id {
			return existing[policy].ARN()
		}
	}

	return ""
}

func getPolicyDetails(id string, existing ...*clustersmgmtv1.AWSSTSPolicy) string {
	for policy := range existing {
		if existing[policy].ID() == id {
			return existing[policy].Details()
		}
	}

	return ""
}
