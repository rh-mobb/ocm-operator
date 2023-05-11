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
	"fmt"
	"strings"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	rosaProduceID = "rosa"

	rosaAccountRolePrefix           = "ManagedOpenShift"
	rosaSupportRolePrefix           = "Support"
	rosaInstallerRolePrefix         = "Installer"
	rosaControlPlaneRolePrefix      = "ControlPlane"
	rosaWorkerRolePrefix            = "Worker"
	rosaPropertyUserRole            = "rosa_creator_arn"
	rosaPropertyProvisioner         = "rosa_provisioner"
	rosaPropertyProvisionerOperator = "ocm-operator"

	rosaDefaultMachineCIDR = "10.0.0.0/16"
	rosaDefaultPodCIDR     = "10.128.0.0/14"
	rosaDefaultServiceCIDR = "172.30.0.0/16"
	rosaDefaultHostPrefix  = 23

	rosaSingleAZCount                = 1
	rosaMultiAZCount                 = 3
	rosaHostedControlPlaneCount      = 0
	rosaHostedControlPlaneInfraCount = 0
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:XValidation:message="singleAZ clusters require a minimum of 2 nodes",rule=(self.multiAZ || self.defaultMachinePool.minimumNodesPerZone >= 2)
// +kubebuilder:validation:XValidation:message="additionalTrustBundle only supported when network.subnets is specified",rule=(has(self.network.subnets) && has(self.additionalTrustBundle) && self.network.subnets.size() > 0 || !has(self.additionalTrustBundle))
// +kubebuilder:validation:XValidation:message="hostedControlPlane cannot have node labels",rule=(!self.hostedControlPlane || self.hostedControlPlane && !has(self.defaultMachinePool.labels) || self.hostedControlPlane && has(self.defaultMachinePool.labels) && self.defaultMachinePool.labels.size() == 0)
// ROSAClusterSpec defines the desired state of ROSACluster.
//
//nolint:lll
type ROSAClusterSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=4
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:XValidation:message="displayName is immutable",rule=(self == oldSelf)
	// Friendly display name as displayed in the OpenShift Cluster Manager
	// console.  If this is empty, the metadata.name field of the parent resource is used
	// to construct the display name.  This is limited to 15 characters as per a backend
	// API limitation.
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:message="accountID is immutable",rule=(self == oldSelf)
	// AWS Account ID where the ROSA Cluster will be provisioned.
	AccountID string `json:"accountID,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:message="hostedControlPlane is immutable",rule=(self == oldSelf)
	// Provision a hosted control plane outside of the AWS account that this is being managed for (default: false).
	// Must have hosted control plane enabled for your OCM organization.  Be aware of
	// valid region differences in the '.spec.region' field.
	HostedControlPlane bool `json:"hostedControlPlane,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="openshiftVersion is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="openshiftVersion must either be blank or valid x.y.z format",rule=(self == "" || self.split(".").size() == 3)
	// +kubebuilder:validation:XValidation:message="openshiftVersion cannot start with a 'v'",rule=(!self.startsWith('v'))
	// OpenShift version used to provision the cluster with.  This is only used for initial provisioning
	// and ignored for future updates.  Version must be in format of x.y.z.  If this is empty, the latest
	// available and supportable version is selected.  If this is used, the version must be a part of
	// the 'stable' channel group.
	OpenShiftVersion string `json:"openshiftVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:message="multiAZ is immutable",rule=(self == oldSelf)
	// Whether the control plane should be provisioned across multiple availability zones (default: false).  Only
	// applicable when hostedControlPlane is set to false as hostedControlPlane is always
	// provisioned across multiple availability zones.
	MultiAZ bool `json:"multiAZ,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:message="enableFIPS is immutable",rule=(self == oldSelf)
	// Enable FIPS-compliant cryptography standards on the cluster.
	EnableFIPS bool `json:"enableFIPS,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:message="disableUserWorkloadMonitoring is immutable",rule=(self == oldSelf)
	// Enables you to monitor your own projects in isolation from Red Hat Site Reliability Engineer (SRE)
	// platform metrics.
	DisableUserWorkloadMonitoring bool `json:"disableUserWorkloadMonitoring,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=us-east-1
	// +kubebuilder:validation:XValidation:message="region is immutable",rule=(self == oldSelf)
	// Region used to provision the ROSA cluster.  Supported regions can be found using the
	// supportability checker located at https://access.redhat.com/labs/rosasc/.  Be aware of
	// valid region differences if using '.spec.hostedControlPlane = true'.
	Region string `json:"region,omitempty"`

	// +kubebuilder:validation:Required
	// Configuration of the default machine pool.
	DefaultMachinePool DefaultMachinePoolFields `json:"defaultMachinePool,omitempty"`

	// +kubebuilder:validation:Optional
	// Encryption configuration settings for the ROSA cluster.
	Encryption ROSAEncryption `json:"encryption,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="additionalTrustBundle is immutable",rule=(self == oldSelf)
	// PEM-encoded X.509 certificate bundle that will be added to each nodes trusted certificate store.
	AdditonalTrustBundle string `json:"additionalTrustBundle,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="tags is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="tags is limited to 10",rule=(self.size() <= 10)
	// +kubebuilder:validation:XValidation:message="red-hat-managed is a reserved tag",rule=!('red-hat-managed' in self)
	// +kubebuilder:validation:XValidation:message="red-hat-clustertype is a reserved tag",rule=!('red-hat-clustertype' in self)
	// Additional tags to apply to all AWS objects. Tags
	// are limited to 10 tags in total.  It should be noted that
	// there are reserved tags that may not be overwritten.  These
	// tags are as follows: red-hat-managed, red-hat-clustertype.
	Tags map[string]string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// ROSA Network configuration options.
	Network ROSANetwork `json:"network"`

	// +kubebuilder:validation:Optional
	// ROSA IAM configuration options including roles and prefixes.
	IAM ROSAIAM `json:"iam,omitempty"`
}

// ROSAEncryption defines the encryption configuration for the ROSA cluster.  It is used to set things like
// EBS encryption and ETCD encryption.
type ROSAEncryption struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="etcd.kmsKey must be a valid aws arn",rule=(self.kmsKey.startsWith("arn:aws"))
	// ETCD encryption configuration.  If specified, ETCD encryption is enabled.
	ETCD ROSAKey `json:"etcd,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="ebs.kmsKey must be a valid aws arn",rule=(self.kmsKey.startsWith("arn:aws"))
	// EBS encryption configuration.  EBS encryption is always enabled by default.  Allows you
	// to use a customer-managed key rather than the default account key.
	EBS ROSAKey `json:"ebs,omitempty"`
}

// ROSAKey is a generic, reusable configuration for AWS KMS Keys.
type ROSAKey struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="kmsKey is immutable",rule=(self == oldSelf)
	// KMS Key ARN to use.  Must be a valid and existing KMS Key ARN.
	Key string `json:"kmsKey,omitempty"`
}

// +kubebuilder:validation:XValidation:message="network.subnets must be provided with a PrivateLink configuration",rule=(has(self.privateLink) && self.privateLink && has(self.subnets) && self.subnets.size() > 0 || !self.privateLink)
// +kubebuilder:validation:XValidation:message="network.proxy configuration only supported when network.subnets is specified",rule=(has(self.proxy) && has(self.subnets) && self.subnets.size() > 0 || !has(self.proxy))
// ROSANetwork represents the ROSA network configuration.
//
//nolint:lll
type ROSANetwork struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:message="network.privateLink is immutable",rule=(self == oldSelf)
	// Enable PrivateLink (default: false).  Forces Red Hat SREs to connect to the cluster over an AWS PrivateLink
	// endpoint.  Requires a pre-existing network configuration and subnets configured
	// via the 'spec.network.subnets' field.
	PrivateLink bool `json:"privateLink,omitempty"`

	// +kubebuilder:validation:Optional
	// ROSA Proxy configuration.
	Proxy ROSAProxy `json:"proxy,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="network.subnets are immutable",rule=(self == oldSelf)
	// Pre-existing subnets used for provisioning a ROSA cluster.
	Subnets []string `json:"subnets,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="172.30.0.0/16"
	// +kubebuilder:validation:XValidation:message="network.serviceCIDR is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="network.serviceCIDR not a valid CIDR",rule=(self.split(".").size() == 4)
	// +kubebuilder:validation:XValidation:message="network.serviceCIDR not a valid CIDR",rule=(self.contains("/"))
	// CIDR to use for the internal cluster service network (default: 172.30.0.0/16).  Required if
	// subnets are not set so that the provisioner may create the network architecture.
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10.128.0.0/14"
	// +kubebuilder:validation:XValidation:message="network.podCIDR is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="network.podCIDR not a valid CIDR",rule=(self.split(".").size() == 4)
	// +kubebuilder:validation:XValidation:message="network.podCIDR not a valid CIDR",rule=(self.contains("/"))
	// CIDR to use for the internal pod network (default: 10.128.0.0/14).  Required if
	// subnets are not set so that the provisioner may create the network architecture.
	PodCIDR string `json:"podCIDR,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10.0.0.0/16"
	// +kubebuilder:validation:XValidation:message="network.machineCIDR is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="network.machineCIDR not a valid CIDR",rule=(self.split(".").size() == 4)
	// +kubebuilder:validation:XValidation:message="network.machineCIDR not a valid CIDR",rule=(self.contains("/"))
	// CIDR to use for the AWS VPC (default: 10.0.0.0/16).  Required if
	// subnets are not set so that the provisioner may create the network architecture.
	MachineCIDR string `json:"machineCIDR,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=23
	// +kubebuilder:validation:XValidation:message="network.hostPrefix is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="network.hostPrefix has a minimum size of /28",rule=(self < 28)
	// Host CIDR Prefix (e.g. /23) to use for host IP addresses within the 'network.machineCIDR' subnet (default: 23).
	// Minimum size available to use is 28 (used as a /28 CIDR Prefix).
	HostPrefix int `json:"hostPrefix,omitempty"`
}

// ROSAProxy represents the ROSA proxy configuration.
type ROSAProxy struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="network.proxy.httpProxy is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="http proxy url must be a valid uri",rule=(self.contains("://"))
	// Valid proxy URL to use for proxying HTTP requests from within the cluster.
	HTTPProxy string `json:"httpProxy,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="network.proxy.httpsProxy is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="https proxy url must be a valid uri",rule=(self.contains("://"))
	// Valid proxy URL to use for proxying HTTPS requests from within the cluster.
	HTTPSProxy string `json:"httpsProxy,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="network.proxy.noProxy is immutable",rule=(self == oldSelf)
	// Comma-separated list of URLs, IP addresses or Network CIDRs to skip proxying for.
	NoProxy string `json:"noProxy,omitempty"`
}

// ROSAIAM represents the ROSA IAM Roles configuration.
type ROSAIAM struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +kubebuilder:validation:Enum=false
	// +kubebuilder:validation:XValidation:message="iam.enableManagedPolicies is immutable",rule=(self == oldSelf)
	// Use policies that are natively managed by AWS.  NOTE: currently this is not possible as the
	// OCM API does not return valid ARNs.  Only 'false' is an option for now.
	EnableManagedPolicies bool `json:"enableManagedPolicies,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="iam.operatorRolesPrefix is immutable",rule=(self == oldSelf)
	// Prefix used for provisioned operator roles.  Defaults to using the cluster name with a randomly
	// generated 6-digit ID.  These will be created as part of the cluster creation process.
	OperatorRolesPrefix string `json:"operatorRolesPrefix,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="ManagedOpenShift"
	// +kubebuilder:validation:XValidation:message="iam.accountRolesPrefix is immutable",rule=(self == oldSelf)
	// +kubebuilder:validation:XValidation:message="accountRolesPrefix may not be blank",rule=(self != "")
	// Prefix used for provisioned account roles (default: ManagedOpenShift).  These should have been created as part of
	// the prerequisite 'rosa create account-roles' step.
	AccountRolesPrefix string `json:"accountRolesPrefix,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:message="iam.userRole is immutable",rule=(self == oldSelf)
	// User role created with the prerequisite 'rosa create user-role' step.  This is the value used
	// as the 'rosa_creator_arn' for the cluster.
	UserRole string `json:"userRole,omitempty"`
}

// ROSAClusterStatus defines the observed state of ROSACluster.
type ROSAClusterStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.clusterID is immutable",rule=(self == oldSelf)
	// Represents the programmatic cluster ID of the cluster, as
	// determined during reconciliation.  This is used to reduce
	// the number of API calls to look up a cluster ID based on
	// the cluster name.
	ClusterID string `json:"clusterID,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.oidcConfigID is immutable",rule=(self == oldSelf)
	// Represents the programmatic OIDC Config ID of the cluster, as
	// determined during reconciliation.  This is used to reduce
	// the number of API calls to look up a cluster ID based on
	// the cluster name.
	OIDCConfigID string `json:"oidcConfigID,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.OIDCProviderARN is immutable",rule=(self == oldSelf)
	// Represents the AWS ARN for the OIDC provider.  This is only
	// set after the provider is created.
	OIDCProviderARN string `json:"oidcProviderARN,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.openshiftVersion is immutable",rule=(self == oldSelf)
	// Represents the OpenShift OCM Version Raw ID which was used
	// to provision the cluster.  This is useful if the version
	// is unset to reduce the amount of calls to the OCM API.
	OpenShiftVersion string `json:"openshiftVersion,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.openshiftVersion is immutable",rule=(self == oldSelf)
	// Represents the OpenShift OCM Version ID which was used
	// to provision the cluster.  This is used to reduce
	// the number of API calls to the OCM API.  This will differ
	// from the 'spec.openshiftVersion' field.
	OpenShiftVersionID string `json:"openshiftVersionID,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.operatorRolesCreated is immutable",rule=(self == oldSelf)
	// Represents whether the operator roles have been created or not.
	// This is used to ensure that we do not attempt to recreate operator
	// roles once they have already been created.
	OperatorRolesCreated bool `json:"operatorRolesCreated,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.operatorRolesPrefix is immutable",rule=(self == oldSelf)
	// The operator roles prefix.  if 'spec.iam.operatorRolesPrefix' is
	// unset, this is the derived value containing a unique id which
	// will be unknown to the requester.
	OperatorRolesPrefix string `json:"operatorRolesPrefix,omitempty"`
}

// +kubebuilder:resource:categories=cluster;clusters
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:validation:XValidation:message="metadata.name limited to 15 characters",rule=(self.metadata.name.size() <= 15)

// ROSACluster is the Schema for the clusters API.
type ROSACluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ROSAClusterSpec   `json:"spec,omitempty"`
	Status ROSAClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ROSAClusterList contains a list of Cluster.
type ROSAClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ROSACluster `json:"items"`
}

// GetConditions returns the status.conditions field from the object.  It is used to
// satisfy the Workload interface.
func (cluster *ROSACluster) GetConditions() []metav1.Condition {
	return cluster.Status.Conditions
}

// SetConditions sets the status.conditions field from the object.  It is used to
// satisfy the Workload interface.
func (cluster *ROSACluster) SetConditions(conditions []metav1.Condition) {
	cluster.Status.Conditions = conditions
}

// CopyFrom copies the current state of an OCM cluster object into a ROSACluster object.
func (cluster *ROSACluster) CopyFrom(source *clustersmgmtv1.Cluster) {
	// openshift/rosa settings
	cluster.Spec.HostedControlPlane = source.Hypershift().Enabled()
	cluster.Spec.OpenShiftVersion = source.Version().RawID()
	cluster.Spec.DisableUserWorkloadMonitoring = source.DisableUserWorkloadMonitoring()

	// basic aws settings
	cluster.Spec.AccountID = source.AWS().AccountID()
	cluster.Spec.MultiAZ = source.MultiAZ()
	cluster.Spec.Region = source.Region().ID()
	cluster.Spec.Tags = source.AWS().Tags()
	cluster.Spec.IAM.OperatorRolesPrefix = source.AWS().STS().OperatorRolePrefix()

	// encryption/cryptography settings
	cluster.Spec.EnableFIPS = source.FIPS()
	cluster.Spec.Encryption.EBS.Key = source.AWS().KMSKeyArn()
	cluster.Spec.Encryption.ETCD.Key = source.AWS().EtcdEncryption().KMSKeyARN()
	cluster.Spec.AdditonalTrustBundle = source.AdditionalTrustBundle()

	// machine pool settings
	cluster.Spec.DefaultMachinePool.InstanceType = source.Nodes().ComputeMachineType().ID()
	cluster.Spec.DefaultMachinePool.Labels = source.Nodes().ComputeLabels()

	if source.Nodes().AutoscaleCompute().MaxReplicas() > 0 {
		cluster.Spec.DefaultMachinePool.MinimumNodesPerZone = source.Nodes().AutoscaleCompute().MinReplicas() / cluster.GetAvailabilityZoneCount()
		cluster.Spec.DefaultMachinePool.MaximumNodesPerZone = source.Nodes().AutoscaleCompute().MaxReplicas() / cluster.GetAvailabilityZoneCount()
	} else {
		cluster.Spec.DefaultMachinePool.MinimumNodesPerZone = source.Nodes().Compute()
	}

	// network settings
	cluster.Spec.Network.PrivateLink = source.AWS().PrivateLink()
	cluster.Spec.Network.Proxy.HTTPProxy = source.Proxy().HTTPProxy()
	cluster.Spec.Network.Proxy.HTTPSProxy = source.Proxy().HTTPSProxy()
	cluster.Spec.Network.Proxy.NoProxy = source.Proxy().NoProxy()
	cluster.Spec.Network.Subnets = source.AWS().SubnetIDs()
	cluster.Spec.Network.HostPrefix = source.Network().HostPrefix()
	cluster.Spec.Network.ServiceCIDR = source.Network().ServiceCIDR()
	cluster.Spec.Network.PodCIDR = source.Network().PodCIDR()
	cluster.Spec.Network.MachineCIDR = source.Network().MachineCIDR()

	// iam settings
	cluster.Spec.IAM.UserRole = source.Properties()[rosaPropertyUserRole]
	cluster.Spec.IAM.OperatorRolesPrefix = source.AWS().STS().OperatorRolePrefix()
	cluster.Spec.IAM.AccountRolesPrefix = getAccountRolesPrefix(source)
}

// Builder builds an object that is used for create and update operations.
func (cluster *ROSACluster) Builder(oidcConfig *clustersmgmtv1.OidcConfig, versionID string) *clustersmgmtv1.ClusterBuilder {
	// create the base builder
	builder := clustersmgmtv1.NewCluster().
		// openshift/rosa settings
		Name(cluster.Spec.DisplayName).
		Product(clustersmgmtv1.NewProduct().ID(rosaProduceID)).
		Hypershift(clustersmgmtv1.NewHypershift().Enabled(cluster.Spec.HostedControlPlane)).
		DisableUserWorkloadMonitoring(cluster.Spec.DisableUserWorkloadMonitoring).
		Properties(
			map[string]string{
				rosaPropertyUserRole:    cluster.Spec.IAM.UserRole,
				rosaPropertyProvisioner: rosaPropertyProvisionerOperator,
			},
		).
		Version(clustersmgmtv1.NewVersion().ID(versionID)).

		// basic aws settings
		MultiAZ(cluster.Spec.MultiAZ).
		Region(clustersmgmtv1.NewCloudRegion().ID(cluster.Spec.Region)).

		// encryption/cryptography settings
		FIPS(cluster.Spec.EnableFIPS).
		EtcdEncryption(cluster.Spec.Encryption.ETCD.Key != "")

	// add trust bundle if specified
	if cluster.Spec.AdditonalTrustBundle != "" {
		builder.AdditionalTrustBundle(cluster.Spec.AdditonalTrustBundle)
	}

	// only add the proxy builder if we have proxy settings
	if cluster.HasProxy() {
		builder.Proxy(cluster.BuildProxy())
	}

	// only add the network builder if we have not specified a
	// preconfigured network architecture via the network.subnets field.
	if len(cluster.Spec.Network.Subnets) == 0 {
		builder.Network(cluster.BuildNetwork())
	}

	builder.AWS(cluster.BuildAWS(oidcConfig)).Nodes(cluster.BuildClusterNodes())

	return builder
}

func (cluster *ROSACluster) BuildProxy() *clustersmgmtv1.ProxyBuilder {
	if !cluster.HasProxy() {
		return nil
	}

	proxyBuilder := clustersmgmtv1.NewProxy()
	if cluster.Spec.Network.Proxy.HTTPProxy != "" {
		proxyBuilder.HTTPProxy(cluster.Spec.Network.Proxy.HTTPProxy)
	}

	if cluster.Spec.Network.Proxy.HTTPSProxy != "" {
		proxyBuilder.HTTPSProxy(cluster.Spec.Network.Proxy.HTTPSProxy)
	}

	if cluster.Spec.Network.Proxy.NoProxy != "" {
		proxyBuilder.NoProxy(cluster.Spec.Network.Proxy.NoProxy)
	}

	return proxyBuilder
}

func (cluster *ROSACluster) BuildAWS(oidcConfig *clustersmgmtv1.OidcConfig) *clustersmgmtv1.AWSBuilder {
	awsBuilder := clustersmgmtv1.NewAWS().
		PrivateLink(cluster.Spec.Network.PrivateLink).
		AccountID(cluster.Spec.AccountID).
		STS(clustersmgmtv1.NewSTS().
			OidcConfig(clustersmgmtv1.NewOidcConfig().Copy(oidcConfig)).
			OperatorRolePrefix(cluster.Spec.IAM.OperatorRolesPrefix).
			RoleARN(cluster.GetInstallerRole()).
			SupportRoleARN(cluster.GetSupportRole()).
			InstanceIAMRoles(clustersmgmtv1.NewInstanceIAMRoles().
				WorkerRoleARN(cluster.GetWorkerRole()).
				MasterRoleARN(cluster.GetControlPlaneRole()),
			).
			AutoMode(true),
		)

	// add tags if specified
	if len(cluster.Spec.Tags) > 0 {
		awsBuilder.Tags(cluster.Spec.Tags)
	}

	// add etcd encryption if specified
	if cluster.Spec.Encryption.ETCD.Key != "" {
		awsBuilder.EtcdEncryption(clustersmgmtv1.NewAwsEtcdEncryption().KMSKeyARN(cluster.Spec.Encryption.ETCD.Key))
	}

	// add ebs customer-managed key if specified
	if cluster.Spec.Encryption.EBS.Key != "" {
		awsBuilder.KMSKeyArn(cluster.Spec.Encryption.EBS.Key)
	}

	// add subnet ids if specified
	if len(cluster.Spec.Network.Subnets) > 0 {
		awsBuilder.SubnetIDs(cluster.Spec.Network.Subnets...)
	}

	return awsBuilder
}

func (cluster *ROSACluster) BuildClusterNodes() *clustersmgmtv1.ClusterNodesBuilder {
	nodeBuilder := clustersmgmtv1.NewClusterNodes().
		ComputeMachineType(clustersmgmtv1.NewMachineType().Name(cluster.Spec.DefaultMachinePool.InstanceType))

	// add labels if specified
	if len(cluster.Spec.DefaultMachinePool.Labels) > 0 {
		nodeBuilder.ComputeLabels(cluster.Spec.DefaultMachinePool.Labels)
	}

	// machine pool settings
	if cluster.Spec.DefaultMachinePool.MaximumNodesPerZone > 0 {
		// add autoscaling min/max replicas
		nodeBuilder.AutoscaleCompute(
			clustersmgmtv1.NewMachinePoolAutoscaling().
				MinReplicas(cluster.GetMachinePoolMinimumNodes()).
				MaxReplicas(cluster.GetMachinePoolMaximumNodes()),
		)
	} else {
		// add the base count
		nodeBuilder.Compute(cluster.GetMachinePoolMinimumNodes())
	}

	return nodeBuilder
}

func (cluster *ROSACluster) BuildNetwork() *clustersmgmtv1.NetworkBuilder {
	return clustersmgmtv1.NewNetwork().
		HostPrefix(cluster.Spec.Network.HostPrefix).
		ServiceCIDR(cluster.Spec.Network.ServiceCIDR).
		PodCIDR(cluster.Spec.Network.PodCIDR).
		MachineCIDR(cluster.Spec.Network.MachineCIDR)
}

// HasProxy determines if a cluster has a proxy configuration or not.
func (cluster *ROSACluster) HasProxy() bool {
	if cluster.Spec.Network.Proxy.HTTPProxy != "" {
		return true
	}

	if cluster.Spec.Network.Proxy.HTTPSProxy != "" {
		return true
	}

	if cluster.Spec.Network.Proxy.NoProxy != "" {
		return true
	}

	return false
}

func (cluster *ROSACluster) GetAvailabilityZoneCount() int {
	// we return the single az because machine pools for a
	// hosted control plan exist only within a single availability zone
	if cluster.Spec.HostedControlPlane {
		return rosaSingleAZCount
	}

	if cluster.Spec.MultiAZ {
		return rosaMultiAZCount
	}

	return rosaSingleAZCount
}

func (cluster *ROSACluster) GetControlPlaneCount() int {
	if cluster.Spec.HostedControlPlane {
		return rosaHostedControlPlaneCount
	}

	if cluster.Spec.MultiAZ {
		return rosaMultiAZCount
	}

	return rosaSingleAZCount
}

func (cluster *ROSACluster) GetInfraCount() int {
	if cluster.Spec.HostedControlPlane {
		return rosaHostedControlPlaneInfraCount
	}

	if cluster.Spec.MultiAZ {
		return rosaMultiAZCount
	}

	return rosaSingleAZCount
}

func (cluster *ROSACluster) GetMachinePoolMinimumNodes() int {
	if cluster.Spec.MultiAZ {
		return cluster.Spec.DefaultMachinePool.MinimumNodesPerZone * rosaMultiAZCount
	}

	return cluster.Spec.DefaultMachinePool.MinimumNodesPerZone * rosaSingleAZCount
}

func (cluster *ROSACluster) GetMachinePoolMaximumNodes() int {
	if cluster.Spec.MultiAZ {
		return cluster.Spec.DefaultMachinePool.MaximumNodesPerZone * rosaMultiAZCount
	}

	return cluster.Spec.DefaultMachinePool.MaximumNodesPerZone * rosaSingleAZCount
}

func (cluster *ROSACluster) GetInstallerRole() string {
	return cluster.getIAMRoleName(rosaInstallerRolePrefix)
}

func (cluster *ROSACluster) GetSupportRole() string {
	return cluster.getIAMRoleName(rosaSupportRolePrefix)
}

func (cluster *ROSACluster) GetControlPlaneRole() string {
	return cluster.getIAMRoleName(rosaControlPlaneRolePrefix)
}

func (cluster *ROSACluster) GetWorkerRole() string {
	return cluster.getIAMRoleName(rosaWorkerRolePrefix)
}

func (cluster *ROSACluster) SetNetworkDefaults() {
	if cluster.Spec.Network.HostPrefix == 0 {
		cluster.Spec.Network.HostPrefix = rosaDefaultHostPrefix
	}

	if cluster.Spec.Network.MachineCIDR == "" {
		cluster.Spec.Network.MachineCIDR = rosaDefaultMachineCIDR
	}

	if cluster.Spec.Network.ServiceCIDR == "" {
		cluster.Spec.Network.ServiceCIDR = rosaDefaultServiceCIDR
	}

	if cluster.Spec.Network.PodCIDR == "" {
		cluster.Spec.Network.PodCIDR = rosaDefaultPodCIDR
	}
}

// getAccountRolesPrefix is a helper function to determine the prefix of the
// account roles.
func getAccountRolesPrefix(cluster *clustersmgmtv1.Cluster) string {
	installerRole := strings.Split(cluster.AWS().STS().RoleARN(), "/")[1]

	return strings.Split(installerRole, "-")[0]
}

// getIAMRoleName is a helper function for each of the Get*Role methods.
func (cluster *ROSACluster) getIAMRoleName(roleType string) string {
	return fmt.Sprintf(
		"arn:aws:iam::%s:role/%s-%s-Role",
		cluster.Spec.AccountID,
		cluster.Spec.IAM.AccountRolesPrefix,
		roleType,
	)
}

func init() {
	SchemeBuilder.Register(&ROSACluster{}, &ROSAClusterList{})
}
