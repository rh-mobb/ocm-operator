# Quick Start

## Prerequisites

- [oc](https://docs.openshift.com/container-platform/4.8/cli_reference/openshift_cli/getting-started-cli.html)
- [aws](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- [ROSA Cluster (4.12+)](https://mobb.ninja/docs/rosa/sts/)

> **Warning:**
> This is only tested on ROSA, but may work on other Kubernetes and OpenShift variants.  It requires
> an AWS STS-enabled cluster.  Additionally, OpenShift versions 4.12+ are recommended, as they 
> enable [CRD CEL validation](https://kubernetes.io/blog/2022/09/23/crd-validation-rules-beta/).  Versions
> prior to 4.12 provide a lower version of Kubernetes that does not enable this feature.  They may 
> work, but provide no input validation when submitting custom resources to the cluster.

Before installing this operator, there are a couple secrets that must be created.

1. OCM Token: used to authenticate against the OCM API.  The controller will not start without
this token as it used for all calls into OCM.
2. AWS Credentials: uses an assume role to a role created in the instructions below in order
to access AWS resources.  This is needed for cluster creation.  See 
[Create AWS IAM Policies and Roles](#create-aws-iam-policies-and-roles) for more details.

> **Note:**
> Certain custom resources have their own specific prerequisites.  Please use `oc explain` 
> or read the docs [here](https://github.com/rh-mobb/ocm-operator/tree/main/docs) for more details.

### Create OCM Token Secret

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

### Create AWS IAM Policies and Roles

The operator will need to elevate privileges in order to perform things like 
creating the operator-roles for the clusters.  Because of this, the operator 
must have a specific role created to allow it these permissions.  In each instance, 
it is a best practice to create a new set of policies and roles for each instance 
of the OCM Operator.  Policies and roles are prefixed with the `ROSA_CLUSTER_NAME` 
environment variable that is specified below.

**NOTE:** please understand what you are doing if you deviate from the known good 
policies.  If errors or more stringent security lockdowns are found, please submit a PR 
so that we can get this fixed.

1. Set required variables, substituting the correct values for your environment:

* `AWS_ACCOUNT_ID`: the AWS account ID in which the ROSA cluster where you are installing 
the OCM operator is running.
* `ROSA_CLUSTER_NAME`: the ROSA cluster name by which you intend to install the OCM
operator upon.

```bash
export AWS_ACCOUNT_ID=111111111111
export ROSA_CLUSTER_NAME=dscott
```

2. Run the script to create the required policies and roles.  This creates a permission boundary, 
a policy for the operator, and a role which allows the operator to assume a role against the OIDC 
identity of the ROSA cluster.  If the policies and roles already exist (prefixed by your cluster 
name), then the creation of them is skipped:

```bash
curl -s https://raw.githubusercontent.com/rh-mobb/ocm-operator/main/test/scripts/generate-iam.sh | bash
```

You will see output at the end of the script which gives you instructions to generate
a credentials file.  This credential is needed by the operator to use the above created
role.

```bash
create your sts credentials for you operator where you intend to run it with the following:

cat <<EOF > /tmp/credentials
[default]
role_arn = arn:aws:iam::111111111111:role/dscott-OCMOperator
web_identity_token_file = /var/run/secrets/openshift/serviceaccount/token
EOF

oc create secret generic aws-credentials \
    --namespace=ocm-operator \
    --from-file=credentials=/tmp/credentials
```

3. Finally, following the instruction from the script output above, create the secret containing 
the assume role credentials:

> **NOTE** Be sure to copy the output from your script run and do not copy from this document 
as the output will be different.

```bash
cat <<EOF > /tmp/credentials
[default]
role_arn = arn:aws:iam::111111111111:role/dscott-OCMOperator
web_identity_token_file = /var/run/secrets/openshift/serviceaccount/token
EOF

oc create secret generic aws-credentials \
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
  startingCSV: ocm-operator.v0.1.0
EOF
```

## Provision Resources

Once the operator is available and running, you can provision any of the 
resources that it manages.  Also note that documentation is always available 
once the operator is installed as well by using the `oc explain` command.  For 
example, `oc explain rosacluster.spec.clusterName` will give you detailed documentation 
about what the field does.

See the following documentation for details:

* [ROSA Clusters](https://github.com/rh-mobb/ocm-operator/blob/main/docs/clusters.md)
* [Machine Pools](https://github.com/rh-mobb/ocm-operator/blob/main/docs/machinepools.md)
* [Identity Providers](https://github.com/rh-mobb/ocm-operator/blob/main/docs/identityproviders.md)
