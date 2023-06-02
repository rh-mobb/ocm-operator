# ocm-operator

Manage OpenShift Cluster Manager objects in a Kubernetes-native fashion using 
an operator.


## Description

This operator allows you to manage OCM objects from a Kubernetes cluster using 
native Kubernetes CRDs.  This allows you to plugin to more modern workflows such 
as GitOps.

* [Machine Pools](https://docs.openshift.com/rosa/rosa_cluster_admin/rosa_nodes/rosa-nodes-machinepools-about.html#machine-pools)
* [ROSA Clusters](https://docs.openshift.com/rosa/welcome/index.html)
* [LDAP Identity Providers](https://docs.openshift.com/rosa/rosa_install_access_delete_clusters/rosa-sts-config-identity-providers.html#config-ldap-idp_rosa-sts-config-identity-providers)
* [GitLab Identity Providers](https://mobb.ninja/docs/idp/gitlab/)


### Quickstart

The quickstart documentation can be found [here](docs/quickstart.md)


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

