// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: request.proto

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

type Set struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Parties []string `protobuf:"bytes,1,rep,name=parties,proto3" json:"parties,omitempty"`
	N       uint32   `protobuf:"varint,2,opt,name=n,proto3" json:"n,omitempty"`
	T       uint32   `protobuf:"varint,3,opt,name=t,proto3" json:"t,omitempty"`
}

func (x *Set) Reset() {
	*x = Set{}
	if protoimpl.UnsafeEnabled {
		mi := &file_request_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Set) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Set) ProtoMessage() {}

func (x *Set) ProtoReflect() protoreflect.Message {
	mi := &file_request_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Set.ProtoReflect.Descriptor instead.
func (*Set) Descriptor() ([]byte, []int) {
	return file_request_proto_rawDescGZIP(), []int{0}
}

func (x *Set) GetParties() []string {
	if x != nil {
		return x.Parties
	}
	return nil
}

func (x *Set) GetN() uint32 {
	if x != nil {
		return x.N
	}
	return 0
}

func (x *Set) GetT() uint32 {
	if x != nil {
		return x.T
	}
	return 0
}

type DefaultSessionProposalData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Indexes []string `protobuf:"bytes,2,rep,name=indexes,proto3" json:"indexes,omitempty"`
	Root    string   `protobuf:"bytes,3,opt,name=root,proto3" json:"root,omitempty"`
}

func (x *DefaultSessionProposalData) Reset() {
	*x = DefaultSessionProposalData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_request_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DefaultSessionProposalData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DefaultSessionProposalData) ProtoMessage() {}

func (x *DefaultSessionProposalData) ProtoReflect() protoreflect.Message {
	mi := &file_request_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DefaultSessionProposalData.ProtoReflect.Descriptor instead.
func (*DefaultSessionProposalData) Descriptor() ([]byte, []int) {
	return file_request_proto_rawDescGZIP(), []int{1}
}

func (x *DefaultSessionProposalData) GetIndexes() []string {
	if x != nil {
		return x.Indexes
	}
	return nil
}

func (x *DefaultSessionProposalData) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

type ReshareSessionProposalData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Set *Set `protobuf:"bytes,1,opt,name=set,proto3" json:"set,omitempty"`
}

func (x *ReshareSessionProposalData) Reset() {
	*x = ReshareSessionProposalData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_request_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReshareSessionProposalData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReshareSessionProposalData) ProtoMessage() {}

func (x *ReshareSessionProposalData) ProtoReflect() protoreflect.Message {
	mi := &file_request_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReshareSessionProposalData.ProtoReflect.Descriptor instead.
func (*ReshareSessionProposalData) Descriptor() ([]byte, []int) {
	return file_request_proto_rawDescGZIP(), []int{2}
}

func (x *ReshareSessionProposalData) GetSet() *Set {
	if x != nil {
		return x.Set
	}
	return nil
}

type DefaultSessionAcceptanceData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Root string `protobuf:"bytes,1,opt,name=root,proto3" json:"root,omitempty"`
}

func (x *DefaultSessionAcceptanceData) Reset() {
	*x = DefaultSessionAcceptanceData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_request_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DefaultSessionAcceptanceData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DefaultSessionAcceptanceData) ProtoMessage() {}

func (x *DefaultSessionAcceptanceData) ProtoReflect() protoreflect.Message {
	mi := &file_request_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DefaultSessionAcceptanceData.ProtoReflect.Descriptor instead.
func (*DefaultSessionAcceptanceData) Descriptor() ([]byte, []int) {
	return file_request_proto_rawDescGZIP(), []int{3}
}

func (x *DefaultSessionAcceptanceData) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

type ReshareSessionAcceptanceData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	New *Set `protobuf:"bytes,1,opt,name=new,proto3" json:"new,omitempty"`
}

func (x *ReshareSessionAcceptanceData) Reset() {
	*x = ReshareSessionAcceptanceData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_request_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReshareSessionAcceptanceData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReshareSessionAcceptanceData) ProtoMessage() {}

func (x *ReshareSessionAcceptanceData) ProtoReflect() protoreflect.Message {
	mi := &file_request_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReshareSessionAcceptanceData.ProtoReflect.Descriptor instead.
func (*ReshareSessionAcceptanceData) Descriptor() ([]byte, []int) {
	return file_request_proto_rawDescGZIP(), []int{4}
}

func (x *ReshareSessionAcceptanceData) GetNew() *Set {
	if x != nil {
		return x.New
	}
	return nil
}

type SignRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data    string     `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Details *anypb.Any `protobuf:"bytes,2,opt,name=details,proto3" json:"details,omitempty"`
}

func (x *SignRequest) Reset() {
	*x = SignRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_request_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SignRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignRequest) ProtoMessage() {}

func (x *SignRequest) ProtoReflect() protoreflect.Message {
	mi := &file_request_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SignRequest.ProtoReflect.Descriptor instead.
func (*SignRequest) Descriptor() ([]byte, []int) {
	return file_request_proto_rawDescGZIP(), []int{5}
}

func (x *SignRequest) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

func (x *SignRequest) GetDetails() *anypb.Any {
	if x != nil {
		return x.Details
	}
	return nil
}

var File_request_proto protoreflect.FileDescriptor

var file_request_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0d, 0x73, 0x65, 0x73, 0x73,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3b, 0x0a, 0x03, 0x53, 0x65, 0x74,
	0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x07, 0x70, 0x61, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x0c, 0x0a, 0x01, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x01, 0x6e, 0x12, 0x0c, 0x0a, 0x01, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x01, 0x74, 0x22, 0x4a, 0x0a, 0x1a, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c,
	0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c,
	0x44, 0x61, 0x74, 0x61, 0x12, 0x18, 0x0a, 0x07, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f,
	0x6f, 0x74, 0x22, 0x34, 0x0a, 0x1a, 0x52, 0x65, 0x73, 0x68, 0x61, 0x72, 0x65, 0x53, 0x65, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x44, 0x61, 0x74, 0x61,
	0x12, 0x16, 0x0a, 0x03, 0x73, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x04, 0x2e,
	0x53, 0x65, 0x74, 0x52, 0x03, 0x73, 0x65, 0x74, 0x22, 0x32, 0x0a, 0x1c, 0x44, 0x65, 0x66, 0x61,
	0x75, 0x6c, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x41, 0x63, 0x63, 0x65, 0x70, 0x74,
	0x61, 0x6e, 0x63, 0x65, 0x44, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x22, 0x36, 0x0a, 0x1c,
	0x52, 0x65, 0x73, 0x68, 0x61, 0x72, 0x65, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x41, 0x63,
	0x63, 0x65, 0x70, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x44, 0x61, 0x74, 0x61, 0x12, 0x16, 0x0a, 0x03,
	0x6e, 0x65, 0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x04, 0x2e, 0x53, 0x65, 0x74, 0x52,
	0x03, 0x6e, 0x65, 0x77, 0x22, 0x51, 0x0a, 0x0b, 0x53, 0x69, 0x67, 0x6e, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x2e, 0x0a, 0x07, 0x64, 0x65, 0x74, 0x61, 0x69,
	0x6c, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x07,
	0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x42, 0x29, 0x5a, 0x27, 0x67, 0x69, 0x74, 0x6c, 0x61,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x72, 0x69, 0x6d, 0x6f, 0x2f, 0x74, 0x73, 0x73,
	0x2f, 0x74, 0x73, 0x73, 0x2d, 0x73, 0x76, 0x63, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_request_proto_rawDescOnce sync.Once
	file_request_proto_rawDescData = file_request_proto_rawDesc
)

func file_request_proto_rawDescGZIP() []byte {
	file_request_proto_rawDescOnce.Do(func() {
		file_request_proto_rawDescData = protoimpl.X.CompressGZIP(file_request_proto_rawDescData)
	})
	return file_request_proto_rawDescData
}

var file_request_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_request_proto_goTypes = []interface{}{
	(*Set)(nil),                          // 0: Set
	(*DefaultSessionProposalData)(nil),   // 1: DefaultSessionProposalData
	(*ReshareSessionProposalData)(nil),   // 2: ReshareSessionProposalData
	(*DefaultSessionAcceptanceData)(nil), // 3: DefaultSessionAcceptanceData
	(*ReshareSessionAcceptanceData)(nil), // 4: ReshareSessionAcceptanceData
	(*SignRequest)(nil),                  // 5: SignRequest
	(*anypb.Any)(nil),                    // 6: google.protobuf.Any
}
var file_request_proto_depIdxs = []int32{
	0, // 0: ReshareSessionProposalData.set:type_name -> Set
	0, // 1: ReshareSessionAcceptanceData.new:type_name -> Set
	6, // 2: SignRequest.details:type_name -> google.protobuf.Any
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_request_proto_init() }
func file_request_proto_init() {
	if File_request_proto != nil {
		return
	}
	file_session_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_request_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Set); i {
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
		file_request_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DefaultSessionProposalData); i {
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
		file_request_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReshareSessionProposalData); i {
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
		file_request_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DefaultSessionAcceptanceData); i {
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
		file_request_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReshareSessionAcceptanceData); i {
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
		file_request_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SignRequest); i {
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
			RawDescriptor: file_request_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_request_proto_goTypes,
		DependencyIndexes: file_request_proto_depIdxs,
		MessageInfos:      file_request_proto_msgTypes,
	}.Build()
	File_request_proto = out.File
	file_request_proto_rawDesc = nil
	file_request_proto_goTypes = nil
	file_request_proto_depIdxs = nil
}
