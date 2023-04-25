# ocm-operator

Manage OpenShift Cluster Manager objects in a Kubernetes-native fashion using 
an operator.


## Description

This operator allows you to manage OCM objects from a Kubernetes cluster using 
native Kubernetes CRDs.  This allows you to plugin to more modern workflows such 
as GitOps.

* [Machine Pools](https://docs.openshift.com/rosa/rosa_cluster_admin/rosa_nodes/rosa-nodes-machinepools-about.html#machine-pools): 
current limitation is that the cluster with the operator may only manage 
machine pools for itself.  See https://github.com/rh-mobb/ocm-operator/issues/1.


## Getting Started

You will need a cluster that supports managing Machine Pools from OCM (e.g. ROSA).  Please 
see https://mobb.ninja/docs/quickstart-rosa/ for a quick start guide.


### Running Outside of the Cluster (Development/Testing)

Prerequisite tooling:

* Make
* [Go => 1.20](https://go.dev/doc/install)
* [oc](https://docs.openshift.com/container-platform/4.12/cli_reference/openshift_cli/getting-started-cli.html)

1. [Retrieve your access token](https://mobb.ninja/docs/quickstart-rosa/#get-a-red-hat-offline-access-token) and 
place the token at `/tmp/ocm.json`

2. To install the custom resources for this operator, make sure you have [logged into 
your test cluster](https://docs.openshift.com/rosa/rosa_install_access_delete_clusters/rosa-sts-accessing-cluster.html) and run the following:

```bash
make install
```

3. To run the controller locally against a test cluster:

```bash
make run
```

4. You can then test the operator creation workflow by creating a sample manifest (
this may take a few minutes until you see the `completed *** reconciliation message`):

* Machine Pool: `oc apply -f config/samples/machinepool/sample_simple.yaml`

**NOTE:** other samples available for different use cases at `config/samples/<object>/sample_*.yaml`

5. You can then test the operator deletion workflow by deleting a sample manifest (
this may take a few minutes until the finalizer is deleted and the object is cleaned
up):

* Machine Pool: `oc delete -f config/samples/machinepool/sample_simple.yaml`

**NOTE:** other samples available for different use cases at `config/samples/<object>/sample_*.yaml`

6. To clean up CRDs from the cluster:

```bash
make uninstall
```


### Running on the Cluster

TODO: will be installed via OperatorHub


### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.


## License

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

