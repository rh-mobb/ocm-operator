# GitLab

The `GitLabIdentityProvider` resource configures a cluster to be 
integrated with an existing GitLab provider.  It requires the following to 
be setup ahead of time:

1. An [application](https://docs.gitlab.com/ee/integration/oauth_provider.html) configured in your GitLab instance.
2. The Client ID from that application configured in the `spec.clientID` field of the resource.
3. The Client Secret from that application, stored in a secret at key `clientSecret`.  The name 
of the secret is configurable and is configured in the `spec.clientSecret.name` field of the resource.  You can 
create this secret with the following command:

```bash
oc create secret generic gitlab \
    --namespace=ocm-operator \
    --from-literal=clientSecret=$MY_CLIENT_SECRET
```

4. A cluster in OCM, capable of configuring Access Control for (e.g. ROSA).

Once the prereqs are met, here is an example configuring the `skynet` cluster to use a GitLab 
identity provider.  Other samples can be found [here](https://github.com/rh-mobb/ocm-operator/tree/main/config/samples/identityprovider).

```yaml
apiVersion: ocm.mobb.redhat.com/v1alpha1
kind: GitLabIdentityProvider
metadata:
  name: gitlab
spec:
  clusterName: skynet
  displayName: gitlab-sample
  mappingMethod: claim
  url: https://gitlab.com
  clientID: test
  clientSecret: 
    name: gitlab
```

# LDAP

The `LDAPIdentityProvider` resource configures a cluster to be integrated with an existing LDAP provider. 
It requires the following to be setup ahead of time:

1. A user capable of querying LDAP.
2. The bind password from the above user, stored in a secret at key `bindPassword`.  The name 
of the secret is configurable and is configured in the `spec.bindPassword.name` field of the resource.  You can 
create this secret with the following command:

```bash
oc create secret generic ldap \
    --namespace=ocm-operator \
    --from-literal=bindPassword=$MY_BIND_PASSWORD
```

3. A cluster in OCM, capable of configuring Access Control for (e.g. ROSA).

Once the prereqs are met, here is an example configuring the `skynet` cluster to use an LDAP 
identity provider.  Other samples can be found [here](https://github.com/rh-mobb/ocm-operator/tree/main/config/samples/identityprovider).

```yaml
apiVersion: ocm.mobb.redhat.com/v1alpha1
kind: LDAPIdentityProvider
metadata:
  name: ldap
spec:
  clusterName: skynet
  displayName: ldap-sample
  mappingMethod: claim
  url: ldap://test.example.com:389
  bindDN: CN=test,OU=Users,DC=example,DC=com
  bindPassword:
    name: ldap
```
