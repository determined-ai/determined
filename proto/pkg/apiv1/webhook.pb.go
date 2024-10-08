// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// source: determined/api/v1/webhook.proto

package apiv1

import (
	webhookv1 "github.com/determined-ai/determined/proto/pkg/webhookv1"
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

// Get a single webhook.
type GetWebhookRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of the webhook.
	WebhookId int32 `protobuf:"varint,1,opt,name=webhook_id,json=webhookId,proto3" json:"webhook_id,omitempty"`
}

func (x *GetWebhookRequest) Reset() {
	*x = GetWebhookRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetWebhookRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetWebhookRequest) ProtoMessage() {}

func (x *GetWebhookRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetWebhookRequest.ProtoReflect.Descriptor instead.
func (*GetWebhookRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{0}
}

func (x *GetWebhookRequest) GetWebhookId() int32 {
	if x != nil {
		return x.WebhookId
	}
	return 0
}

// Response to GetWebhookRequest.
type GetWebhookResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The requested Webhook.
	Webhook *webhookv1.Webhook `protobuf:"bytes,1,opt,name=webhook,proto3" json:"webhook,omitempty"`
}

func (x *GetWebhookResponse) Reset() {
	*x = GetWebhookResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetWebhookResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetWebhookResponse) ProtoMessage() {}

func (x *GetWebhookResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetWebhookResponse.ProtoReflect.Descriptor instead.
func (*GetWebhookResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{1}
}

func (x *GetWebhookResponse) GetWebhook() *webhookv1.Webhook {
	if x != nil {
		return x.Webhook
	}
	return nil
}

// Get a list of webhooks.
type GetWebhooksRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetWebhooksRequest) Reset() {
	*x = GetWebhooksRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetWebhooksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetWebhooksRequest) ProtoMessage() {}

func (x *GetWebhooksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetWebhooksRequest.ProtoReflect.Descriptor instead.
func (*GetWebhooksRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{2}
}

// Response to GetWebhooksRequest.
type GetWebhooksResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The list of returned webhooks.
	Webhooks []*webhookv1.Webhook `protobuf:"bytes,1,rep,name=webhooks,proto3" json:"webhooks,omitempty"`
}

func (x *GetWebhooksResponse) Reset() {
	*x = GetWebhooksResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetWebhooksResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetWebhooksResponse) ProtoMessage() {}

func (x *GetWebhooksResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetWebhooksResponse.ProtoReflect.Descriptor instead.
func (*GetWebhooksResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{3}
}

func (x *GetWebhooksResponse) GetWebhooks() []*webhookv1.Webhook {
	if x != nil {
		return x.Webhooks
	}
	return nil
}

// Request for creating a webhook
type PostWebhookRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The webhook to store.
	Webhook *webhookv1.Webhook `protobuf:"bytes,1,opt,name=webhook,proto3" json:"webhook,omitempty"`
}

func (x *PostWebhookRequest) Reset() {
	*x = PostWebhookRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostWebhookRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostWebhookRequest) ProtoMessage() {}

func (x *PostWebhookRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostWebhookRequest.ProtoReflect.Descriptor instead.
func (*PostWebhookRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{4}
}

func (x *PostWebhookRequest) GetWebhook() *webhookv1.Webhook {
	if x != nil {
		return x.Webhook
	}
	return nil
}

// Response to PostWebhookRequest.
type PostWebhookResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The webhook created.
	Webhook *webhookv1.Webhook `protobuf:"bytes,1,opt,name=webhook,proto3" json:"webhook,omitempty"`
}

func (x *PostWebhookResponse) Reset() {
	*x = PostWebhookResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostWebhookResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostWebhookResponse) ProtoMessage() {}

func (x *PostWebhookResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostWebhookResponse.ProtoReflect.Descriptor instead.
func (*PostWebhookResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{5}
}

func (x *PostWebhookResponse) GetWebhook() *webhookv1.Webhook {
	if x != nil {
		return x.Webhook
	}
	return nil
}

// Request for deleting a webhook.
type DeleteWebhookRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of the webhook.
	Id int32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *DeleteWebhookRequest) Reset() {
	*x = DeleteWebhookRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteWebhookRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteWebhookRequest) ProtoMessage() {}

func (x *DeleteWebhookRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteWebhookRequest.ProtoReflect.Descriptor instead.
func (*DeleteWebhookRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{6}
}

func (x *DeleteWebhookRequest) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

// Response to DeleteWebhookRequest.
type DeleteWebhookResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeleteWebhookResponse) Reset() {
	*x = DeleteWebhookResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteWebhookResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteWebhookResponse) ProtoMessage() {}

func (x *DeleteWebhookResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteWebhookResponse.ProtoReflect.Descriptor instead.
func (*DeleteWebhookResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{7}
}

// Request for testing a webhook.
type TestWebhookRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of the webhook.
	Id int32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *TestWebhookRequest) Reset() {
	*x = TestWebhookRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestWebhookRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestWebhookRequest) ProtoMessage() {}

func (x *TestWebhookRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestWebhookRequest.ProtoReflect.Descriptor instead.
func (*TestWebhookRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{8}
}

func (x *TestWebhookRequest) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

// Response to TestWebhookRequest.
type TestWebhookResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Status of test.
	Completed bool `protobuf:"varint,1,opt,name=completed,proto3" json:"completed,omitempty"`
}

func (x *TestWebhookResponse) Reset() {
	*x = TestWebhookResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestWebhookResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestWebhookResponse) ProtoMessage() {}

func (x *TestWebhookResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestWebhookResponse.ProtoReflect.Descriptor instead.
func (*TestWebhookResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{9}
}

func (x *TestWebhookResponse) GetCompleted() bool {
	if x != nil {
		return x.Completed
	}
	return false
}

// Request for triggering custom trigger.
type PostWebhookEventDataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The event data for custom webhook trigger.
	Data *webhookv1.CustomWebhookEventData `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	// The id of the experiment.
	ExperimentId int32 `protobuf:"varint,2,opt,name=experiment_id,json=experimentId,proto3" json:"experiment_id,omitempty"`
	// The id of the trial.
	TrialId int32 `protobuf:"varint,3,opt,name=trial_id,json=trialId,proto3" json:"trial_id,omitempty"`
}

func (x *PostWebhookEventDataRequest) Reset() {
	*x = PostWebhookEventDataRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostWebhookEventDataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostWebhookEventDataRequest) ProtoMessage() {}

func (x *PostWebhookEventDataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostWebhookEventDataRequest.ProtoReflect.Descriptor instead.
func (*PostWebhookEventDataRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{10}
}

func (x *PostWebhookEventDataRequest) GetData() *webhookv1.CustomWebhookEventData {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *PostWebhookEventDataRequest) GetExperimentId() int32 {
	if x != nil {
		return x.ExperimentId
	}
	return 0
}

func (x *PostWebhookEventDataRequest) GetTrialId() int32 {
	if x != nil {
		return x.TrialId
	}
	return 0
}

// Response to PostWebhookEventDataRequest.
type PostWebhookEventDataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *PostWebhookEventDataResponse) Reset() {
	*x = PostWebhookEventDataResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostWebhookEventDataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostWebhookEventDataResponse) ProtoMessage() {}

func (x *PostWebhookEventDataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostWebhookEventDataResponse.ProtoReflect.Descriptor instead.
func (*PostWebhookEventDataResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{11}
}

// Request for updating a webhook.
type PatchWebhookRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of the webhook.
	Id int32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// The desired webhook fields and values to update.
	Webhook *webhookv1.PatchWebhook `protobuf:"bytes,2,opt,name=webhook,proto3" json:"webhook,omitempty"`
}

func (x *PatchWebhookRequest) Reset() {
	*x = PatchWebhookRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PatchWebhookRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PatchWebhookRequest) ProtoMessage() {}

func (x *PatchWebhookRequest) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PatchWebhookRequest.ProtoReflect.Descriptor instead.
func (*PatchWebhookRequest) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{12}
}

func (x *PatchWebhookRequest) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *PatchWebhookRequest) GetWebhook() *webhookv1.PatchWebhook {
	if x != nil {
		return x.Webhook
	}
	return nil
}

// Response to PatchWebhookRequest.
type PatchWebhookResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *PatchWebhookResponse) Reset() {
	*x = PatchWebhookResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_determined_api_v1_webhook_proto_msgTypes[13]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PatchWebhookResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PatchWebhookResponse) ProtoMessage() {}

func (x *PatchWebhookResponse) ProtoReflect() protoreflect.Message {
	mi := &file_determined_api_v1_webhook_proto_msgTypes[13]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PatchWebhookResponse.ProtoReflect.Descriptor instead.
func (*PatchWebhookResponse) Descriptor() ([]byte, []int) {
	return file_determined_api_v1_webhook_proto_rawDescGZIP(), []int{13}
}

var File_determined_api_v1_webhook_proto protoreflect.FileDescriptor

var file_determined_api_v1_webhook_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x76, 0x31, 0x2f, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x11, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x1a, 0x2c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e,
	0x2d, 0x73, 0x77, 0x61, 0x67, 0x67, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x23, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2f, 0x76, 0x31, 0x2f, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f,
	0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x32, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x57, 0x65,
	0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a,
	0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x09, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x49, 0x64, 0x22, 0x5f, 0x0a, 0x12, 0x47,
	0x65, 0x74, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x38, 0x0a, 0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e,
	0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f,
	0x6f, 0x6b, 0x52, 0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x3a, 0x0f, 0x92, 0x41, 0x0c,
	0x0a, 0x0a, 0xd2, 0x01, 0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x22, 0x14, 0x0a, 0x12,
	0x47, 0x65, 0x74, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x22, 0x63, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3a, 0x0a, 0x08, 0x77, 0x65, 0x62,
	0x68, 0x6f, 0x6f, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x64, 0x65,
	0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b,
	0x2e, 0x76, 0x31, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x08, 0x77, 0x65, 0x62,
	0x68, 0x6f, 0x6f, 0x6b, 0x73, 0x3a, 0x10, 0x92, 0x41, 0x0d, 0x0a, 0x0b, 0xd2, 0x01, 0x08, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x73, 0x22, 0x5f, 0x0a, 0x12, 0x50, 0x6f, 0x73, 0x74, 0x57,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x38, 0x0a,
	0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e,
	0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x77, 0x65, 0x62, 0x68,
	0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x07,
	0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x3a, 0x0f, 0x92, 0x41, 0x0c, 0x0a, 0x0a, 0xd2, 0x01,
	0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x22, 0x60, 0x0a, 0x13, 0x50, 0x6f, 0x73, 0x74,
	0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x38, 0x0a, 0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1e, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x77, 0x65,
	0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b,
	0x52, 0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x3a, 0x0f, 0x92, 0x41, 0x0c, 0x0a, 0x0a,
	0xd2, 0x01, 0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x22, 0x32, 0x0a, 0x14, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02,
	0x69, 0x64, 0x3a, 0x0a, 0x92, 0x41, 0x07, 0x0a, 0x05, 0xd2, 0x01, 0x02, 0x69, 0x64, 0x22, 0x17,
	0x0a, 0x15, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x30, 0x0a, 0x12, 0x54, 0x65, 0x73, 0x74, 0x57,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02, 0x69, 0x64, 0x3a, 0x0a, 0x92,
	0x41, 0x07, 0x0a, 0x05, 0xd2, 0x01, 0x02, 0x69, 0x64, 0x22, 0x46, 0x0a, 0x13, 0x54, 0x65, 0x73,
	0x74, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x1c, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x3a, 0x11,
	0x92, 0x41, 0x0e, 0x0a, 0x0c, 0xd2, 0x01, 0x09, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65,
	0x64, 0x22, 0xbe, 0x01, 0x0a, 0x1b, 0x50, 0x6f, 0x73, 0x74, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f,
	0x6b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x41, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2d, 0x2e, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x77, 0x65, 0x62,
	0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x57, 0x65,
	0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x44, 0x61, 0x74, 0x61, 0x52, 0x04,
	0x64, 0x61, 0x74, 0x61, 0x12, 0x23, 0x0a, 0x0d, 0x65, 0x78, 0x70, 0x65, 0x72, 0x69, 0x6d, 0x65,
	0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x65, 0x78, 0x70,
	0x65, 0x72, 0x69, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x74, 0x72, 0x69,
	0x61, 0x6c, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x74, 0x72, 0x69,
	0x61, 0x6c, 0x49, 0x64, 0x3a, 0x1c, 0x92, 0x41, 0x19, 0x0a, 0x17, 0xd2, 0x01, 0x04, 0x64, 0x61,
	0x74, 0x61, 0xd2, 0x01, 0x0d, 0x65, 0x78, 0x70, 0x65, 0x72, 0x69, 0x6d, 0x65, 0x6e, 0x74, 0x5f,
	0x69, 0x64, 0x22, 0x1e, 0x0a, 0x1c, 0x50, 0x6f, 0x73, 0x74, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f,
	0x6b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x79, 0x0a, 0x13, 0x50, 0x61, 0x74, 0x63, 0x68, 0x57, 0x65, 0x62, 0x68, 0x6f,
	0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02, 0x69, 0x64, 0x12, 0x3d, 0x0a, 0x07, 0x77, 0x65, 0x62,
	0x68, 0x6f, 0x6f, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64, 0x65, 0x74,
	0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e,
	0x76, 0x31, 0x2e, 0x50, 0x61, 0x74, 0x63, 0x68, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52,
	0x07, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x3a, 0x13, 0x92, 0x41, 0x10, 0x0a, 0x0e, 0xd2,
	0x01, 0x02, 0x69, 0x64, 0xd2, 0x01, 0x06, 0x77, 0x65, 0x62, 0x6f, 0x6f, 0x6b, 0x22, 0x16, 0x0a,
	0x14, 0x50, 0x61, 0x74, 0x63, 0x68, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2d, 0x61,
	0x69, 0x2f, 0x64, 0x65, 0x74, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x65, 0x64, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_determined_api_v1_webhook_proto_rawDescOnce sync.Once
	file_determined_api_v1_webhook_proto_rawDescData = file_determined_api_v1_webhook_proto_rawDesc
)

func file_determined_api_v1_webhook_proto_rawDescGZIP() []byte {
	file_determined_api_v1_webhook_proto_rawDescOnce.Do(func() {
		file_determined_api_v1_webhook_proto_rawDescData = protoimpl.X.CompressGZIP(file_determined_api_v1_webhook_proto_rawDescData)
	})
	return file_determined_api_v1_webhook_proto_rawDescData
}

var file_determined_api_v1_webhook_proto_msgTypes = make([]protoimpl.MessageInfo, 14)
var file_determined_api_v1_webhook_proto_goTypes = []interface{}{
	(*GetWebhookRequest)(nil),                // 0: determined.api.v1.GetWebhookRequest
	(*GetWebhookResponse)(nil),               // 1: determined.api.v1.GetWebhookResponse
	(*GetWebhooksRequest)(nil),               // 2: determined.api.v1.GetWebhooksRequest
	(*GetWebhooksResponse)(nil),              // 3: determined.api.v1.GetWebhooksResponse
	(*PostWebhookRequest)(nil),               // 4: determined.api.v1.PostWebhookRequest
	(*PostWebhookResponse)(nil),              // 5: determined.api.v1.PostWebhookResponse
	(*DeleteWebhookRequest)(nil),             // 6: determined.api.v1.DeleteWebhookRequest
	(*DeleteWebhookResponse)(nil),            // 7: determined.api.v1.DeleteWebhookResponse
	(*TestWebhookRequest)(nil),               // 8: determined.api.v1.TestWebhookRequest
	(*TestWebhookResponse)(nil),              // 9: determined.api.v1.TestWebhookResponse
	(*PostWebhookEventDataRequest)(nil),      // 10: determined.api.v1.PostWebhookEventDataRequest
	(*PostWebhookEventDataResponse)(nil),     // 11: determined.api.v1.PostWebhookEventDataResponse
	(*PatchWebhookRequest)(nil),              // 12: determined.api.v1.PatchWebhookRequest
	(*PatchWebhookResponse)(nil),             // 13: determined.api.v1.PatchWebhookResponse
	(*webhookv1.Webhook)(nil),                // 14: determined.webhook.v1.Webhook
	(*webhookv1.CustomWebhookEventData)(nil), // 15: determined.webhook.v1.CustomWebhookEventData
	(*webhookv1.PatchWebhook)(nil),           // 16: determined.webhook.v1.PatchWebhook
}
var file_determined_api_v1_webhook_proto_depIdxs = []int32{
	14, // 0: determined.api.v1.GetWebhookResponse.webhook:type_name -> determined.webhook.v1.Webhook
	14, // 1: determined.api.v1.GetWebhooksResponse.webhooks:type_name -> determined.webhook.v1.Webhook
	14, // 2: determined.api.v1.PostWebhookRequest.webhook:type_name -> determined.webhook.v1.Webhook
	14, // 3: determined.api.v1.PostWebhookResponse.webhook:type_name -> determined.webhook.v1.Webhook
	15, // 4: determined.api.v1.PostWebhookEventDataRequest.data:type_name -> determined.webhook.v1.CustomWebhookEventData
	16, // 5: determined.api.v1.PatchWebhookRequest.webhook:type_name -> determined.webhook.v1.PatchWebhook
	6,  // [6:6] is the sub-list for method output_type
	6,  // [6:6] is the sub-list for method input_type
	6,  // [6:6] is the sub-list for extension type_name
	6,  // [6:6] is the sub-list for extension extendee
	0,  // [0:6] is the sub-list for field type_name
}

func init() { file_determined_api_v1_webhook_proto_init() }
func file_determined_api_v1_webhook_proto_init() {
	if File_determined_api_v1_webhook_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_determined_api_v1_webhook_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetWebhookRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetWebhookResponse); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetWebhooksRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetWebhooksResponse); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostWebhookRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostWebhookResponse); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteWebhookRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteWebhookResponse); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestWebhookRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestWebhookResponse); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostWebhookEventDataRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostWebhookEventDataResponse); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PatchWebhookRequest); i {
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
		file_determined_api_v1_webhook_proto_msgTypes[13].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PatchWebhookResponse); i {
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
			RawDescriptor: file_determined_api_v1_webhook_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   14,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_determined_api_v1_webhook_proto_goTypes,
		DependencyIndexes: file_determined_api_v1_webhook_proto_depIdxs,
		MessageInfos:      file_determined_api_v1_webhook_proto_msgTypes,
	}.Build()
	File_determined_api_v1_webhook_proto = out.File
	file_determined_api_v1_webhook_proto_rawDesc = nil
	file_determined_api_v1_webhook_proto_goTypes = nil
	file_determined_api_v1_webhook_proto_depIdxs = nil
}
