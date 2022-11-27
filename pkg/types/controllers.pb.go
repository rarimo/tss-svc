// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: controllers.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/codec/types"
	proto "github.com/gogo/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type ControllerType int32

const (
	ControllerType_CONTROLLER_KEYGEN         ControllerType = 0
	ControllerType_CONTROLLER_PROPOSAL       ControllerType = 1
	ControllerType_CONTROLLER_RESHARE        ControllerType = 2
	ControllerType_CONTROLLER_ACCEPTANCE     ControllerType = 3
	ControllerType_CONTROLLER_SIGN           ControllerType = 5
	ControllerType_CONTROLLER_FINISH_DEFAULT ControllerType = 6
	ControllerType_CONTROLLER_FINISH_RESHARE ControllerType = 7
)

var ControllerType_name = map[int32]string{
	0: "CONTROLLER_KEYGEN",
	1: "CONTROLLER_PROPOSAL",
	2: "CONTROLLER_RESHARE",
	3: "CONTROLLER_ACCEPTANCE",
	5: "CONTROLLER_SIGN",
	6: "CONTROLLER_FINISH_DEFAULT",
	7: "CONTROLLER_FINISH_RESHARE",
}

var ControllerType_value = map[string]int32{
	"CONTROLLER_KEYGEN":         0,
	"CONTROLLER_PROPOSAL":       1,
	"CONTROLLER_RESHARE":        2,
	"CONTROLLER_ACCEPTANCE":     3,
	"CONTROLLER_SIGN":           5,
	"CONTROLLER_FINISH_DEFAULT": 6,
	"CONTROLLER_FINISH_RESHARE": 7,
}

func (x ControllerType) String() string {
	return proto.EnumName(ControllerType_name, int32(x))
}

func (ControllerType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_fad12f42efe8f126, []int{0}
}

func init() {
	proto.RegisterEnum("ControllerType", ControllerType_name, ControllerType_value)
}

func init() { proto.RegisterFile("controllers.proto", fileDescriptor_fad12f42efe8f126) }

var fileDescriptor_fad12f42efe8f126 = []byte{
	// 262 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4c, 0xce, 0xcf, 0x2b,
	0x29, 0xca, 0xcf, 0xc9, 0x49, 0x2d, 0x2a, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x97, 0x92, 0x4c,
	0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x07, 0xf3, 0x92, 0x4a, 0xd3, 0xf4, 0x13, 0xf3, 0x2a, 0x21,
	0x52, 0x5a, 0xc7, 0x18, 0xb9, 0xf8, 0x9c, 0xe1, 0x1a, 0x42, 0x2a, 0x0b, 0x52, 0x85, 0x44, 0xb9,
	0x04, 0x9d, 0xfd, 0xfd, 0x42, 0x82, 0xfc, 0x7d, 0x7c, 0x5c, 0x83, 0xe2, 0xbd, 0x5d, 0x23, 0xdd,
	0x5d, 0xfd, 0x04, 0x18, 0x84, 0xc4, 0xb9, 0x84, 0x91, 0x84, 0x03, 0x82, 0xfc, 0x03, 0xfc, 0x83,
	0x1d, 0x7d, 0x04, 0x18, 0x85, 0xc4, 0xb8, 0x84, 0x90, 0x24, 0x82, 0x5c, 0x83, 0x3d, 0x1c, 0x83,
	0x5c, 0x05, 0x98, 0x84, 0x24, 0xb9, 0x44, 0x91, 0xc4, 0x1d, 0x9d, 0x9d, 0x5d, 0x03, 0x42, 0x1c,
	0xfd, 0x9c, 0x5d, 0x05, 0x98, 0x85, 0x84, 0xb9, 0xf8, 0x91, 0xa4, 0x82, 0x3d, 0xdd, 0xfd, 0x04,
	0x58, 0x85, 0x64, 0xb9, 0x24, 0x91, 0x04, 0xdd, 0x3c, 0xfd, 0x3c, 0x83, 0x3d, 0xe2, 0x5d, 0x5c,
	0xdd, 0x1c, 0x43, 0x7d, 0x42, 0x04, 0xd8, 0xb0, 0x4b, 0xc3, 0x6c, 0x63, 0x77, 0x72, 0x3b, 0xf1,
	0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96, 0x63, 0xb8,
	0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x86, 0x28, 0x9d, 0xf4, 0xcc, 0x92, 0x9c, 0xc4, 0x24,
	0xbd, 0xe4, 0xfc, 0x5c, 0xfd, 0xa2, 0xc4, 0xa2, 0xcc, 0xb4, 0x4a, 0x5d, 0xb0, 0xdf, 0x93, 0xf3,
	0x73, 0xf4, 0x4b, 0x8a, 0x8b, 0x75, 0x8b, 0xcb, 0x92, 0xf5, 0x0b, 0xb2, 0xd3, 0xf5, 0x4b, 0x2a,
	0x0b, 0x52, 0x8b, 0x93, 0xd8, 0xc0, 0x72, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0xae, 0xe5,
	0xed, 0xf3, 0x47, 0x01, 0x00, 0x00,
}
