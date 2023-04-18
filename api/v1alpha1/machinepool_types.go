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
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MachinePoolSpec defines the desired state of MachinePool
type MachinePoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=4
	// +kubebuilder:validation:MaxLength=64
	// Friendly display name of the machine pool as displayed in the OpenShift Cluster Manager
	// console.  If this is empty, the metadata.name field of the parent resource is used
	// to construct the display name.
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:Required
	// Cluster ID in OpenShift Cluster Manager by which this MachinePool should be managed for.  The cluster ID
	// can be obtained on the Clusters page for the individual cluster.  It may also be known as the
	// 'External ID' in some CLI clients.  It shows up in the format of 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
	// where the 'x' represents any alphanumeric character.
	ClusterName string `json:"clusterName,omitempty"`

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
	// Instance type to use for all nodes within this MachinePool.  Please see the following for
	// a list of supported instance types based on the provider type (ROSA/OSD only supported for now):
	//
	// *ROSA/OSD: https://docs.openshift.com/rosa/rosa_architecture/rosa_policy_service_definition/rosa-service-definition.html
	InstanceType string `json:"instanceType,omitempty"`

	// +kubebuilder:validation:Optional
	// Additional labels to apply to this MachinePool.  It should be noted that
	// 'ocm.mobb.redhat.com/managed' = 'true' is automatically applied.
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	// Taints that should be applied to this machine pool.  For information please see
	// https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/.
	Taints []corev1.Taint `json:"taints,omitempty"`

	// +kubebuilder:validation:Optional
	// Represents the AWS provider specific configuration options.
	AWS MachinePoolProviderAWS `json:"aws,omitempty"`
}

// MachinePoolProviderAWS represents the provider specific configuration for an AWS provider.
type MachinePoolProviderAWS struct {
	// +kubebuilder:validation:Optional
	// Configuration of AWS Spot Instances for this MachinePool.
	SpotInstances MachinePoolProviderAWSSpotInstances `json:"spotInstances,omitempty"`
}

// MachinePoolProviderAWSSpotInstances represents the AWS Spot Intance configuration.
type MachinePoolProviderAWSSpotInstances struct {
	// +kubebuilder:validation:Optional
	// Request spot instances when scaling up this MachinePool.  If enabled a maximum
	// price for the spot instances may be set in spec.aws.spotInstances.maximumPrice.
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// Maximum price to pay for spot instance.
	// To be used with spec.aws.spotInstances.enabled. If no maximum price is set,
	// the spot instance configuration defaults to on-demand pricing.
	MaximumPrice int `json:"maximumPrice,omitempty"`
}

// MachinePoolStatus defines the observed state of MachinePool
type MachinePoolStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MachinePool is the Schema for the machinepools API
type MachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachinePoolSpec   `json:"spec,omitempty"`
	Status MachinePoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MachinePoolList contains a list of MachinePool
type MachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachinePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachinePool{}, &MachinePoolList{})
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
	// set the managed label
	machinePool.Spec.Labels[ocm.LabelPrefixManaged] = "true"

	// set the name label
	machinePool.Spec.Labels[ocm.LabelPrefixName] = machinePool.GetDisplayName()
}

// HasManagedLabels determines if the MachinePool object has the appropriate managed
// labels in its desired state.
func (machinePool *MachinePool) HasManagedLabels() bool {
	for _, label := range []string{
		ocm.LabelPrefixManaged,
		ocm.LabelPrefixName,
	} {
		if machinePool.Spec.Labels[label] == "" {
			return false
		}
	}

	return true
}

// CopyFrom copies an OCM MachinePool object into a MachinePool object that is recognizable by this
// controller.
func (machinePool *MachinePool) CopyFrom(source *clustersmgmtv1.MachinePool) error {
	machinePool.Spec.DisplayName = source.ID()
	machinePool.Spec.InstanceType = source.InstanceType()
	machinePool.Spec.Labels = source.Labels()
	machinePool.Spec.Taints = copyTaints(source.Taints())
	machinePool.Spec.MinimumNodesPerZone = copyMinimumNodesPerZone(source)
	machinePool.Spec.MaximumNodesPerZone = copyMaximumNodesPerZone(source)
	machinePool.Spec.AWS = copyAWSConfig(source.AWS())

	return nil
}

// Builder builds an OCM MachinePoolBuilder object.
func (machinePool *MachinePool) Builder() *clustersmgmtv1.MachinePoolBuilder {
	builder := clustersmgmtv1.NewMachinePool().
		ID(machinePool.Spec.DisplayName).
		InstanceType(machinePool.Spec.InstanceType).
		Labels(machinePool.Spec.Labels).
		Taints(machinePool.convertTaints()...).
		AWS(machinePool.convertAWSMachinePool())

	if machinePool.Spec.MaximumNodesPerZone > 0 {
		builder = builder.Autoscaling(machinePool.convertAutoscaling())
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

func (machinePool *MachinePool) convertAutoscaling() (builder *clustersmgmtv1.MachinePoolAutoscalingBuilder) {
	if machinePool.Spec.MaximumNodesPerZone > 0 {
		return clustersmgmtv1.NewMachinePoolAutoscaling().
			MinReplicas(machinePool.Spec.MinimumNodesPerZone).
			MaxReplicas(machinePool.Spec.MaximumNodesPerZone)
	}

	return clustersmgmtv1.NewMachinePoolAutoscaling()
}

func (machinePool *MachinePool) convertAWSMachinePool() (builder *clustersmgmtv1.AWSMachinePoolBuilder) {
	if machinePool.Spec.AWS.SpotInstances.Enabled {
		if machinePool.Spec.AWS.SpotInstances.MaximumPrice > 0 {
			return clustersmgmtv1.NewAWSMachinePool().
				SpotMarketOptions(
					clustersmgmtv1.NewAWSSpotMarketOptions().
						MaxPrice(float64(machinePool.Spec.AWS.SpotInstances.MaximumPrice)),
				)
		}
	}

	return builder
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

func copyMinimumNodesPerZone(source *clustersmgmtv1.MachinePool) int {
	if source.Autoscaling().MaxReplicas() > 0 {
		return source.Autoscaling().MinReplicas()
	}

	return source.Replicas()
}

func copyMaximumNodesPerZone(source *clustersmgmtv1.MachinePool) int {
	if source.Autoscaling().MaxReplicas() > 0 {
		return source.Autoscaling().MaxReplicas()
	}

	return 0
}

func copyAWSConfig(source *clustersmgmtv1.AWSMachinePool) MachinePoolProviderAWS {
	if source.SpotMarketOptions().Empty() {
		return MachinePoolProviderAWS{}
	}

	return MachinePoolProviderAWS{
		SpotInstances: MachinePoolProviderAWSSpotInstances{
			Enabled:      true,
			MaximumPrice: int(source.SpotMarketOptions().MaxPrice()),
		},
	}
}
