//go:build !ignore_autogenerated

/*
Copyright 2024.

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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Driver) DeepCopyInto(out *Driver) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Driver.
func (in *Driver) DeepCopy() *Driver {
	if in == nil {
		return nil
	}
	out := new(Driver)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Driver) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DriverList) DeepCopyInto(out *DriverList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Driver, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DriverList.
func (in *DriverList) DeepCopy() *DriverList {
	if in == nil {
		return nil
	}
	out := new(DriverList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DriverList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DriverSpec) DeepCopyInto(out *DriverSpec) {
	*out = *in
	if in.Logging != nil {
		in, out := &in.Logging, &out.Logging
		*out = new(LogSpec)
		**out = **in
	}
	if in.ImageSet != nil {
		in, out := &in.ImageSet, &out.ImageSet
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.ClusterName != nil {
		in, out := &in.ClusterName, &out.ClusterName
		*out = new(string)
		**out = **in
	}
	if in.EnableMetadata != nil {
		in, out := &in.EnableMetadata, &out.EnableMetadata
		*out = new(bool)
		**out = **in
	}
	if in.GenerateOMapInfo != nil {
		in, out := &in.GenerateOMapInfo, &out.GenerateOMapInfo
		*out = new(bool)
		**out = **in
	}
	if in.Encryption != nil {
		in, out := &in.Encryption, &out.Encryption
		*out = new(EncryptionSpec)
		**out = **in
	}
	if in.Plugin != nil {
		in, out := &in.Plugin, &out.Plugin
		*out = new(PluginSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Provisioner != nil {
		in, out := &in.Provisioner, &out.Provisioner
		*out = new(ProvisionerSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.AttachRequired != nil {
		in, out := &in.AttachRequired, &out.AttachRequired
		*out = new(bool)
		**out = **in
	}
	if in.Liveness != nil {
		in, out := &in.Liveness, &out.Liveness
		*out = new(LivenessSpec)
		**out = **in
	}
	if in.LeaderElection != nil {
		in, out := &in.LeaderElection, &out.LeaderElection
		*out = new(LeaderElectionSpec)
		**out = **in
	}
	if in.DeployCsiAddons != nil {
		in, out := &in.DeployCsiAddons, &out.DeployCsiAddons
		*out = new(bool)
		**out = **in
	}
	if in.KernelMountOptions != nil {
		in, out := &in.KernelMountOptions, &out.KernelMountOptions
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.FuseMountOptionss != nil {
		in, out := &in.FuseMountOptionss, &out.FuseMountOptionss
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DriverSpec.
func (in *DriverSpec) DeepCopy() *DriverSpec {
	if in == nil {
		return nil
	}
	out := new(DriverSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DriverStatus) DeepCopyInto(out *DriverStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DriverStatus.
func (in *DriverStatus) DeepCopy() *DriverStatus {
	if in == nil {
		return nil
	}
	out := new(DriverStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EncryptionSpec) DeepCopyInto(out *EncryptionSpec) {
	*out = *in
	out.ConfigMapRef = in.ConfigMapRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EncryptionSpec.
func (in *EncryptionSpec) DeepCopy() *EncryptionSpec {
	if in == nil {
		return nil
	}
	out := new(EncryptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LeaderElectionSpec) DeepCopyInto(out *LeaderElectionSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LeaderElectionSpec.
func (in *LeaderElectionSpec) DeepCopy() *LeaderElectionSpec {
	if in == nil {
		return nil
	}
	out := new(LeaderElectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LivenessSpec) DeepCopyInto(out *LivenessSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LivenessSpec.
func (in *LivenessSpec) DeepCopy() *LivenessSpec {
	if in == nil {
		return nil
	}
	out := new(LivenessSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogSpec) DeepCopyInto(out *LogSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogSpec.
func (in *LogSpec) DeepCopy() *LogSpec {
	if in == nil {
		return nil
	}
	out := new(LogSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorConfig) DeepCopyInto(out *OperatorConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorConfig.
func (in *OperatorConfig) DeepCopy() *OperatorConfig {
	if in == nil {
		return nil
	}
	out := new(OperatorConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatorConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorConfigList) DeepCopyInto(out *OperatorConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OperatorConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorConfigList.
func (in *OperatorConfigList) DeepCopy() *OperatorConfigList {
	if in == nil {
		return nil
	}
	out := new(OperatorConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatorConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorConfigSpec) DeepCopyInto(out *OperatorConfigSpec) {
	*out = *in
	if in.DriverSpecDefaults != nil {
		in, out := &in.DriverSpecDefaults, &out.DriverSpecDefaults
		*out = new(DriverSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorConfigSpec.
func (in *OperatorConfigSpec) DeepCopy() *OperatorConfigSpec {
	if in == nil {
		return nil
	}
	out := new(OperatorConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorConfigStatus) DeepCopyInto(out *OperatorConfigStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorConfigStatus.
func (in *OperatorConfigStatus) DeepCopy() *OperatorConfigStatus {
	if in == nil {
		return nil
	}
	out := new(OperatorConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PluginResourcesSpec) DeepCopyInto(out *PluginResourcesSpec) {
	*out = *in
	if in.Registrar != nil {
		in, out := &in.Registrar, &out.Registrar
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Liveness != nil {
		in, out := &in.Liveness, &out.Liveness
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Plugin != nil {
		in, out := &in.Plugin, &out.Plugin
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PluginResourcesSpec.
func (in *PluginResourcesSpec) DeepCopy() *PluginResourcesSpec {
	if in == nil {
		return nil
	}
	out := new(PluginResourcesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PluginSpec) DeepCopyInto(out *PluginSpec) {
	*out = *in
	in.PodCommonSpec.DeepCopyInto(&out.PodCommonSpec)
	if in.UpdateStrategy != nil {
		in, out := &in.UpdateStrategy, &out.UpdateStrategy
		*out = new(appsv1.DaemonSetUpdateStrategy)
		(*in).DeepCopyInto(*out)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnableSeLinuxHostMount != nil {
		in, out := &in.EnableSeLinuxHostMount, &out.EnableSeLinuxHostMount
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PluginSpec.
func (in *PluginSpec) DeepCopy() *PluginSpec {
	if in == nil {
		return nil
	}
	out := new(PluginSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodCommonSpec) DeepCopyInto(out *PodCommonSpec) {
	*out = *in
	if in.PrioritylClassName != nil {
		in, out := &in.PrioritylClassName, &out.PrioritylClassName
		*out = new(string)
		**out = **in
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodCommonSpec.
func (in *PodCommonSpec) DeepCopy() *PodCommonSpec {
	if in == nil {
		return nil
	}
	out := new(PodCommonSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvisionerResourcesSpec) DeepCopyInto(out *ProvisionerResourcesSpec) {
	*out = *in
	if in.Attacher != nil {
		in, out := &in.Attacher, &out.Attacher
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Snapshotter != nil {
		in, out := &in.Snapshotter, &out.Snapshotter
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Resizer != nil {
		in, out := &in.Resizer, &out.Resizer
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Provisioner != nil {
		in, out := &in.Provisioner, &out.Provisioner
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.OMapGenerator != nil {
		in, out := &in.OMapGenerator, &out.OMapGenerator
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Liveness != nil {
		in, out := &in.Liveness, &out.Liveness
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Plugin != nil {
		in, out := &in.Plugin, &out.Plugin
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvisionerResourcesSpec.
func (in *ProvisionerResourcesSpec) DeepCopy() *ProvisionerResourcesSpec {
	if in == nil {
		return nil
	}
	out := new(ProvisionerResourcesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvisionerSpec) DeepCopyInto(out *ProvisionerSpec) {
	*out = *in
	in.PodCommonSpec.DeepCopyInto(&out.PodCommonSpec)
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvisionerSpec.
func (in *ProvisionerSpec) DeepCopy() *ProvisionerSpec {
	if in == nil {
		return nil
	}
	out := new(ProvisionerSpec)
	in.DeepCopyInto(out)
	return out
}
