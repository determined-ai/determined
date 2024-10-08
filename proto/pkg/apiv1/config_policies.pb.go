// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// source: determined/api/v1/config_policies.proto

package apiv1

import (
	_struct "github.com/golang/protobuf/ptypes/struct"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// PutWorkspaceConfigPoliciesRequest sets config
// policies for the workspace and workload type.
type PutWorkspaceConfigPoliciesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The workspace the config policies apply to. Use global API for
	// global config policies.
	WorkspaceId int32 `protobuf:"varint,1,opt,name=workspace_id,json=workspaceId,proto3" json:"workspace_id,omitempty"`
	// The workload type the config policies apply to: EXPERIMENT or NTSC.
	WorkloadType string `protobuf:"bytes,2,opt,name=workload_type,json=workloadType,proto3" json:"workload_type,omitempty"`
	// The config policies to use. Contains both invariant configs and constraints
	// in yaml or json format.
	ConfigPolicies string `protobuf:"bytes,3,opt,name=config_policies,json=configPolicies,proto3" json:"config_policies,omitempty"`
}

func (x *PutWorkspaceConfigPoliciesRequest) Reset() {
	*x = PutWorkspaceConfigPoliciesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PutWorkspaceConfigPoliciesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PutWorkspaceConfigPoliciesRequest) ProtoMessage() {}

func (x *PutWorkspaceConfigPoliciesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PutWorkspaceConfigPoliciesRequest.ProtoReflect.Descriptor instead.
func (*PutWorkspaceConfigPoliciesRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{0}
}

func (x *PutWorkspaceConfigPoliciesRequest) GetWorkspaceId() int32 {
	if x != nil {
		return x.WorkspaceId
	}
	return 0
}

func (x *PutWorkspaceConfigPoliciesRequest) GetWorkloadType() string {
	if x != nil {
		return x.WorkloadType
	}
	return ""
}

func (x *PutWorkspaceConfigPoliciesRequest) GetConfigPolicies() string {
	if x != nil {
		return x.ConfigPolicies
	}
	return ""
}

// Response to PutWorkspaceConfigPoliciesRequest.
type PutWorkspaceConfigPoliciesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The config policies saved. Contains both invariant configs and constraints
	// in yaml or json format.
	ConfigPolicies *_struct.Struct `protobuf:"bytes,1,opt,name=config_policies,json=configPolicies,proto3" json:"config_policies,omitempty"`
}

func (x *PutWorkspaceConfigPoliciesResponse) Reset() {
	*x = PutWorkspaceConfigPoliciesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PutWorkspaceConfigPoliciesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PutWorkspaceConfigPoliciesResponse) ProtoMessage() {}

func (x *PutWorkspaceConfigPoliciesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PutWorkspaceConfigPoliciesResponse.ProtoReflect.Descriptor instead.
func (*PutWorkspaceConfigPoliciesResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{1}
}

func (x *PutWorkspaceConfigPoliciesResponse) GetConfigPolicies() *_struct.Struct {
	if x != nil {
		return x.ConfigPolicies
	}
	return nil
}

// PutGlobalConfigPoliciesRequest sets global config
// policies for the workload type.
type PutGlobalConfigPoliciesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The workload type the config policies apply to: EXPERIMENT or NTSC.
	WorkloadType string `protobuf:"bytes,1,opt,name=workload_type,json=workloadType,proto3" json:"workload_type,omitempty"`
	// The config policies to use. Contains both invariant configs and constraints
	// in yaml or json format.
	ConfigPolicies string `protobuf:"bytes,2,opt,name=config_policies,json=configPolicies,proto3" json:"config_policies,omitempty"`
}

func (x *PutGlobalConfigPoliciesRequest) Reset() {
	*x = PutGlobalConfigPoliciesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PutGlobalConfigPoliciesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PutGlobalConfigPoliciesRequest) ProtoMessage() {}

func (x *PutGlobalConfigPoliciesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PutGlobalConfigPoliciesRequest.ProtoReflect.Descriptor instead.
func (*PutGlobalConfigPoliciesRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{2}
}

func (x *PutGlobalConfigPoliciesRequest) GetWorkloadType() string {
	if x != nil {
		return x.WorkloadType
	}
	return ""
}

func (x *PutGlobalConfigPoliciesRequest) GetConfigPolicies() string {
	if x != nil {
		return x.ConfigPolicies
	}
	return ""
}

// Response to PutGlobalConfigPoliciesRequest.
type PutGlobalConfigPoliciesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The config policies saved. Contains both invariant configs and constraints
	// in yaml or json format.
	ConfigPolicies *_struct.Struct `protobuf:"bytes,1,opt,name=config_policies,json=configPolicies,proto3" json:"config_policies,omitempty"`
}

func (x *PutGlobalConfigPoliciesResponse) Reset() {
	*x = PutGlobalConfigPoliciesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PutGlobalConfigPoliciesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PutGlobalConfigPoliciesResponse) ProtoMessage() {}

func (x *PutGlobalConfigPoliciesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PutGlobalConfigPoliciesResponse.ProtoReflect.Descriptor instead.
func (*PutGlobalConfigPoliciesResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{3}
}

func (x *PutGlobalConfigPoliciesResponse) GetConfigPolicies() *_struct.Struct {
	if x != nil {
		return x.ConfigPolicies
	}
	return nil
}

// GetWorkspaceConfigPoliciesRequest lists task config policies
// for a given workspace and workload type.
type GetWorkspaceConfigPoliciesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The workspace the config policies apply to. Use global API for
	// global config policies.
	WorkspaceId int32 `protobuf:"varint,1,opt,name=workspace_id,json=workspaceId,proto3" json:"workspace_id,omitempty"`
	// The workload type the config policies apply to: EXPERIMENT or NTSC.
	WorkloadType string `protobuf:"bytes,2,opt,name=workload_type,json=workloadType,proto3" json:"workload_type,omitempty"`
}

func (x *GetWorkspaceConfigPoliciesRequest) Reset() {
	*x = GetWorkspaceConfigPoliciesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetWorkspaceConfigPoliciesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetWorkspaceConfigPoliciesRequest) ProtoMessage() {}

func (x *GetWorkspaceConfigPoliciesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetWorkspaceConfigPoliciesRequest.ProtoReflect.Descriptor instead.
func (*GetWorkspaceConfigPoliciesRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{4}
}

func (x *GetWorkspaceConfigPoliciesRequest) GetWorkspaceId() int32 {
	if x != nil {
		return x.WorkspaceId
	}
	return 0
}

func (x *GetWorkspaceConfigPoliciesRequest) GetWorkloadType() string {
	if x != nil {
		return x.WorkloadType
	}
	return ""
}

// Response to GetWorkspaceConfigPoliciesRequest.
type GetWorkspaceConfigPoliciesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The current config policies saved for the workspace. Contains both
	// invariant configs and constraints in yaml or json format.
	ConfigPolicies *_struct.Struct `protobuf:"bytes,1,opt,name=config_policies,json=configPolicies,proto3" json:"config_policies,omitempty"`
}

func (x *GetWorkspaceConfigPoliciesResponse) Reset() {
	*x = GetWorkspaceConfigPoliciesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetWorkspaceConfigPoliciesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetWorkspaceConfigPoliciesResponse) ProtoMessage() {}

func (x *GetWorkspaceConfigPoliciesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetWorkspaceConfigPoliciesResponse.ProtoReflect.Descriptor instead.
func (*GetWorkspaceConfigPoliciesResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{5}
}

func (x *GetWorkspaceConfigPoliciesResponse) GetConfigPolicies() *_struct.Struct {
	if x != nil {
		return x.ConfigPolicies
	}
	return nil
}

// GetGlobalConfigPoliciesRequest lists global task config
// policies for a given workload type.
type GetGlobalConfigPoliciesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The workload type the config policies apply to: EXPERIMENT or NTSC.
	WorkloadType string `protobuf:"bytes,1,opt,name=workload_type,json=workloadType,proto3" json:"workload_type,omitempty"`
}

func (x *GetGlobalConfigPoliciesRequest) Reset() {
	*x = GetGlobalConfigPoliciesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGlobalConfigPoliciesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGlobalConfigPoliciesRequest) ProtoMessage() {}

func (x *GetGlobalConfigPoliciesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGlobalConfigPoliciesRequest.ProtoReflect.Descriptor instead.
func (*GetGlobalConfigPoliciesRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{6}
}

func (x *GetGlobalConfigPoliciesRequest) GetWorkloadType() string {
	if x != nil {
		return x.WorkloadType
	}
	return ""
}

// Response to GetGlobalConfigPoliciesRequest.
type GetGlobalConfigPoliciesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The global current config policies saved. Contains both invariant configs
	// and constraints in yaml or json format.
	ConfigPolicies *_struct.Struct `protobuf:"bytes,1,opt,name=config_policies,json=configPolicies,proto3" json:"config_policies,omitempty"`
}

func (x *GetGlobalConfigPoliciesResponse) Reset() {
	*x = GetGlobalConfigPoliciesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGlobalConfigPoliciesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGlobalConfigPoliciesResponse) ProtoMessage() {}

func (x *GetGlobalConfigPoliciesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGlobalConfigPoliciesResponse.ProtoReflect.Descriptor instead.
func (*GetGlobalConfigPoliciesResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{7}
}

func (x *GetGlobalConfigPoliciesResponse) GetConfigPolicies() *_struct.Struct {
	if x != nil {
		return x.ConfigPolicies
	}
	return nil
}

// DeleteWorkspaceConfigPoliciesRequest is used to delete all task config
// policies for the workspace and workload type.
type DeleteWorkspaceConfigPoliciesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The workspace the config policies apply to. Use global API for
	// global config policies.
	WorkspaceId int32 `protobuf:"varint,1,opt,name=workspace_id,json=workspaceId,proto3" json:"workspace_id,omitempty"`
	// The workload type the config policies apply to: EXPERIMENT or NTSC.
	WorkloadType string `protobuf:"bytes,2,opt,name=workload_type,json=workloadType,proto3" json:"workload_type,omitempty"`
}

func (x *DeleteWorkspaceConfigPoliciesRequest) Reset() {
	*x = DeleteWorkspaceConfigPoliciesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteWorkspaceConfigPoliciesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteWorkspaceConfigPoliciesRequest) ProtoMessage() {}

func (x *DeleteWorkspaceConfigPoliciesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteWorkspaceConfigPoliciesRequest.ProtoReflect.Descriptor instead.
func (*DeleteWorkspaceConfigPoliciesRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{8}
}

func (x *DeleteWorkspaceConfigPoliciesRequest) GetWorkspaceId() int32 {
	if x != nil {
		return x.WorkspaceId
	}
	return 0
}

func (x *DeleteWorkspaceConfigPoliciesRequest) GetWorkloadType() string {
	if x != nil {
		return x.WorkloadType
	}
	return ""
}

// Response to DeleteWorkspaceConfigPoliciesRequest.
type DeleteWorkspaceConfigPoliciesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeleteWorkspaceConfigPoliciesResponse) Reset() {
	*x = DeleteWorkspaceConfigPoliciesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteWorkspaceConfigPoliciesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteWorkspaceConfigPoliciesResponse) ProtoMessage() {}

func (x *DeleteWorkspaceConfigPoliciesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteWorkspaceConfigPoliciesResponse.ProtoReflect.Descriptor instead.
func (*DeleteWorkspaceConfigPoliciesResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{9}
}

// DeleteGlobalConfigPoliciesRequest is used to delete all global task config
// policies for the workload type.
type DeleteGlobalConfigPoliciesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The workload type the config policies apply to: EXPERIMENT or NTSC.
	WorkloadType string `protobuf:"bytes,1,opt,name=workload_type,json=workloadType,proto3" json:"workload_type,omitempty"`
}

func (x *DeleteGlobalConfigPoliciesRequest) Reset() {
	*x = DeleteGlobalConfigPoliciesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteGlobalConfigPoliciesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteGlobalConfigPoliciesRequest) ProtoMessage() {}

func (x *DeleteGlobalConfigPoliciesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteGlobalConfigPoliciesRequest.ProtoReflect.Descriptor instead.
func (*DeleteGlobalConfigPoliciesRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{10}
}

func (x *DeleteGlobalConfigPoliciesRequest) GetWorkloadType() string {
	if x != nil {
		return x.WorkloadType
	}
	return ""
}

// Response to DeleteGlobalConfigPoliciesRequest.
type DeleteGlobalConfigPoliciesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeleteGlobalConfigPoliciesResponse) Reset() {
	*x = DeleteGlobalConfigPoliciesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_config_policies_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteGlobalConfigPoliciesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteGlobalConfigPoliciesResponse) ProtoMessage() {}

func (x *DeleteGlobalConfigPoliciesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_config_policies_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteGlobalConfigPoliciesResponse.ProtoReflect.Descriptor instead.
func (*DeleteGlobalConfigPoliciesResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_config_policies_proto_rawDescGZIP(), []int{11}
}

var File_determined_api_v1_config_policies_proto protoreflect.FileDescriptor

var file_determined_api_v1_config_policies_proto_rawDesc = []byte{
	0x0a, 0x27, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63,
	0x69, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11, 0x64, 0x65, 0x74, 0x65, 0x72,
	0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x1a, 0x1c, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x73, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2c, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x73, 0x77, 0x61, 0x67, 0x67, 0x65, 0x72, 0x2f, 0x6f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xcc, 0x01, 0x0a, 0x21, 0x50, 0x75, 0x74,
	0x57, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50,
	0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x21,
	0x0a, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x0b, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x49,
	0x64, 0x12, 0x23, 0x0a, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79,
	0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f,
	0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x3a,
	0x36, 0x92, 0x41, 0x33, 0x0a, 0x31, 0xd2, 0x01, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x5f, 0x69, 0x64, 0xd2, 0x01, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64,
	0x5f, 0x74, 0x79, 0x70, 0x65, 0xd2, 0x01, 0x0f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70,
	0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x66, 0x0a, 0x22, 0x50, 0x75, 0x74, 0x57, 0x6f,
	0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c,
	0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x40, 0x0a,
	0x0f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52,
	0x0e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22,
	0x97, 0x01, 0x0a, 0x1e, 0x50, 0x75, 0x74, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74,
	0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x6c,
	0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73,
	0x3a, 0x27, 0x92, 0x41, 0x24, 0x0a, 0x22, 0xd2, 0x01, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f,
	0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0xd2, 0x01, 0x0f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x63, 0x0a, 0x1f, 0x50, 0x75, 0x74,
	0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69,
	0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x40, 0x0a, 0x0f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x0e,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x91,
	0x01, 0x0a, 0x21, 0x47, 0x65, 0x74, 0x57, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0b, 0x77, 0x6f, 0x72, 0x6b,
	0x73, 0x70, 0x61, 0x63, 0x65, 0x49, 0x64, 0x12, 0x23, 0x0a, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c,
	0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c,
	0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x3a, 0x24, 0x92, 0x41,
	0x21, 0x0a, 0x1f, 0xd2, 0x01, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x5f,
	0x69, 0x64, 0xd2, 0x01, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79,
	0x70, 0x65, 0x22, 0x66, 0x0a, 0x22, 0x47, 0x65, 0x74, 0x57, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x40, 0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x0e, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x5c, 0x0a, 0x1e, 0x47, 0x65,
	0x74, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c,
	0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d,
	0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x54, 0x79, 0x70,
	0x65, 0x3a, 0x15, 0x92, 0x41, 0x12, 0x0a, 0x10, 0xd2, 0x01, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c,
	0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x22, 0x63, 0x0a, 0x1f, 0x47, 0x65, 0x74, 0x47,
	0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63,
	0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x40, 0x0a, 0x0f, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x0e, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x94, 0x01,
	0x0a, 0x24, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x57, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0b, 0x77, 0x6f,
	0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x49, 0x64, 0x12, 0x23, 0x0a, 0x0d, 0x77, 0x6f, 0x72,
	0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x3a, 0x24,
	0x92, 0x41, 0x21, 0x0a, 0x1f, 0xd2, 0x01, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x5f, 0x69, 0x64, 0xd2, 0x01, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f,
	0x74, 0x79, 0x70, 0x65, 0x22, 0x27, 0x0a, 0x25, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x57, 0x6f,
	0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c,
	0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x5f, 0x0a,
	0x21, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74,
	0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x6c,
	0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x3a, 0x15, 0x92, 0x41, 0x12, 0x0a, 0x10, 0xd2, 0x01,
	0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x22, 0x24,
	0x0a, 0x22, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2d, 0x61, 0x69,
	0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_determined_api_v1_config_policies_proto_rawDescOnce sync.Once
	file_determined_api_v1_config_policies_proto_rawDescData = file_determined_api_v1_config_policies_proto_rawDesc
)

func file_determined_api_v1_config_policies_proto_rawDescGZIP() []byte {
	file_determined_api_v1_config_policies_proto_rawDescOnce.Do(func() {
		file_determined_api_v1_config_policies_proto_rawDescData = protoimpl.X.CompressGZIP(file_determined_api_v1_config_policies_proto_rawDescData)
	})
	return file_determined_api_v1_config_policies_proto_rawDescData
}

var file_determined_api_v1_config_policies_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_determined_api_v1_config_policies_proto_goTypes = []interface{}{
	(*PutWorkspaceConfigPoliciesRequest)(nil),     // 0: determined.api.v1.PutWorkspaceConfigPoliciesRequest
	(*PutWorkspaceConfigPoliciesResponse)(nil),    // 1: determined.api.v1.PutWorkspaceConfigPoliciesResponse
	(*PutGlobalConfigPoliciesRequest)(nil),        // 2: determined.api.v1.PutGlobalConfigPoliciesRequest
	(*PutGlobalConfigPoliciesResponse)(nil),       // 3: determined.api.v1.PutGlobalConfigPoliciesResponse
	(*GetWorkspaceConfigPoliciesRequest)(nil),     // 4: determined.api.v1.GetWorkspaceConfigPoliciesRequest
	(*GetWorkspaceConfigPoliciesResponse)(nil),    // 5: determined.api.v1.GetWorkspaceConfigPoliciesResponse
	(*GetGlobalConfigPoliciesRequest)(nil),        // 6: determined.api.v1.GetGlobalConfigPoliciesRequest
	(*GetGlobalConfigPoliciesResponse)(nil),       // 7: determined.api.v1.GetGlobalConfigPoliciesResponse
	(*DeleteWorkspaceConfigPoliciesRequest)(nil),  // 8: determined.api.v1.DeleteWorkspaceConfigPoliciesRequest
	(*DeleteWorkspaceConfigPoliciesResponse)(nil), // 9: determined.api.v1.DeleteWorkspaceConfigPoliciesResponse
	(*DeleteGlobalConfigPoliciesRequest)(nil),     // 10: determined.api.v1.DeleteGlobalConfigPoliciesRequest
	(*DeleteGlobalConfigPoliciesResponse)(nil),    // 11: determined.api.v1.DeleteGlobalConfigPoliciesResponse
	(*_struct.Struct)(nil),                        // 12: google.protobuf.Struct
}
var file_determined_api_v1_config_policies_proto_depIdxs = []int32{
	12, // 0: determined.api.v1.PutWorkspaceConfigPoliciesResponse.config_policies:type_name -> google.protobuf.Struct
	12, // 1: determined.api.v1.PutGlobalConfigPoliciesResponse.config_policies:type_name -> google.protobuf.Struct
	12, // 2: determined.api.v1.GetWorkspaceConfigPoliciesResponse.config_policies:type_name -> google.protobuf.Struct
	12, // 3: determined.api.v1.GetGlobalConfigPoliciesResponse.config_policies:type_name -> google.protobuf.Struct
	4,  // [4:4] is the sub-list for method output_type
	4,  // [4:4] is the sub-list for method input_type
	4,  // [4:4] is the sub-list for extension type_name
	4,  // [4:4] is the sub-list for extension extendee
	0,  // [0:4] is the sub-list for field type_name
}

func init() { file_determined_api_v1_config_policies_proto_init() }
func file_determined_api_v1_config_policies_proto_init() {
	if File_determined_api_v1_config_policies_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_determined_api_v1_config_policies_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PutWorkspaceConfigPoliciesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PutWorkspaceConfigPoliciesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PutGlobalConfigPoliciesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PutGlobalConfigPoliciesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetWorkspaceConfigPoliciesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetWorkspaceConfigPoliciesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGlobalConfigPoliciesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGlobalConfigPoliciesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteWorkspaceConfigPoliciesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteWorkspaceConfigPoliciesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteGlobalConfigPoliciesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_determined_api_v1_config_policies_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteGlobalConfigPoliciesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_determined_api_v1_config_policies_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_determined_api_v1_config_policies_proto_goTypes,
		DependencyIndexes: file_determined_api_v1_config_policies_proto_depIdxs,
		MessageInfos:      file_determined_api_v1_config_policies_proto_msgTypes,
	}.Build()
	File_determined_api_v1_config_policies_proto = out.File
	file_determined_api_v1_config_policies_proto_rawDesc = nil
	file_determined_api_v1_config_policies_proto_goTypes = nil
	file_determined_api_v1_config_policies_proto_depIdxs = nil
}
