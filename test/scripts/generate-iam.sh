#!/usr/bin/env bash

# check required tooling
if [ -z `which aws` ]; then
    echo "missing aws cli...please install and configure"
    echo "see https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html for details"
    exit 1
fi

if [ -z `which rosa` ]; then
    echo "missing rosa cli...please install and configure"
    echo "see https://docs.openshift.com/rosa/rosa_cli/rosa-get-started-cli.html for details"
    exit 1
fi

if [ -z `which jq` ]; then
    echo "missing jq cli...please install and configure"
    echo "see https://jqlang.github.io/jq/download/ for details"
    exit 1
fi

# check required input environment variables
: ${AWS_ACCOUNT_ID?missing AWS_ACCOUNT_ID environment variable}
: ${ROSA_CLUSTER_NAME?missing ROSA_CLUSTER_NAME environment variable}

# retrieve the oidc endpoint url
# NOTE: we do this here to fail fast in case we cannot pull the oidc information
OIDC_ENDPOINT_URL=$(rosa describe cluster -c $ROSA_CLUSTER_NAME -o json | jq -r '.aws.sts.oidc_endpoint_url')
if [ -z "${OIDC_ENDPOINT_URL}" ]; then
    echo "unable to determine OIDC_ENDPOINT_URL for cluster : ${ROSA_CLUSTER_NAME}"
    exit 1
fi
OIDC_ENDPOINT_HOST=$(echo $OIDC_ENDPOINT_URL | awk -F'https://' '{print $NF}')

# set default environment
# NOTE: AWS_ARN_PREFIX needed to support GovCloud
AWS_ARN_PREFIX="${AWS_ARN_PREFIX:-aws}"
OCM_OPERATOR_NAMESPACE="${OCM_OPERATOR_NAMESPACE:-ocm-operator}"

# policy variables
# NOTE: these are variables related to the permissions that ocm directly uses
# NOTE: filters below unused until https://github.com/rh-mobb/ocm-operator/issues/19 is finished
OCM_POLICY_RESOURCE_FILTER="*"
OCM_POLICY_NAME="${OCM_POLICY_NAME:-OCMOperator}"
if [ -n "${ROSA_CLUSTER_NAME}" ]; then
    OCM_POLICY_NAME="${ROSA_CLUSTER_NAME}-${OCM_POLICY_NAME}"
    # NOTE: filters below unused until https://github.com/rh-mobb/ocm-operator/issues/19 is finished
    # OCM_POLICY_RESOURCE_FILTER="arn:${AWS_ARN_PREFIX}:iam::${AWS_ACCOUNT_ID}:policy/${ROSA_CLUSTER_NAME}-*"
fi
OCM_POLICY_ARN="arn:${AWS_ARN_PREFIX}:iam::${AWS_ACCOUNT_ID}:policy/${OCM_POLICY_NAME}"
OCM_POLICY_FILE="${OCM_POLICY_FILE:-https://raw.githubusercontent.com/rh-mobb/ocm-operator/main/test/aws/iam_policy_variables.json}"

# role variables
OCM_ROLE_RESOURCE_FILTER="*"
OCM_ROLE_NAME="${OCM_ROLE_NAME:-OCMOperator}"
if [ -n "${ROSA_CLUSTER_NAME}" ]; then
    OCM_ROLE_NAME="${ROSA_CLUSTER_NAME}-${OCM_ROLE_NAME}"
    # NOTE: filters below unused until https://github.com/rh-mobb/ocm-operator/issues/19 is finished
    # OCM_ROLE_RESOURCE_FILTER="arn:${AWS_ARN_PREFIX}:iam::${AWS_ACCOUNT_ID}:role/${ROSA_CLUSTER_NAME}-*"
fi
OCM_ROLE_ARN="arn:${AWS_ARN_PREFIX}:iam::${AWS_ACCOUNT_ID}:role/${OCM_ROLE_NAME}"
OCM_ROLE_TRUST_POLICY_FILE="${OCM_ROLE_TRUST_POLICY_FILE:-https://raw.githubusercontent.com/rh-mobb/ocm-operator/main/test/aws/trust_policy_variables.json}"

# ensure the ocm operator policy exists in the account
echo "retrieving ocm operator policy: '${OCM_POLICY_ARN}'"
aws iam get-policy --policy-arn=${OCM_POLICY_ARN}

if [ $? != 0 ]; then
    echo "missing ocm operator policy...creating from '${OCM_POLICY_FILE}'..."
    policy=$(curl -s "${OCM_POLICY_FILE}")
    new_policy=${new_policy//\$OCM_ROLE_RESOURCE_FILTER/$OCM_ROLE_RESOURCE_FILTER}
    new_policy=${new_policy//\$OCM_POLICY_RESOURCE_FILTER/$OCM_POLICY_RESOURCE_FILTER}
    aws iam create-policy \
        --policy-name "${OCM_POLICY_NAME}" \
        --policy-document "$new_policy"
fi

# ensure the ocm operator roles exists in the account
echo "retrieving ocm operator role: '${OCM_ROLE_ARN}'"
aws iam get-role --role-name=${OCM_ROLE_NAME}

if [ $? != 0 ]; then
    echo "missing ocm operator role...creating..."
    trust_policy=$(curl -s "${OCM_ROLE_TRUST_POLICY_FILE}")
    new_trust_policy=${trust_policy//\$AWS_ACCOUNT_ID/$AWS_ACCOUNT_ID}
    new_trust_policy=${new_trust_policy//\$OCM_OPERATOR_NAMESPACE/$OCM_OPERATOR_NAMESPACE}
    new_trust_policy=${new_trust_policy//\$OIDC_ENDPOINT_HOST/$OIDC_ENDPOINT_HOST}
    aws iam create-role \
        --role-name "${OCM_ROLE_NAME}" \
        --assume-role-policy-document "$new_trust_policy"

    echo "attaching policy '${OCM_POLICY_ARN}' to role: '${OCM_ROLE_ARN}'"
    aws iam attach-role-policy \
        --role-name "${OCM_ROLE_NAME}" \
        --policy-arn "${OCM_POLICY_ARN}"
fi

# print the script output for generating the credentials file for the ocm-operator
echo
echo
echo "create your sts credentials for you operator where you intend to run it with the following:"
echo
echo "cat <<EOF > /tmp/credentials
[default]
role_arn = $OCM_ROLE_ARN
web_identity_token_file = /var/run/secrets/openshift/serviceaccount/token
EOF

oc create secret generic aws-credentials \\
    --namespace=$OCM_OPERATOR_NAMESPACE \\
    --from-file=credentials=/tmp/credentials
"
