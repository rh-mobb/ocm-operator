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
	"context"
	"fmt"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:XValidation:message="maximumNodesPerZone must be greater than or equal to minimumNodesPerZone",rule=(self.maximumNodesPerZone == 0 || self.minimumNodesPerZone <= self.maximumNodesPerZone)
// MachinePoolSpec defines the desired state of MachinePool.
//
//nolint:lll
type MachinePoolSpec struct {
	DefaultMachinePoolFields `json:",inline"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:message="clusterName is immutable",rule=(self == oldSelf)
	// Cluster name in OpenShift Cluster Manager by which this should be managed for.  A cluster with this
	// name should exist in the organization by which the operator is associated.  If the cluster does
	// not exist, the reconciliation process will continue until one does.
	ClusterName string `json:"clusterName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=4
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:XValidation:message="displayName is immutable",rule=(self == oldSelf)
	// Friendly display name as displayed in the OpenShift Cluster Manager
	// console.  If this is empty, the metadata.name field of the parent resource is used
	// to construct the display name.  This is limited to 15 characters as per the backend
	// API limitation.
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:Optional
	// Taints that should be applied to this machine pool.  For information please see
	// https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/.
	Taints []corev1.Taint `json:"taints,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Wait for the machine pool to enter a ready state.  If this is set to true, it is
	// assumed that the operator is running in the cluster that machine pools are
	// being controlled for.  This is due to a limitation in the OCM API which does
	// not expose the ready state of a machine pool.  If this is set to false, the
	// reconciler will perform a "fire and forget" approach and assume if the object
	// is created, it will eventually be correctly reconciled.
	Wait bool `json:"wait,omitempty"`

	// +kubebuilder:validation:Optional
	// Represents the AWS provider specific configuration options.
	AWS MachinePoolProviderAWS `json:"aws,omitempty"`
}

// DefaultMachinePoolFields represents the fields relevant to the default machine pool.  It
// is broken out as a separate object to allow other types of machine pools to use this
// struct inheritance as well.
//
//nolint:lll
type DefaultMachinePoolFields struct {
	// +kubebuilder:validation:Required
	// Minimum amount of nodes allowed per availability zone.  For single availability zone
	// clusters, the minimum allowed is 2 per zone.  For multiple availability zone clusters,
	// the minimum allowed is 1 per zone.  If spec.maximumNodesPerZone is also set,
	// autoscaling will be enabled for this machine pool.
	MinimumNodesPerZone int `json:"minimumNodesPerZone,omitempty"`

	// +kubebuilder:validation:Optional
	// Maximum amount of nodes allowed per availability zone.  Must be greater than or equal
	// to spec.minimumNodesPerZone.  If this field is set, autoscaling will be enabled
	// for this machine pool.
	MaximumNodesPerZone int `json:"maximumNodesPerZone,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default="m5.xlarge"
	// +kubebuilder:validation:XValidation:message="instanceType is immutable",rule=(self == oldSelf)
	// Instance type to use for all nodes within this MachinePool.  Please see the following for
	// a list of supported instance types based on the provider type (ROSA/OSD only supported for now):
	//
	// *ROSA/OSD: https://docs.openshift.com/rosa/rosa_architecture/rosa_policy_service_definition/rosa-service-definition.html
	InstanceType string `json:"instanceType,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="ocm.mobb.redhat.com/name is a reserved label",rule=!('ocm.mobb.redhat.com/name' in self)
	// +kubebuilder:validation:XValidation:message="ocm.mobb.redhat.com/managed is a reserved label",rule=!('ocm.mobb.redhat.com/managed' in self)
	// Additional labels to apply to this MachinePool.  It should be noted that
	// 'ocm.mobb.redhat.com/managed' = 'true' is automatically applied as well
	// as 'ocm.mobb.redhat.com/name' = spec.displayName.  Both of these labels
	// are reserved and cannot be used as part of the spec.labels field.
	Labels map[string]string `json:"labels,omitempty"`
}

// MachinePoolProviderAWS represents the provider specific configuration for an AWS provider.
type MachinePoolProviderAWS struct {
	// +kubebuilder:validation:Optional
	// Configuration of AWS Spot Instances for this MachinePool.  This section
	// is not valid and is ignored if the cluster is using hosted
	// control plane.
	SpotInstances MachinePoolProviderAWSSpotInstances `json:"spotInstances,omitempty"`
}

// MachinePoolProviderAWSSpotInstances represents the AWS Spot Instance configuration.
type MachinePoolProviderAWSSpotInstances struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="aws.spotInstances.enabled is immutable",rule=(self == oldSelf)
	// Request spot instances when scaling up this MachinePool.  If enabled a maximum
	// price for the spot instances may be set in spec.aws.spotInstances.maximumPrice.
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:message="aws.spotInstances.maximumPrice is immutable",rule=(self == oldSelf)
	// Maximum price to pay for spot instance.
	// To be used with spec.aws.spotInstances.enabled. If no maximum price is set,
	// the spot instance configuration defaults to on-demand pricing.
	MaximumPrice int `json:"maximumPrice,omitempty"`
}

// MachinePoolStatus defines the observed state of MachinePool.
type MachinePoolStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.clusterID is immutable",rule=(self == oldSelf)
	// Represents the programmatic cluster ID of the cluster, as
	// determined during reconciliation.  This is used to reduce
	// the number of API calls to look up a cluster ID based on
	// the cluster name.
	ClusterID string `json:"clusterID,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.AvailabilityZoneCount is immutable",rule=(self == oldSelf)
	// Represents the number of availability zones that the cluster
	// resides in.  Used to calculate the total number of replicas.
	AvailabilityZones []string `json:"availabilityZones,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.Subnets is immutable",rule=(self == oldSelf)
	// Represents the subnets where the cluster is provisioned.
	Subnets []string `json:"subnets,omitempty"`

	// +kubebuilder:validation:XValidation:message="status.Hosted is immutable",rule=(self == oldSelf)
	// Whether this cluster is using a hosted control plane.
	Hosted bool `json:"hosted,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:validation:XValidation:message="metadata.name limited to 15 characters",rule=(self.metadata.name.size() <= 15)

// MachinePool is the Schema for the machinepools API.
type MachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachinePoolSpec   `json:"spec,omitempty"`
	Status MachinePoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MachinePoolList contains a list of MachinePool.
type MachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachinePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachinePool{}, &MachinePoolList{})
}

// FindAll gets a complete list of resources in the cluster for this type.
func (machinePool *MachinePool) FindAll(
	ctx context.Context,
	c kubernetes.Client,
) ([]MachinePool, error) {
	objects := &MachinePoolList{}

	if err := c.List(ctx, objects); err != nil {
		return []MachinePool{}, fmt.Errorf("unable to retrieve machine pools - %w", err)
	}

	return objects.Items, nil
}

// FindAllByClusterID gets a list of resources which have a particular cluster ID in the status field.
func (machinePool *MachinePool) FindAllByClusterID(
	ctx context.Context,
	c kubernetes.Client,
	clusterID string,
) ([]*MachinePool, error) {
	objects, err := machinePool.FindAll(ctx, c)
	if err != nil {
		return []*MachinePool{}, err
	}

	matches := []*MachinePool{}

	for i := range objects {
		if objects[i].Status.ClusterID == clusterID {
			matches = append(matches, &objects[i])
		}
	}

	return matches, nil
}

// ExistsForClusterID returns if a particular object is associated with a cluster ID.
func (machinePool *MachinePool) ExistsForClusterID(
	ctx context.Context,
	c kubernetes.Client,
	clusterID string,
) (bool, error) {
	objects, err := machinePool.FindAllByClusterID(ctx, c, clusterID)

	return (len(objects) > 0), err
}

// DesiredState returns the desired state of an object that should exist in
// OCM.  This is required because there are certain things that get set
// that are not a part of the spec such as managed labels.
func (machinePool *MachinePool) DesiredState() *MachinePool {
	desiredState := machinePool.DeepCopy()

	// set the display name
	desiredState.Spec.DisplayName = desiredState.GetDisplayName()

	// set the managed labels on the desired state.  we do this because we expect
	// that the current state should have these labels.
	desiredState.SetMachinePoolLabels()

	return desiredState
}

// GetClusterID gets the status.clusterID field from the object.  It is used to
// satisfy the Workload interface.
func (machinePool *MachinePool) GetClusterID() string {
	return machinePool.Status.ClusterID
}

// GetConditions returns the status.conditions field from the object.  It is used to
// satisfy the Workload interface.
func (machinePool *MachinePool) GetConditions() []metav1.Condition {
	return machinePool.Status.Conditions
}

// SetConditions sets the status.conditions field from the object.  It is used to
// satisfy the Workload interface.
func (machinePool *MachinePool) SetConditions(conditions []metav1.Condition) {
	machinePool.Status.Conditions = conditions
}

// GetDisplayName returns the name for the OCM MachinePool.  It defaults to wanting to use
// the spec.displayName field but returns the metadata.name field if unset.
func (machinePool *MachinePool) GetDisplayName() string {
	if machinePool.Spec.DisplayName == "" {
		return machinePool.GetName()
	}

	return machinePool.Spec.DisplayName
}

// SetMachinePoolLabels sets the required labels on the object.
func (machinePool *MachinePool) SetMachinePoolLabels() {
	// create the labels if unset
	if machinePool.Spec.Labels == nil {
		machinePool.Spec.Labels = map[string]string{}
	}

	// set the managed label
	machinePool.Spec.Labels[ocm.LabelPrefixManaged] = "true"

	// set the name label
	machinePool.Spec.Labels[ocm.LabelPrefixName] = machinePool.GetDisplayName()
}

// HasManagedLabels determines if the MachinePool object has the appropriate managed
// labels in its desired state.
func (machinePool *MachinePool) HasManagedLabels() bool {
	for _, label := range ocm.ManagedLabels() {
		if machinePool.Spec.Labels[label] == "" {
			return false
		}
	}

	return true
}

// CopyFromMachinePool copies an OCM MachinePool object into a MachinePool object that is recognizable by this
// controller.
func (machinePool *MachinePool) CopyFromMachinePool(source *clustersmgmtv1.MachinePool, clusterName string) error {
	machinePool.Spec.ClusterName = clusterName
	machinePool.Spec.DisplayName = source.ID()
	machinePool.Spec.InstanceType = source.InstanceType()
	machinePool.Spec.Labels = source.Labels()
	machinePool.Spec.Taints = copyTaints(source.Taints())
	machinePool.Spec.MinimumNodesPerZone = copyMachinePoolMinimumNodesPerZone(source)
	machinePool.Spec.MaximumNodesPerZone = copyMachinePoolMaximumNodesPerZone(source)
	machinePool.Spec.AWS = copyAWSConfig(source.AWS())

	return nil
}

// CopyFromNodePool copies an OCM NodePool object into a MachinePool object that is recognizable by this
// controller.
func (machinePool *MachinePool) CopyFromNodePool(source *clustersmgmtv1.NodePool, clusterName string) error {
	machinePool.Spec.ClusterName = clusterName
	machinePool.Spec.DisplayName = source.ID()
	machinePool.Spec.InstanceType = source.AWSNodePool().InstanceType()
	machinePool.Spec.Labels = source.Labels()
	machinePool.Spec.Taints = copyTaints(source.Taints())
	machinePool.Spec.MinimumNodesPerZone = copyNodePoolMinimumNodesPerZone(source)
	machinePool.Spec.MaximumNodesPerZone = copyNodePoolMaximumNodesPerZone(source)

	// spot instances for node pools are not an option
	machinePool.Spec.AWS = MachinePoolProviderAWS{}

	return nil
}

// MachinePoolBuilder builds an OCM MachinePoolBuilder object.
func (machinePool *MachinePool) MachinePoolBuilder() *clustersmgmtv1.MachinePoolBuilder {
	builder := clustersmgmtv1.NewMachinePool().
		ID(machinePool.Spec.DisplayName).
		InstanceType(machinePool.Spec.InstanceType).
		Labels(machinePool.Spec.Labels).
		Taints(machinePool.convertTaints()...).
		AWS(machinePool.convertAWSMachinePool())

	if machinePool.Spec.MaximumNodesPerZone > 0 {
		builder = builder.Autoscaling(machinePool.convertMachinePoolAutoscaling())
	} else {
		builder = builder.Replicas(machinePool.Spec.MinimumNodesPerZone)
	}

	return builder
}

// NodePoolBuilder builds an OCM NodePoolBuilder object.
func (machinePool *MachinePool) NodePoolBuilder() *clustersmgmtv1.NodePoolBuilder {
	builder := clustersmgmtv1.NewNodePool().
		ID(machinePool.Spec.DisplayName).
		Labels(machinePool.Spec.Labels).
		Taints(machinePool.convertTaints()...).
		AWSNodePool(clustersmgmtv1.NewAWSNodePool().InstanceType(machinePool.Spec.InstanceType))

	if machinePool.Spec.MaximumNodesPerZone > 0 {
		builder = builder.Autoscaling(machinePool.convertNodePoolAutoscaling())
	} else {
		builder = builder.Replicas(machinePool.Spec.MinimumNodesPerZone)
	}

	return builder
}

func (machinePool *MachinePool) convertTaints() (builders []*clustersmgmtv1.TaintBuilder) {
	if len(machinePool.Spec.Taints) < 1 {
		return builders
	}

	taints := make([]*clustersmgmtv1.TaintBuilder, len(machinePool.Spec.Taints))

	for i, source := range machinePool.Spec.Taints {
		taints[i] = clustersmgmtv1.NewTaint().
			Key(source.Key).
			Value(source.Value).
			Effect(string(source.Effect))
	}

	return taints
}

func (machinePool *MachinePool) convertMachinePoolAutoscaling() (builder *clustersmgmtv1.MachinePoolAutoscalingBuilder) {
	if machinePool.Spec.MaximumNodesPerZone > 0 {
		return clustersmgmtv1.NewMachinePoolAutoscaling().
			MinReplicas(machinePool.Spec.MinimumNodesPerZone * machinePool.availabilityZoneCount()).
			MaxReplicas(machinePool.Spec.MaximumNodesPerZone * machinePool.availabilityZoneCount())
	}

	return clustersmgmtv1.NewMachinePoolAutoscaling()
}

func (machinePool *MachinePool) convertNodePoolAutoscaling() (builder *clustersmgmtv1.NodePoolAutoscalingBuilder) {
	if machinePool.Spec.MaximumNodesPerZone > 0 {
		return clustersmgmtv1.NewNodePoolAutoscaling().
			MinReplica(machinePool.Spec.MinimumNodesPerZone * machinePool.availabilityZoneCount()).
			MaxReplica(machinePool.Spec.MaximumNodesPerZone * machinePool.availabilityZoneCount())
	}

	return clustersmgmtv1.NewNodePoolAutoscaling()
}

func (machinePool *MachinePool) convertAWSMachinePool() (builder *clustersmgmtv1.AWSMachinePoolBuilder) {
	if machinePool.Spec.AWS.SpotInstances.Enabled {
		if machinePool.Spec.AWS.SpotInstances.MaximumPrice > 0 {
			return clustersmgmtv1.NewAWSMachinePool().
				SpotMarketOptions(
					clustersmgmtv1.NewAWSSpotMarketOptions().
						MaxPrice(float64(machinePool.Spec.AWS.SpotInstances.MaximumPrice)),
				)
		} else {
			return clustersmgmtv1.NewAWSMachinePool().
				SpotMarketOptions(&clustersmgmtv1.AWSSpotMarketOptionsBuilder{})
		}
	}

	return builder
}

func (machinePool *MachinePool) availabilityZoneCount() int {
	return len(machinePool.Status.AvailabilityZones)
}

func copyTaints(source []*clustersmgmtv1.Taint) (taints []corev1.Taint) {
	if len(source) < 1 {
		return taints
	}

	for i := range source {
		taints = append(taints, corev1.Taint{
			Key:    source[i].Key(),
			Value:  source[i].Value(),
			Effect: corev1.TaintEffect(source[i].Effect()),
		})
	}

	return taints
}

func copyMachinePoolMinimumNodesPerZone(source *clustersmgmtv1.MachinePool) int {
	if source.Autoscaling().MaxReplicas() > 0 {
		return (source.Autoscaling().MinReplicas() / len(source.AvailabilityZones()))
	}

	return (source.Replicas() / len(source.AvailabilityZones()))
}

func copyMachinePoolMaximumNodesPerZone(source *clustersmgmtv1.MachinePool) int {
	if source.Autoscaling().MaxReplicas() > 0 {
		return (source.Autoscaling().MaxReplicas() / len(source.AvailabilityZones()))
	}

	return 0
}

func copyNodePoolMinimumNodesPerZone(source *clustersmgmtv1.NodePool) int {
	if source.Autoscaling().MaxReplica() > 0 {
		// TODO: if node pools are provisioned in multiple azs, this will break.  does
		// not seem possible today.
		return source.Autoscaling().MinReplica()
	}

	return source.Replicas()
}

func copyNodePoolMaximumNodesPerZone(source *clustersmgmtv1.NodePool) int {
	if source.Autoscaling().MaxReplica() > 0 {
		// TODO: if node pools are provisioned in multiple azs, this will break.  does
		// not seem possible today.
		return source.Autoscaling().MaxReplica()
	}

	return 0
}

func copyAWSConfig(source *clustersmgmtv1.AWSMachinePool) MachinePoolProviderAWS {
	if source == nil {
		return MachinePoolProviderAWS{}
	}

	return MachinePoolProviderAWS{
		SpotInstances: MachinePoolProviderAWSSpotInstances{
			Enabled:      true,
			MaximumPrice: int(source.SpotMarketOptions().MaxPrice()),
		},
	}
}
