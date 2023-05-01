//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitLabIdentityProvider) DeepCopyInto(out *GitLabIdentityProvider) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitLabIdentityProvider.
func (in *GitLabIdentityProvider) DeepCopy() *GitLabIdentityProvider {
	if in == nil {
		return nil
	}
	out := new(GitLabIdentityProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GitLabIdentityProvider) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitLabIdentityProviderList) DeepCopyInto(out *GitLabIdentityProviderList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GitLabIdentityProvider, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitLabIdentityProviderList.
func (in *GitLabIdentityProviderList) DeepCopy() *GitLabIdentityProviderList {
	if in == nil {
		return nil
	}
	out := new(GitLabIdentityProviderList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GitLabIdentityProviderList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitLabIdentityProviderSpec) DeepCopyInto(out *GitLabIdentityProviderSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitLabIdentityProviderSpec.
func (in *GitLabIdentityProviderSpec) DeepCopy() *GitLabIdentityProviderSpec {
	if in == nil {
		return nil
	}
	out := new(GitLabIdentityProviderSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitLabIdentityProviderStatus) DeepCopyInto(out *GitLabIdentityProviderStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitLabIdentityProviderStatus.
func (in *GitLabIdentityProviderStatus) DeepCopy() *GitLabIdentityProviderStatus {
	if in == nil {
		return nil
	}
	out := new(GitLabIdentityProviderStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePool) DeepCopyInto(out *MachinePool) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePool.
func (in *MachinePool) DeepCopy() *MachinePool {
	if in == nil {
		return nil
	}
	out := new(MachinePool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachinePool) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePoolList) DeepCopyInto(out *MachinePoolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MachinePool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePoolList.
func (in *MachinePoolList) DeepCopy() *MachinePoolList {
	if in == nil {
		return nil
	}
	out := new(MachinePoolList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachinePoolList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePoolProviderAWS) DeepCopyInto(out *MachinePoolProviderAWS) {
	*out = *in
	out.SpotInstances = in.SpotInstances
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePoolProviderAWS.
func (in *MachinePoolProviderAWS) DeepCopy() *MachinePoolProviderAWS {
	if in == nil {
		return nil
	}
	out := new(MachinePoolProviderAWS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePoolProviderAWSSpotInstances) DeepCopyInto(out *MachinePoolProviderAWSSpotInstances) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePoolProviderAWSSpotInstances.
func (in *MachinePoolProviderAWSSpotInstances) DeepCopy() *MachinePoolProviderAWSSpotInstances {
	if in == nil {
		return nil
	}
	out := new(MachinePoolProviderAWSSpotInstances)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePoolSpec) DeepCopyInto(out *MachinePoolSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Taints != nil {
		in, out := &in.Taints, &out.Taints
		*out = make([]corev1.Taint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.AWS = in.AWS
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePoolSpec.
func (in *MachinePoolSpec) DeepCopy() *MachinePoolSpec {
	if in == nil {
		return nil
	}
	out := new(MachinePoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePoolStatus) DeepCopyInto(out *MachinePoolStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AvailabilityZones != nil {
		in, out := &in.AvailabilityZones, &out.AvailabilityZones
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Subnets != nil {
		in, out := &in.Subnets, &out.Subnets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePoolStatus.
func (in *MachinePoolStatus) DeepCopy() *MachinePoolStatus {
	if in == nil {
		return nil
	}
	out := new(MachinePoolStatus)
	in.DeepCopyInto(out)
	return out
}
