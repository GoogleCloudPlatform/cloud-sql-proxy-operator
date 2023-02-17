//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthProxyContainerSpec) DeepCopyInto(out *AuthProxyContainerSpec) {
	*out = *in
	if in.Container != nil {
		in, out := &in.Container, &out.Container
		*out = new(corev1.Container)
		(*in).DeepCopyInto(*out)
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(corev1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Telemetry != nil {
		in, out := &in.Telemetry, &out.Telemetry
		*out = new(TelemetrySpec)
		(*in).DeepCopyInto(*out)
	}
	if in.MaxConnections != nil {
		in, out := &in.MaxConnections, &out.MaxConnections
		*out = new(int64)
		**out = **in
	}
	if in.MaxSigtermDelay != nil {
		in, out := &in.MaxSigtermDelay, &out.MaxSigtermDelay
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthProxyContainerSpec.
func (in *AuthProxyContainerSpec) DeepCopy() *AuthProxyContainerSpec {
	if in == nil {
		return nil
	}
	out := new(AuthProxyContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthProxyWorkload) DeepCopyInto(out *AuthProxyWorkload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthProxyWorkload.
func (in *AuthProxyWorkload) DeepCopy() *AuthProxyWorkload {
	if in == nil {
		return nil
	}
	out := new(AuthProxyWorkload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AuthProxyWorkload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthProxyWorkloadList) DeepCopyInto(out *AuthProxyWorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AuthProxyWorkload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthProxyWorkloadList.
func (in *AuthProxyWorkloadList) DeepCopy() *AuthProxyWorkloadList {
	if in == nil {
		return nil
	}
	out := new(AuthProxyWorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AuthProxyWorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthProxyWorkloadSpec) DeepCopyInto(out *AuthProxyWorkloadSpec) {
	*out = *in
	in.Workload.DeepCopyInto(&out.Workload)
	if in.Instances != nil {
		in, out := &in.Instances, &out.Instances
		*out = make([]InstanceSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AuthProxyContainer != nil {
		in, out := &in.AuthProxyContainer, &out.AuthProxyContainer
		*out = new(AuthProxyContainerSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthProxyWorkloadSpec.
func (in *AuthProxyWorkloadSpec) DeepCopy() *AuthProxyWorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(AuthProxyWorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthProxyWorkloadStatus) DeepCopyInto(out *AuthProxyWorkloadStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]*v1.Condition, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(v1.Condition)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.WorkloadStatus != nil {
		in, out := &in.WorkloadStatus, &out.WorkloadStatus
		*out = make([]*WorkloadStatus, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(WorkloadStatus)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthProxyWorkloadStatus.
func (in *AuthProxyWorkloadStatus) DeepCopy() *AuthProxyWorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(AuthProxyWorkloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceSpec) DeepCopyInto(out *InstanceSpec) {
	*out = *in
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int32)
		**out = **in
	}
	if in.AutoIAMAuthN != nil {
		in, out := &in.AutoIAMAuthN, &out.AutoIAMAuthN
		*out = new(bool)
		**out = **in
	}
	if in.PrivateIP != nil {
		in, out := &in.PrivateIP, &out.PrivateIP
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceSpec.
func (in *InstanceSpec) DeepCopy() *InstanceSpec {
	if in == nil {
		return nil
	}
	out := new(InstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetrySpec) DeepCopyInto(out *TelemetrySpec) {
	*out = *in
	if in.HTTPPort != nil {
		in, out := &in.HTTPPort, &out.HTTPPort
		*out = new(int32)
		**out = **in
	}
	if in.AdminPort != nil {
		in, out := &in.AdminPort, &out.AdminPort
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetrySpec.
func (in *TelemetrySpec) DeepCopy() *TelemetrySpec {
	if in == nil {
		return nil
	}
	out := new(TelemetrySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadSelectorSpec) DeepCopyInto(out *WorkloadSelectorSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadSelectorSpec.
func (in *WorkloadSelectorSpec) DeepCopy() *WorkloadSelectorSpec {
	if in == nil {
		return nil
	}
	out := new(WorkloadSelectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadStatus) DeepCopyInto(out *WorkloadStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]*v1.Condition, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(v1.Condition)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadStatus.
func (in *WorkloadStatus) DeepCopy() *WorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(WorkloadStatus)
	in.DeepCopyInto(out)
	return out
}
