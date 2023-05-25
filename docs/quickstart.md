# Quick Start

Before installing this operator, there are a couple secrets that must be created.

## Create OCM Token Secret

1. Create a namespace where you wish to install the operator:

```bash
oc new-project ocm-operator
```

2. Create a secret containing the OCM_TOKEN.  This token can be obtained form 
https://console.redhat.com/openshift/token and is used by the operator to authenticate 
against the OCM API.  This token must exist in the same namespace that the operator 
is running and be named `ocm-token`.  It also expects the key to be called `OCM_TOKEN` 
as the operator is expecting this value as an environment variable.

```bash
oc create secret generic ocm-token \
    --namespace=ocm-operator \
    --from-literal=OCM_TOKEN=${MY_OCM_TOKEN}
```

## Create AWS IAM Policies and Roles

The operator will need to elevate privileges in order to perform things like 
creating the operator-roles for the clusters.  Because of this, the operator 
must have a specific role created to allow it these permissions.

**NOTE:** please understand what you are doing if you deviate from the known good 
policies.  If errors or more stringent security lockdowns are found, please submit a PR 
so that we can get this fixed.

1. Set the account ID environment variable to define your AWS account ID:

```bash
export ACCOUNT_ID=111111111111
```

2. Create the permissions boundary.  Because the operator needs to create policies and 
roles, the boundary ensures that the operator is not allowed to create additional
permissions outside of the defined boundary.  The sample permission set is located 
at `test/aws/boundary.json` in this repository:

```bash
boundary=$(curl -s "https://raw.githubusercontent.com/rh-mobb/ocm-operator/main/test/aws/boundary.json")
aws iam create-policy \
  --policy-name "OCMOperatorBoundary" \
  --policy-document "$boundary"
```

3. Create the policy.  This policy sets what the operator is allowed to do.  For any 
`iam` permission, the boundary created above is used.  The sample permission set is 
located at `test/aws/iam_policy.json` in this repository:

```bash
policy=$(curl -s "https://raw.githubusercontent.com/rh-mobb/ocm-operator/main/test/aws/iam_policy.json")
aws iam create-policy \
  --policy-name "OCMOperator" \
  --policy-document "$policy"
```

4. Create the role using a trust policy and attach the previously created role.  The trust 
policy is located at `test/aws/trust_policy.json` in this repository.  Please note that 
the trust policy requires an OIDC configuration.  The OIDC configuration refers to 
**where the operator is running, NOT what it is provisioning**:

```bash
trust_policy=$(curl -s "https://raw.githubusercontent.com/rh-mobb/ocm-operator/main/test/aws/trust_policy.json")
aws iam create-role \
    --role-name OCMOperator \
    --assume-role-policy-document "$trust_policy"
aws iam attach-role-policy \
    --role-name OCMOperator \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/OCMOperator
```

5. Finally, create the secret containing the assume role credentials.  The previous steps allow 
the operator to assume the role you created in the previous step with the permissions created 
via the previous policies.

```bash
cat <<EOF > /tmp/credentials
[default]
role_arn = arn:aws:iam::$ACCOUNT_ID:role/ECRLogin
web_identity_token_file = /var/run/secrets/openshift/serviceaccount/token
EOF

oc create secret generic ocm-token aws-credentials \
    --namespace=ocm-operator \ 
    --from-file=credentials=/tmp/credentials
```

## Install the Operator

```bash
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ocm-operator
  namespace: ocm-operator
spec:
  channel: v0.1
  installPlanApproval: Automatic
  name: ocm-operator
  source: community-operators
  sourceNamespace: ocm-operator
  startingCSV: ocm-operator.v0.1.0-alpha.3
EOF
```
