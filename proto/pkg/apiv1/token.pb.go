// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// source: determined/api/v1/token.proto

package apiv1

import (
	userv1 "github.com/determined-ai/determined/proto/pkg/userv1"
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

// Sort token info by the given field.
type GetAccessTokensRequest_SortBy int32

const (
	// Returns token info in an unsorted list.
	GetAccessTokensRequest_SORT_BY_UNSPECIFIED GetAccessTokensRequest_SortBy = 0
	// Returns token info sorted by user id.
	GetAccessTokensRequest_SORT_BY_USER_ID GetAccessTokensRequest_SortBy = 1
	// Returns token info sorted by expiry.
	GetAccessTokensRequest_SORT_BY_EXPIRY GetAccessTokensRequest_SortBy = 2
	// Returns token info sorted by created at.
	GetAccessTokensRequest_SORT_BY_CREATED_AT GetAccessTokensRequest_SortBy = 3
	// Returns token info sorted by token type.
	GetAccessTokensRequest_SORT_BY_TOKEN_TYPE GetAccessTokensRequest_SortBy = 4
	// Returns token info sorted by if it is revoked.
	GetAccessTokensRequest_SORT_BY_REVOKED GetAccessTokensRequest_SortBy = 5
	// Returns token info sorted by description of token.
	GetAccessTokensRequest_SORT_BY_DESCRIPTION GetAccessTokensRequest_SortBy = 6
)

// Enum value maps for GetAccessTokensRequest_SortBy.
var (
	GetAccessTokensRequest_SortBy_name = map[int32]string{
		0: "SORT_BY_UNSPECIFIED",
		1: "SORT_BY_USER_ID",
		2: "SORT_BY_EXPIRY",
		3: "SORT_BY_CREATED_AT",
		4: "SORT_BY_TOKEN_TYPE",
		5: "SORT_BY_REVOKED",
		6: "SORT_BY_DESCRIPTION",
	}
	GetAccessTokensRequest_SortBy_value = map[string]int32{
		"SORT_BY_UNSPECIFIED": 0,
		"SORT_BY_USER_ID":     1,
		"SORT_BY_EXPIRY":      2,
		"SORT_BY_CREATED_AT":  3,
		"SORT_BY_TOKEN_TYPE":  4,
		"SORT_BY_REVOKED":     5,
		"SORT_BY_DESCRIPTION": 6,
	}
)

func (x GetAccessTokensRequest_SortBy) Enum() *GetAccessTokensRequest_SortBy {
	p := new(GetAccessTokensRequest_SortBy)
	*p = x
	return p
}

func (x GetAccessTokensRequest_SortBy) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (GetAccessTokensRequest_SortBy) Descriptor() protoreflect.EnumDescriptor {
	return file_determined_api_v1_token_proto_enumTypes[0].Descriptor()
}

func (GetAccessTokensRequest_SortBy) Type() protoreflect.EnumType {
	return &file_determined_api_v1_token_proto_enumTypes[0]
}

func (x GetAccessTokensRequest_SortBy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use GetAccessTokensRequest_SortBy.Descriptor instead.
func (GetAccessTokensRequest_SortBy) EnumDescriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{2, 0}
}

// Create the requested user's accessToken.
type PostAccessTokenRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of the user.
	UserId int32 `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	// Lifespan expressing how long the token should last. Should be a Go-format
	// duration (e.g. "2s", "4m", "72h".)
	Lifespan *string `protobuf:"bytes,2,opt,name=lifespan,proto3,oneof" json:"lifespan,omitempty"`
	// Description of the token.
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
}

func (x *PostAccessTokenRequest) Reset() {
	*x = PostAccessTokenRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_token_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostAccessTokenRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostAccessTokenRequest) ProtoMessage() {}

func (x *PostAccessTokenRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_token_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostAccessTokenRequest.ProtoReflect.Descriptor instead.
func (*PostAccessTokenRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{0}
}

func (x *PostAccessTokenRequest) GetUserId() int32 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *PostAccessTokenRequest) GetLifespan() string {
	if x != nil && x.Lifespan != nil {
		return *x.Lifespan
	}
	return ""
}

func (x *PostAccessTokenRequest) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

// Response to PostAccessTokenRequest.
type PostAccessTokenResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// token value string.
	Token string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	// token id.
	TokenId int32 `protobuf:"varint,2,opt,name=token_id,json=tokenId,proto3" json:"token_id,omitempty"`
}

func (x *PostAccessTokenResponse) Reset() {
	*x = PostAccessTokenResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_token_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostAccessTokenResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostAccessTokenResponse) ProtoMessage() {}

func (x *PostAccessTokenResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_token_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostAccessTokenResponse.ProtoReflect.Descriptor instead.
func (*PostAccessTokenResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{1}
}

func (x *PostAccessTokenResponse) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *PostAccessTokenResponse) GetTokenId() int32 {
	if x != nil {
		return x.TokenId
	}
	return 0
}

// Get access tokens info for admin.
type GetAccessTokensRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Sort token info by the given field.
	SortBy GetAccessTokensRequest_SortBy `protobuf:"varint,1,opt,name=sort_by,json=sortBy,proto3,enum=determined.api.v1.GetAccessTokensRequest_SortBy" json:"sort_by,omitempty"`
	// Order token info in either ascending or descending order.
	OrderBy OrderBy `protobuf:"varint,2,opt,name=order_by,json=orderBy,proto3,enum=determined.api.v1.OrderBy" json:"order_by,omitempty"`
	// Skip the number of projects before returning results. Negative values
	// denote number of projects to skip from the end before returning results.
	Offset int32 `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	// Limit the number of projects. A value of 0 denotes no limit.
	Limit int32 `protobuf:"varint,4,opt,name=limit,proto3" json:"limit,omitempty"`
	// Filter on token_ids
	TokenIds []int32 `protobuf:"varint,5,rep,packed,name=token_ids,json=tokenIds,proto3" json:"token_ids,omitempty"`
	// Filter by username.
	Username string `protobuf:"bytes,6,opt,name=username,proto3" json:"username,omitempty"`
	// Filter by active status.
	ShowInactive bool `protobuf:"varint,7,opt,name=show_inactive,json=showInactive,proto3" json:"show_inactive,omitempty"`
}

func (x *GetAccessTokensRequest) Reset() {
	*x = GetAccessTokensRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_token_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAccessTokensRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAccessTokensRequest) ProtoMessage() {}

func (x *GetAccessTokensRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_token_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAccessTokensRequest.ProtoReflect.Descriptor instead.
func (*GetAccessTokensRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{2}
}

func (x *GetAccessTokensRequest) GetSortBy() GetAccessTokensRequest_SortBy {
	if x != nil {
		return x.SortBy
	}
	return GetAccessTokensRequest_SORT_BY_UNSPECIFIED
}

func (x *GetAccessTokensRequest) GetOrderBy() OrderBy {
	if x != nil {
		return x.OrderBy
	}
	return OrderBy_ORDER_BY_UNSPECIFIED
}

func (x *GetAccessTokensRequest) GetOffset() int32 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *GetAccessTokensRequest) GetLimit() int32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

func (x *GetAccessTokensRequest) GetTokenIds() []int32 {
	if x != nil {
		return x.TokenIds
	}
	return nil
}

func (x *GetAccessTokensRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *GetAccessTokensRequest) GetShowInactive() bool {
	if x != nil {
		return x.ShowInactive
	}
	return false
}

// Response to GetAccessTokensRequest.
type GetAccessTokensResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// List of token information.
	TokenInfo []*userv1.TokenInfo `protobuf:"bytes,1,rep,name=token_info,json=tokenInfo,proto3" json:"token_info,omitempty"`
	// Pagination information of the full dataset.
	Pagination *Pagination `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

func (x *GetAccessTokensResponse) Reset() {
	*x = GetAccessTokensResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_token_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAccessTokensResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAccessTokensResponse) ProtoMessage() {}

func (x *GetAccessTokensResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_token_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAccessTokensResponse.ProtoReflect.Descriptor instead.
func (*GetAccessTokensResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{3}
}

func (x *GetAccessTokensResponse) GetTokenInfo() []*userv1.TokenInfo {
	if x != nil {
		return x.TokenInfo
	}
	return nil
}

func (x *GetAccessTokensResponse) GetPagination() *Pagination {
	if x != nil {
		return x.Pagination
	}
	return nil
}

// Patch user's access token info.
type PatchAccessTokenRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of the token.
	TokenId int32 `protobuf:"varint,1,opt,name=token_id,json=tokenId,proto3" json:"token_id,omitempty"`
	// The requested updated token description.
	Description *string `protobuf:"bytes,2,opt,name=description,proto3,oneof" json:"description,omitempty"`
	// The requested updated token revoke status.
	SetRevoked bool `protobuf:"varint,3,opt,name=set_revoked,json=setRevoked,proto3" json:"set_revoked,omitempty"`
}

func (x *PatchAccessTokenRequest) Reset() {
	*x = PatchAccessTokenRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_token_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PatchAccessTokenRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PatchAccessTokenRequest) ProtoMessage() {}

func (x *PatchAccessTokenRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_token_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PatchAccessTokenRequest.ProtoReflect.Descriptor instead.
func (*PatchAccessTokenRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{4}
}

func (x *PatchAccessTokenRequest) GetTokenId() int32 {
	if x != nil {
		return x.TokenId
	}
	return 0
}

func (x *PatchAccessTokenRequest) GetDescription() string {
	if x != nil && x.Description != nil {
		return *x.Description
	}
	return ""
}

func (x *PatchAccessTokenRequest) GetSetRevoked() bool {
	if x != nil {
		return x.SetRevoked
	}
	return false
}

// Response to PatchAccessTokenRequest.
type PatchAccessTokenResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The updated token information.
	TokenInfo *userv1.TokenInfo `protobuf:"bytes,1,opt,name=token_info,json=tokenInfo,proto3" json:"token_info,omitempty"`
}

func (x *PatchAccessTokenResponse) Reset() {
	*x = PatchAccessTokenResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_token_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PatchAccessTokenResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PatchAccessTokenResponse) ProtoMessage() {}

func (x *PatchAccessTokenResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_token_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PatchAccessTokenResponse.ProtoReflect.Descriptor instead.
func (*PatchAccessTokenResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_token_proto_rawDescGZIP(), []int{5}
}

func (x *PatchAccessTokenResponse) GetTokenInfo() *userv1.TokenInfo {
	if x != nil {
		return x.TokenInfo
	}
	return nil
}

var File_determined_api_v1_token_proto protoreflect.FileDescriptor

var file_determined_api_v1_token_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x76, 0x31, 0x2f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x11, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x76, 0x31, 0x1a, 0x1d, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x75,
	0x73, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x22, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65,
	0x6e, 0x2d, 0x73, 0x77, 0x61, 0x67, 0x67, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x92, 0x01, 0x0a, 0x16, 0x50, 0x6f, 0x73, 0x74, 0x41, 0x63, 0x63, 0x65,
	0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17,
	0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x08, 0x6c, 0x69, 0x66, 0x65, 0x73,
	0x70, 0x61, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x08, 0x6c, 0x69, 0x66,
	0x65, 0x73, 0x70, 0x61, 0x6e, 0x88, 0x01, 0x01, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x3a, 0x0f, 0x92, 0x41, 0x0c, 0x0a,
	0x0a, 0xd2, 0x01, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x42, 0x0b, 0x0a, 0x09, 0x5f,
	0x6c, 0x69, 0x66, 0x65, 0x73, 0x70, 0x61, 0x6e, 0x22, 0x4a, 0x0a, 0x17, 0x50, 0x6f, 0x73, 0x74,
	0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x19, 0x0a, 0x08, 0x74, 0x6f, 0x6b,
	0x65, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x74, 0x6f, 0x6b,
	0x65, 0x6e, 0x49, 0x64, 0x22, 0xd8, 0x03, 0x0a, 0x16, 0x47, 0x65, 0x74, 0x41, 0x63, 0x63, 0x65,
	0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x49, 0x0a, 0x07, 0x73, 0x6f, 0x72, 0x74, 0x5f, 0x62, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x30, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x6f,
	0x6b, 0x65, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x6f, 0x72, 0x74,
	0x42, 0x79, 0x52, 0x06, 0x73, 0x6f, 0x72, 0x74, 0x42, 0x79, 0x12, 0x35, 0x0a, 0x08, 0x6f, 0x72,
	0x64, 0x65, 0x72, 0x5f, 0x62, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a, 0x2e, 0x64,
	0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x42, 0x79, 0x52, 0x07, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x42,
	0x79, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x69, 0x6d,
	0x69, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x12,
	0x1b, 0x0a, 0x09, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x5f, 0x69, 0x64, 0x73, 0x18, 0x05, 0x20, 0x03,
	0x28, 0x05, 0x52, 0x08, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x64, 0x73, 0x12, 0x1a, 0x0a, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x73, 0x68, 0x6f, 0x77,
	0x5f, 0x69, 0x6e, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x0c, 0x73, 0x68, 0x6f, 0x77, 0x49, 0x6e, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x22, 0xa8, 0x01,
	0x0a, 0x06, 0x53, 0x6f, 0x72, 0x74, 0x42, 0x79, 0x12, 0x17, 0x0a, 0x13, 0x53, 0x4f, 0x52, 0x54,
	0x5f, 0x42, 0x59, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10,
	0x00, 0x12, 0x13, 0x0a, 0x0f, 0x53, 0x4f, 0x52, 0x54, 0x5f, 0x42, 0x59, 0x5f, 0x55, 0x53, 0x45,
	0x52, 0x5f, 0x49, 0x44, 0x10, 0x01, 0x12, 0x12, 0x0a, 0x0e, 0x53, 0x4f, 0x52, 0x54, 0x5f, 0x42,
	0x59, 0x5f, 0x45, 0x58, 0x50, 0x49, 0x52, 0x59, 0x10, 0x02, 0x12, 0x16, 0x0a, 0x12, 0x53, 0x4f,
	0x52, 0x54, 0x5f, 0x42, 0x59, 0x5f, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x44, 0x5f, 0x41, 0x54,
	0x10, 0x03, 0x12, 0x16, 0x0a, 0x12, 0x53, 0x4f, 0x52, 0x54, 0x5f, 0x42, 0x59, 0x5f, 0x54, 0x4f,
	0x4b, 0x45, 0x4e, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x10, 0x04, 0x12, 0x13, 0x0a, 0x0f, 0x53, 0x4f,
	0x52, 0x54, 0x5f, 0x42, 0x59, 0x5f, 0x52, 0x45, 0x56, 0x4f, 0x4b, 0x45, 0x44, 0x10, 0x05, 0x12,
	0x17, 0x0a, 0x13, 0x53, 0x4f, 0x52, 0x54, 0x5f, 0x42, 0x59, 0x5f, 0x44, 0x45, 0x53, 0x43, 0x52,
	0x49, 0x50, 0x54, 0x49, 0x4f, 0x4e, 0x10, 0x06, 0x3a, 0x05, 0x92, 0x41, 0x02, 0x0a, 0x00, 0x22,
	0xaa, 0x01, 0x0a, 0x17, 0x47, 0x65, 0x74, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x6f, 0x6b,
	0x65, 0x6e, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3c, 0x0a, 0x0a, 0x74,
	0x6f, 0x6b, 0x65, 0x6e, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1d, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x75, 0x73, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x09,
	0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x3d, 0x0a, 0x0a, 0x70, 0x61, 0x67,
	0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e,
	0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x50, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x70, 0x61,
	0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x3a, 0x12, 0x92, 0x41, 0x0f, 0x0a, 0x0d, 0xd2,
	0x01, 0x0a, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x22, 0x9e, 0x01, 0x0a,
	0x17, 0x50, 0x61, 0x74, 0x63, 0x68, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x49, 0x64, 0x12, 0x25, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x88, 0x01, 0x01, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x65,
	0x74, 0x5f, 0x72, 0x65, 0x76, 0x6f, 0x6b, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x0a, 0x73, 0x65, 0x74, 0x52, 0x65, 0x76, 0x6f, 0x6b, 0x65, 0x64, 0x3a, 0x10, 0x92, 0x41, 0x0d,
	0x0a, 0x0b, 0xd2, 0x01, 0x08, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x5f, 0x69, 0x64, 0x42, 0x0e, 0x0a,
	0x0c, 0x5f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x58, 0x0a,
	0x18, 0x50, 0x61, 0x74, 0x63, 0x68, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3c, 0x0a, 0x0a, 0x74, 0x6f, 0x6b,
	0x65, 0x6e, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e,
	0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x09, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64,
	0x2d, 0x61, 0x69, 0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x76, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_determined_api_v1_token_proto_rawDescOnce sync.Once
	file_determined_api_v1_token_proto_rawDescData = file_determined_api_v1_token_proto_rawDesc
)

func file_determined_api_v1_token_proto_rawDescGZIP() []byte {
	file_determined_api_v1_token_proto_rawDescOnce.Do(func() {
		file_determined_api_v1_token_proto_rawDescData = protoimpl.X.CompressGZIP(file_determined_api_v1_token_proto_rawDescData)
	})
	return file_determined_api_v1_token_proto_rawDescData
}

var file_determined_api_v1_token_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_determined_api_v1_token_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_determined_api_v1_token_proto_goTypes = []interface{}{
	(GetAccessTokensRequest_SortBy)(0), // 0: determined.api.v1.GetAccessTokensRequest.SortBy
	(*PostAccessTokenRequest)(nil),     // 1: determined.api.v1.PostAccessTokenRequest
	(*PostAccessTokenResponse)(nil),    // 2: determined.api.v1.PostAccessTokenResponse
	(*GetAccessTokensRequest)(nil),     // 3: determined.api.v1.GetAccessTokensRequest
	(*GetAccessTokensResponse)(nil),    // 4: determined.api.v1.GetAccessTokensResponse
	(*PatchAccessTokenRequest)(nil),    // 5: determined.api.v1.PatchAccessTokenRequest
	(*PatchAccessTokenResponse)(nil),   // 6: determined.api.v1.PatchAccessTokenResponse
	(OrderBy)(0),                       // 7: determined.api.v1.OrderBy
	(*userv1.TokenInfo)(nil),           // 8: determined.user.v1.TokenInfo
	(*Pagination)(nil),                 // 9: determined.api.v1.Pagination
}
var file_determined_api_v1_token_proto_depIdxs = []int32{
	0, // 0: determined.api.v1.GetAccessTokensRequest.sort_by:type_name -> determined.api.v1.GetAccessTokensRequest.SortBy
	7, // 1: determined.api.v1.GetAccessTokensRequest.order_by:type_name -> determined.api.v1.OrderBy
	8, // 2: determined.api.v1.GetAccessTokensResponse.token_info:type_name -> determined.user.v1.TokenInfo
	9, // 3: determined.api.v1.GetAccessTokensResponse.pagination:type_name -> determined.api.v1.Pagination
	8, // 4: determined.api.v1.PatchAccessTokenResponse.token_info:type_name -> determined.user.v1.TokenInfo
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_determined_api_v1_token_proto_init() }
func file_determined_api_v1_token_proto_init() {
	if File_determined_api_v1_token_proto != nil {
		return
	}
	file_determined_api_v1_pagination_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_determined_api_v1_token_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostAccessTokenRequest); i {
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
		file_determined_api_v1_token_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostAccessTokenResponse); i {
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
		file_determined_api_v1_token_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAccessTokensRequest); i {
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
		file_determined_api_v1_token_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAccessTokensResponse); i {
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
		file_determined_api_v1_token_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PatchAccessTokenRequest); i {
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
		file_determined_api_v1_token_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PatchAccessTokenResponse); i {
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
	file_determined_api_v1_token_proto_msgTypes[0].OneofWrappers = []interface{}{}
	file_determined_api_v1_token_proto_msgTypes[4].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_determined_api_v1_token_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_determined_api_v1_token_proto_goTypes,
		DependencyIndexes: file_determined_api_v1_token_proto_depIdxs,
		EnumInfos:         file_determined_api_v1_token_proto_enumTypes,
		MessageInfos:      file_determined_api_v1_token_proto_msgTypes,
	}.Build()
	File_determined_api_v1_token_proto = out.File
	file_determined_api_v1_token_proto_rawDesc = nil
	file_determined_api_v1_token_proto_goTypes = nil
	file_determined_api_v1_token_proto_depIdxs = nil
}
