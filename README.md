# ocm-operator

Manage OpenShift Cluster Manager Machine Pools in a Kubernetes-native fashion using 
an operator.


## Description

This operator allows you to manage OCM Machine Pools from a Kubernetes cluster using 
native Kubernetes CRDs.  This allows you to plugin to more modern workflows such 
as GitOps.

**NOTE:** current limitation is that the cluster with the operator may only manage 
machine pools for itself.  See https://github.com/rh-mobb/ocm-operator/issues/1.


## Getting Started

You will need a cluster that supports managing Machine Pools from OCM (e.g. ROSA).  Please 
see https://mobb.ninja/docs/quickstart-rosa/ for a quick start guide.


### Running Outside of the Cluster (Development/Testing)

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

4. You can then test the operator by creating the sample manifests:

**NOTE:** other samples available for different use cases at `config/samples/sample_*.yaml`

```bash
oc apply -f config/samples/sample_simple.yaml
```

5. To clean up:

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

