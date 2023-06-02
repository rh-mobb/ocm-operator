# ROSA Cluster

The `ROSACluster` resource provisions a ROSA cluster.

1. Create the AWS IAM policies and roles in the quickstart guide [here](https://github.com/rh-mobb/ocm-operator/blob/main/docs/quickstart.md).  
These policies and roles are needed to authenticate with the AWS API to perform various functions such as 
creating operator roles and OIDC providers.


Once the prereqs are met, here is an example provisioning a ROSA Classic cluster:

```yaml
apiVersion: ocm.mobb.redhat.com/v1alpha1
kind: ROSACluster
metadata:
  name: rosa-classic
spec:
  accountID: "111111111111"
  displayName: rosa-classic
  tags:
    owner: dscott
  iam:
    userRole: "arn:aws:iam::111111111111:role/ManagedOpenShift-User-dscott_mobb-Role"
  defaultMachinePool:
    minimumNodesPerZone: 2
    instanceType: m5.xlarge
    labels:
      this: that
```

Here is an example provisioning a ROSA Hosted Control Plane Cluster (this requires your OCM organization 
to be setup for HCP at this time):

```yaml
apiVersion: ocm.mobb.redhat.com/v1alpha1
kind: ROSACluster
metadata:
  name: rosa-hosted
spec:
  hostedControlPlane: true
  multiAZ: true
  displayName: rosa-hosted
  openshiftVersion: "4.12.12"
  accountID: "111111111111"
  region: us-west-2
  tags:
    owner: dscott
  iam:
    userRole: "arn:aws:iam::111111111111:role/ManagedOpenShift-User-dscott_mobb-Role"
    enableManagedPolicies: false
  defaultMachinePool:
    minimumNodesPerZone: 2
    maximumNodesPerZone: 3
    instanceType: m6id.xlarge
  # NOTE: additional configuration required for KMS
  # encryption:
  #   etcd:
  #     kmsKey: "arn:aws:kms:us-east-1:660250927410:key/b121f0ea-7ad4-4153-b270-1592872f2e7d"
  network:
    machineCIDR: "10.10.0.0/16"
    subnets:
      - "subnet-04a4aead114ba92b0"
      - "subnet-04117f78f5866c4a2"
```
