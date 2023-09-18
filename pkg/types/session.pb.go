// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: session.proto

package types

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	anypb "google.golang.org/protobuf/types/known/anypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type SessionType int32

const (
	SessionType_DefaultSession SessionType = 0
	SessionType_ReshareSession SessionType = 1
	SessionType_KeygenSession  SessionType = 2
)

// Enum value maps for SessionType.
var (
	SessionType_name = map[int32]string{
		0: "DefaultSession",
		1: "ReshareSession",
		2: "KeygenSession",
	}
	SessionType_value = map[string]int32{
		"DefaultSession": 0,
		"ReshareSession": 1,
		"KeygenSession":  2,
	}
)

func (x SessionType) Enum() *SessionType {
	p := new(SessionType)
	*p = x
	return p
}

func (x SessionType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SessionType) Descriptor() protoreflect.EnumDescriptor {
	return file_session_proto_enumTypes[0].Descriptor()
}

func (SessionType) Type() protoreflect.EnumType {
	return &file_session_proto_enumTypes[0]
}

func (x SessionType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SessionType.Descriptor instead.
func (SessionType) EnumDescriptor() ([]byte, []int) {
	return file_session_proto_rawDescGZIP(), []int{0}
}

type SessionStatus int32

const (
	SessionStatus_SessionProcessing SessionStatus = 0
	SessionStatus_SessionFailed     SessionStatus = 1
	SessionStatus_SessionSucceeded  SessionStatus = 3
)

// Enum value maps for SessionStatus.
var (
	SessionStatus_name = map[int32]string{
		0: "SessionProcessing",
		1: "SessionFailed",
		3: "SessionSucceeded",
	}
	SessionStatus_value = map[string]int32{
		"SessionProcessing": 0,
		"SessionFailed":     1,
		"SessionSucceeded":  3,
	}
)

func (x SessionStatus) Enum() *SessionStatus {
	p := new(SessionStatus)
	*p = x
	return p
}

func (x SessionStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SessionStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_session_proto_enumTypes[1].Descriptor()
}

func (SessionStatus) Type() protoreflect.EnumType {
	return &file_session_proto_enumTypes[1]
}

func (x SessionStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SessionStatus.Descriptor instead.
func (SessionStatus) EnumDescriptor() ([]byte, []int) {
	return file_session_proto_rawDescGZIP(), []int{1}
}

type Session struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         uint64        `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Status     SessionStatus `protobuf:"varint,2,opt,name=status,proto3,enum=SessionStatus" json:"status,omitempty"`
	StartBlock uint64        `protobuf:"varint,3,opt,name=startBlock,proto3" json:"startBlock,omitempty"`
	EndBlock   uint64        `protobuf:"varint,4,opt,name=endBlock,proto3" json:"endBlock,omitempty"`
	Type       SessionType   `protobuf:"varint,5,opt,name=type,proto3,enum=SessionType" json:"type,omitempty"`
	Data       *anypb.Any    `protobuf:"bytes,6,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Session) Reset() {
	*x = Session{}
	if protoimpl.UnsafeEnabled {
		mi := &file_session_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Session) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session) ProtoMessage() {}

func (x *Session) ProtoReflect() protoreflect.Message {
	mi := &file_session_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session.ProtoReflect.Descriptor instead.
func (*Session) Descriptor() ([]byte, []int) {
	return file_session_proto_rawDescGZIP(), []int{0}
}

func (x *Session) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Session) GetStatus() SessionStatus {
	if x != nil {
		return x.Status
	}
	return SessionStatus_SessionProcessing
}

func (x *Session) GetStartBlock() uint64 {
	if x != nil {
		return x.StartBlock
	}
	return 0
}

func (x *Session) GetEndBlock() uint64 {
	if x != nil {
		return x.EndBlock
	}
	return 0
}

func (x *Session) GetType() SessionType {
	if x != nil {
		return x.Type
	}
	return SessionType_DefaultSession
}

func (x *Session) GetData() *anypb.Any {
	if x != nil {
		return x.Data
	}
	return nil
}

type DefaultSessionData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Parties   []string `protobuf:"bytes,1,rep,name=parties,proto3" json:"parties,omitempty"`
	Proposer  string   `protobuf:"bytes,2,opt,name=proposer,proto3" json:"proposer,omitempty"`
	Indexes   []string `protobuf:"bytes,3,rep,name=indexes,proto3" json:"indexes,omitempty"`
	Root      string   `protobuf:"bytes,4,opt,name=root,proto3" json:"root,omitempty"`
	Accepted  []string `protobuf:"bytes,5,rep,name=accepted,proto3" json:"accepted,omitempty"`
	Signature string   `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *DefaultSessionData) Reset() {
	*x = DefaultSessionData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_session_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DefaultSessionData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DefaultSessionData) ProtoMessage() {}

func (x *DefaultSessionData) ProtoReflect() protoreflect.Message {
	mi := &file_session_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DefaultSessionData.ProtoReflect.Descriptor instead.
func (*DefaultSessionData) Descriptor() ([]byte, []int) {
	return file_session_proto_rawDescGZIP(), []int{1}
}

func (x *DefaultSessionData) GetParties() []string {
	if x != nil {
		return x.Parties
	}
	return nil
}

func (x *DefaultSessionData) GetProposer() string {
	if x != nil {
		return x.Proposer
	}
	return ""
}

func (x *DefaultSessionData) GetIndexes() []string {
	if x != nil {
		return x.Indexes
	}
	return nil
}

func (x *DefaultSessionData) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

func (x *DefaultSessionData) GetAccepted() []string {
	if x != nil {
		return x.Accepted
	}
	return nil
}

func (x *DefaultSessionData) GetSignature() string {
	if x != nil {
		return x.Signature
	}
	return ""
}

type ReshareSessionData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Parties      []string `protobuf:"bytes,1,rep,name=parties,proto3" json:"parties,omitempty"`
	Proposer     string   `protobuf:"bytes,2,opt,name=proposer,proto3" json:"proposer,omitempty"`
	OldKey       string   `protobuf:"bytes,3,opt,name=oldKey,proto3" json:"oldKey,omitempty"`
	NewKey       string   `protobuf:"bytes,4,opt,name=newKey,proto3" json:"newKey,omitempty"`
	KeySignature string   `protobuf:"bytes,5,opt,name=keySignature,proto3" json:"keySignature,omitempty"`
	Signature    string   `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
	Root         string   `protobuf:"bytes,7,opt,name=root,proto3" json:"root,omitempty"`
}

func (x *ReshareSessionData) Reset() {
	*x = ReshareSessionData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_session_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReshareSessionData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReshareSessionData) ProtoMessage() {}

func (x *ReshareSessionData) ProtoReflect() protoreflect.Message {
	mi := &file_session_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReshareSessionData.ProtoReflect.Descriptor instead.
func (*ReshareSessionData) Descriptor() ([]byte, []int) {
	return file_session_proto_rawDescGZIP(), []int{2}
}

func (x *ReshareSessionData) GetParties() []string {
	if x != nil {
		return x.Parties
	}
	return nil
}

func (x *ReshareSessionData) GetProposer() string {
	if x != nil {
		return x.Proposer
	}
	return ""
}

func (x *ReshareSessionData) GetOldKey() string {
	if x != nil {
		return x.OldKey
	}
	return ""
}

func (x *ReshareSessionData) GetNewKey() string {
	if x != nil {
		return x.NewKey
	}
	return ""
}

func (x *ReshareSessionData) GetKeySignature() string {
	if x != nil {
		return x.KeySignature
	}
	return ""
}

func (x *ReshareSessionData) GetSignature() string {
	if x != nil {
		return x.Signature
	}
	return ""
}

func (x *ReshareSessionData) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

type KeygenSessionData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Parties []string `protobuf:"bytes,1,rep,name=parties,proto3" json:"parties,omitempty"`
	Key     string   `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *KeygenSessionData) Reset() {
	*x = KeygenSessionData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_session_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KeygenSessionData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KeygenSessionData) ProtoMessage() {}

func (x *KeygenSessionData) ProtoReflect() protoreflect.Message {
	mi := &file_session_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KeygenSessionData.ProtoReflect.Descriptor instead.
func (*KeygenSessionData) Descriptor() ([]byte, []int) {
	return file_session_proto_rawDescGZIP(), []int{3}
}

func (x *KeygenSessionData) GetParties() []string {
	if x != nil {
		return x.Parties
	}
	return nil
}

func (x *KeygenSessionData) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

var File_session_proto protoreflect.FileDescriptor

var file_session_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc9, 0x01, 0x0a, 0x07, 0x53,
	0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x26, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1e,
	0x0a, 0x0a, 0x73, 0x74, 0x61, 0x72, 0x74, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x72, 0x74, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x1a,
	0x0a, 0x08, 0x65, 0x6e, 0x64, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x08, 0x65, 0x6e, 0x64, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x20, 0x0a, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e, 0x53, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x28, 0x0a, 0x04,
	0x64, 0x61, 0x74, 0x61, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79,
	0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0xb2, 0x01, 0x0a, 0x12, 0x44, 0x65, 0x66, 0x61, 0x75,
	0x6c, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x44, 0x61, 0x74, 0x61, 0x12, 0x18, 0x0a,
	0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07,
	0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x70, 0x6f,
	0x73, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x72, 0x6f, 0x70, 0x6f,
	0x73, 0x65, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f,
	0x74, 0x12, 0x1a, 0x0a, 0x08, 0x61, 0x63, 0x63, 0x65, 0x70, 0x74, 0x65, 0x64, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x08, 0x61, 0x63, 0x63, 0x65, 0x70, 0x74, 0x65, 0x64, 0x12, 0x1c, 0x0a,
	0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0xd0, 0x01, 0x0a, 0x12,
	0x52, 0x65, 0x73, 0x68, 0x61, 0x72, 0x65, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x44, 0x61,
	0x74, 0x61, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x1a, 0x0a, 0x08,
	0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x6c, 0x64, 0x4b,
	0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6f, 0x6c, 0x64, 0x4b, 0x65, 0x79,
	0x12, 0x16, 0x0a, 0x06, 0x6e, 0x65, 0x77, 0x4b, 0x65, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x6e, 0x65, 0x77, 0x4b, 0x65, 0x79, 0x12, 0x22, 0x0a, 0x0c, 0x6b, 0x65, 0x79, 0x53,
	0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c,
	0x6b, 0x65, 0x79, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1c, 0x0a, 0x09,
	0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f,
	0x6f, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x22, 0x3f,
	0x0a, 0x11, 0x4b, 0x65, 0x79, 0x67, 0x65, 0x6e, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x44,
	0x61, 0x74, 0x61, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x2a,
	0x48, 0x0a, 0x0b, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x12,
	0x0a, 0x0e, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x10, 0x00, 0x12, 0x12, 0x0a, 0x0e, 0x52, 0x65, 0x73, 0x68, 0x61, 0x72, 0x65, 0x53, 0x65, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d, 0x4b, 0x65, 0x79, 0x67, 0x65, 0x6e,
	0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x10, 0x02, 0x2a, 0x4f, 0x0a, 0x0d, 0x53, 0x65, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x15, 0x0a, 0x11, 0x53, 0x65,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x69, 0x6e, 0x67, 0x10,
	0x00, 0x12, 0x11, 0x0a, 0x0d, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x46, 0x61, 0x69, 0x6c,
	0x65, 0x64, 0x10, 0x01, 0x12, 0x14, 0x0a, 0x10, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x53,
	0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x10, 0x03, 0x42, 0x29, 0x5a, 0x27, 0x67, 0x69,
	0x74, 0x6c, 0x61, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x72, 0x69, 0x6d, 0x6f, 0x2f,
	0x74, 0x73, 0x73, 0x2f, 0x74, 0x73, 0x73, 0x2d, 0x73, 0x76, 0x63, 0x2f, 0x70, 0x6b, 0x67, 0x2f,
	0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_session_proto_rawDescOnce sync.Once
	file_session_proto_rawDescData = file_session_proto_rawDesc
)

func file_session_proto_rawDescGZIP() []byte {
	file_session_proto_rawDescOnce.Do(func() {
		file_session_proto_rawDescData = protoimpl.X.CompressGZIP(file_session_proto_rawDescData)
	})
	return file_session_proto_rawDescData
}

var file_session_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_session_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_session_proto_goTypes = []interface{}{
	(SessionType)(0),           // 0: SessionType
	(SessionStatus)(0),         // 1: SessionStatus
	(*Session)(nil),            // 2: Session
	(*DefaultSessionData)(nil), // 3: DefaultSessionData
	(*ReshareSessionData)(nil), // 4: ReshareSessionData
	(*KeygenSessionData)(nil),  // 5: KeygenSessionData
	(*anypb.Any)(nil),          // 6: google.protobuf.Any
}
var file_session_proto_depIdxs = []int32{
	1, // 0: Session.status:type_name -> SessionStatus
	0, // 1: Session.type:type_name -> SessionType
	6, // 2: Session.data:type_name -> google.protobuf.Any
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_session_proto_init() }
func file_session_proto_init() {
	if File_session_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_session_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Session); i {
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
		file_session_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DefaultSessionData); i {
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
		file_session_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReshareSessionData); i {
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
		file_session_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KeygenSessionData); i {
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
			RawDescriptor: file_session_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_session_proto_goTypes,
		DependencyIndexes: file_session_proto_depIdxs,
		EnumInfos:         file_session_proto_enumTypes,
		MessageInfos:      file_session_proto_msgTypes,
	}.Build()
	File_session_proto = out.File
	file_session_proto_rawDesc = nil
	file_session_proto_goTypes = nil
	file_session_proto_depIdxs = nil
}
