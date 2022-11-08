// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: request.proto

package types

import (
	fmt "fmt"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
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

type RequestType int32

const (
	RequestType_Proposal   RequestType = 0
	RequestType_Acceptance RequestType = 1
	RequestType_Sign       RequestType = 2
	RequestType_Reshare    RequestType = 3
	RequestType_Keygen     RequestType = 4
)

var RequestType_name = map[int32]string{
	0: "Proposal",
	1: "Acceptance",
	2: "Sign",
	3: "Reshare",
	4: "Keygen",
}

var RequestType_value = map[string]int32{
	"Proposal":   0,
	"Acceptance": 1,
	"Sign":       2,
	"Reshare":    3,
	"Keygen":     4,
}

func (x RequestType) String() string {
	return proto.EnumName(RequestType_name, int32(x))
}

func (RequestType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_7f73548e33e655fe, []int{0}
}

type ProposalRequest struct {
	Session uint64 `protobuf:"varint,1,opt,name=session,proto3" json:"session,omitempty"`
	// List of operation ids
	Indexes []string `protobuf:"bytes,2,rep,name=indexes,proto3" json:"indexes,omitempty"`
	// Merkle tree based on operations root hash in hex
	Root string `protobuf:"bytes,3,opt,name=root,proto3" json:"root,omitempty"`
}

func (m *ProposalRequest) Reset()         { *m = ProposalRequest{} }
func (m *ProposalRequest) String() string { return proto.CompactTextString(m) }
func (*ProposalRequest) ProtoMessage()    {}
func (*ProposalRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_7f73548e33e655fe, []int{0}
}
func (m *ProposalRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ProposalRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ProposalRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ProposalRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProposalRequest.Merge(m, src)
}
func (m *ProposalRequest) XXX_Size() int {
	return m.Size()
}
func (m *ProposalRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ProposalRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ProposalRequest proto.InternalMessageInfo

func (m *ProposalRequest) GetSession() uint64 {
	if m != nil {
		return m.Session
	}
	return 0
}

func (m *ProposalRequest) GetIndexes() []string {
	if m != nil {
		return m.Indexes
	}
	return nil
}

func (m *ProposalRequest) GetRoot() string {
	if m != nil {
		return m.Root
	}
	return ""
}

type AcceptanceRequest struct {
	// Merkle tree based on operations root hash in hex
	Root string `protobuf:"bytes,1,opt,name=root,proto3" json:"root,omitempty"`
}

func (m *AcceptanceRequest) Reset()         { *m = AcceptanceRequest{} }
func (m *AcceptanceRequest) String() string { return proto.CompactTextString(m) }
func (*AcceptanceRequest) ProtoMessage()    {}
func (*AcceptanceRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_7f73548e33e655fe, []int{1}
}
func (m *AcceptanceRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AcceptanceRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AcceptanceRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AcceptanceRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AcceptanceRequest.Merge(m, src)
}
func (m *AcceptanceRequest) XXX_Size() int {
	return m.Size()
}
func (m *AcceptanceRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AcceptanceRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AcceptanceRequest proto.InternalMessageInfo

func (m *AcceptanceRequest) GetRoot() string {
	if m != nil {
		return m.Root
	}
	return ""
}

type SignRequest struct {
	Root    string     `protobuf:"bytes,1,opt,name=root,proto3" json:"root,omitempty"`
	Details *types.Any `protobuf:"bytes,2,opt,name=details,proto3" json:"details,omitempty"`
}

func (m *SignRequest) Reset()         { *m = SignRequest{} }
func (m *SignRequest) String() string { return proto.CompactTextString(m) }
func (*SignRequest) ProtoMessage()    {}
func (*SignRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_7f73548e33e655fe, []int{2}
}
func (m *SignRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SignRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SignRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SignRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignRequest.Merge(m, src)
}
func (m *SignRequest) XXX_Size() int {
	return m.Size()
}
func (m *SignRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SignRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SignRequest proto.InternalMessageInfo

func (m *SignRequest) GetRoot() string {
	if m != nil {
		return m.Root
	}
	return ""
}

func (m *SignRequest) GetDetails() *types.Any {
	if m != nil {
		return m.Details
	}
	return nil
}

func init() {
	proto.RegisterEnum("RequestType", RequestType_name, RequestType_value)
	proto.RegisterType((*ProposalRequest)(nil), "ProposalRequest")
	proto.RegisterType((*AcceptanceRequest)(nil), "AcceptanceRequest")
	proto.RegisterType((*SignRequest)(nil), "SignRequest")
}

func init() { proto.RegisterFile("request.proto", fileDescriptor_7f73548e33e655fe) }

var fileDescriptor_7f73548e33e655fe = []byte{
	// 310 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x8f, 0xbd, 0x4e, 0xc3, 0x30,
	0x14, 0x85, 0xe3, 0x36, 0xea, 0xcf, 0x0d, 0x3f, 0xc1, 0x62, 0x08, 0x0c, 0x51, 0xd5, 0x85, 0x08,
	0xd1, 0x44, 0x82, 0x27, 0x28, 0x03, 0x0b, 0x12, 0x82, 0xc0, 0x02, 0x9b, 0x9b, 0xde, 0x86, 0x88,
	0x60, 0x1b, 0xdb, 0x45, 0xf8, 0x2d, 0x78, 0x2c, 0xc6, 0x8e, 0x8c, 0xa8, 0x7d, 0x11, 0xd4, 0x14,
	0x0b, 0x26, 0x36, 0x1f, 0x9d, 0xcf, 0xf7, 0x9c, 0x03, 0xdb, 0x0a, 0x5f, 0xe6, 0xa8, 0x4d, 0x2a,
	0x95, 0x30, 0xe2, 0xf0, 0xa0, 0x14, 0xa2, 0xac, 0x31, 0x6b, 0xd4, 0x64, 0x3e, 0xcb, 0x18, 0xb7,
	0x1b, 0x6b, 0x78, 0x0f, 0xbb, 0xd7, 0x4a, 0x48, 0xa1, 0x59, 0x9d, 0x6f, 0xfe, 0xd0, 0x08, 0xba,
	0x1a, 0xb5, 0xae, 0x04, 0x8f, 0xc8, 0x80, 0x24, 0x7e, 0xee, 0xe4, 0xda, 0xa9, 0xf8, 0x14, 0xdf,
	0x50, 0x47, 0xad, 0x41, 0x3b, 0xe9, 0xe7, 0x4e, 0x52, 0x0a, 0xbe, 0x12, 0xc2, 0x44, 0xed, 0x01,
	0x49, 0xfa, 0x79, 0xf3, 0x1e, 0x1e, 0xc1, 0xde, 0xb8, 0x28, 0x50, 0x1a, 0xc6, 0x0b, 0x74, 0xc7,
	0x1d, 0x48, 0xfe, 0x80, 0x37, 0x10, 0xdc, 0x56, 0x25, 0xff, 0x07, 0xa1, 0x29, 0x74, 0xa7, 0x68,
	0x58, 0x55, 0xaf, 0x93, 0x49, 0x12, 0x9c, 0xee, 0xa7, 0x9b, 0x4d, 0xa9, 0xdb, 0x94, 0x8e, 0xb9,
	0xcd, 0x1d, 0x74, 0x7c, 0x05, 0xc1, 0xcf, 0xb9, 0x3b, 0x2b, 0x91, 0x6e, 0x41, 0xcf, 0xad, 0x0c,
	0x3d, 0xba, 0x03, 0xf0, 0x5b, 0x2c, 0x24, 0xb4, 0x07, 0xfe, 0x3a, 0x3f, 0x6c, 0xd1, 0x00, 0xba,
	0x39, 0xea, 0x47, 0xa6, 0x30, 0x6c, 0x53, 0x80, 0xce, 0x25, 0xda, 0x12, 0x79, 0xe8, 0x9f, 0x5f,
	0x7c, 0x2c, 0x63, 0xb2, 0x58, 0xc6, 0xe4, 0x6b, 0x19, 0x93, 0xf7, 0x55, 0xec, 0x2d, 0x56, 0xb1,
	0xf7, 0xb9, 0x8a, 0xbd, 0x87, 0x93, 0xb2, 0x32, 0x35, 0x9b, 0xa4, 0x85, 0x78, 0xce, 0x14, 0x53,
	0xd5, 0xcc, 0x8e, 0x9a, 0x4a, 0x85, 0xa8, 0x33, 0xa3, 0xf5, 0x48, 0xbf, 0x16, 0x99, 0x7c, 0x2a,
	0x33, 0x63, 0x25, 0xea, 0x49, 0xa7, 0xf1, 0xce, 0xbe, 0x03, 0x00, 0x00, 0xff, 0xff, 0x0d, 0xb5,
	0xf5, 0x8b, 0xa1, 0x01, 0x00, 0x00,
}

func (m *ProposalRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ProposalRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ProposalRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Root) > 0 {
		i -= len(m.Root)
		copy(dAtA[i:], m.Root)
		i = encodeVarintRequest(dAtA, i, uint64(len(m.Root)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Indexes) > 0 {
		for iNdEx := len(m.Indexes) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Indexes[iNdEx])
			copy(dAtA[i:], m.Indexes[iNdEx])
			i = encodeVarintRequest(dAtA, i, uint64(len(m.Indexes[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if m.Session != 0 {
		i = encodeVarintRequest(dAtA, i, uint64(m.Session))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *AcceptanceRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AcceptanceRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AcceptanceRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Root) > 0 {
		i -= len(m.Root)
		copy(dAtA[i:], m.Root)
		i = encodeVarintRequest(dAtA, i, uint64(len(m.Root)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *SignRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SignRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Details != nil {
		{
			size, err := m.Details.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintRequest(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Root) > 0 {
		i -= len(m.Root)
		copy(dAtA[i:], m.Root)
		i = encodeVarintRequest(dAtA, i, uint64(len(m.Root)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintRequest(dAtA []byte, offset int, v uint64) int {
	offset -= sovRequest(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ProposalRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Session != 0 {
		n += 1 + sovRequest(uint64(m.Session))
	}
	if len(m.Indexes) > 0 {
		for _, s := range m.Indexes {
			l = len(s)
			n += 1 + l + sovRequest(uint64(l))
		}
	}
	l = len(m.Root)
	if l > 0 {
		n += 1 + l + sovRequest(uint64(l))
	}
	return n
}

func (m *AcceptanceRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Root)
	if l > 0 {
		n += 1 + l + sovRequest(uint64(l))
	}
	return n
}

func (m *SignRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Root)
	if l > 0 {
		n += 1 + l + sovRequest(uint64(l))
	}
	if m.Details != nil {
		l = m.Details.Size()
		n += 1 + l + sovRequest(uint64(l))
	}
	return n
}

func sovRequest(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozRequest(x uint64) (n int) {
	return sovRequest(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ProposalRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRequest
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ProposalRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProposalRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Session", wireType)
			}
			m.Session = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Session |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Indexes", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthRequest
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRequest
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Indexes = append(m.Indexes, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Root", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthRequest
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRequest
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Root = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRequest(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRequest
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *AcceptanceRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRequest
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AcceptanceRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AcceptanceRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Root", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthRequest
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRequest
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Root = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRequest(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRequest
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *SignRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRequest
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: SignRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SignRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Root", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthRequest
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRequest
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Root = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Details", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthRequest
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthRequest
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Details == nil {
				m.Details = &types.Any{}
			}
			if err := m.Details.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRequest(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRequest
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipRequest(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRequest
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRequest
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthRequest
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupRequest
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthRequest
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthRequest        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRequest          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupRequest = fmt.Errorf("proto: unexpected end of group")
)
