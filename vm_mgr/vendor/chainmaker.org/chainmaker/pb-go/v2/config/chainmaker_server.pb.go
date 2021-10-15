// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: config/chainmaker_server.proto

package config

import (
	fmt "fmt"
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

// Request for chainmaker version
type ChainMakerVersionRequest struct {
}

func (m *ChainMakerVersionRequest) Reset()         { *m = ChainMakerVersionRequest{} }
func (m *ChainMakerVersionRequest) String() string { return proto.CompactTextString(m) }
func (*ChainMakerVersionRequest) ProtoMessage()    {}
func (*ChainMakerVersionRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dfdb570eb1e057d, []int{0}
}
func (m *ChainMakerVersionRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ChainMakerVersionRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ChainMakerVersionRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ChainMakerVersionRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChainMakerVersionRequest.Merge(m, src)
}
func (m *ChainMakerVersionRequest) XXX_Size() int {
	return m.Size()
}
func (m *ChainMakerVersionRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ChainMakerVersionRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ChainMakerVersionRequest proto.InternalMessageInfo

// Response for chainmaker version
type ChainMakerVersionResponse struct {
	// 0 success
	// 1 fail
	Code    int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
}

func (m *ChainMakerVersionResponse) Reset()         { *m = ChainMakerVersionResponse{} }
func (m *ChainMakerVersionResponse) String() string { return proto.CompactTextString(m) }
func (*ChainMakerVersionResponse) ProtoMessage()    {}
func (*ChainMakerVersionResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dfdb570eb1e057d, []int{1}
}
func (m *ChainMakerVersionResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ChainMakerVersionResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ChainMakerVersionResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ChainMakerVersionResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChainMakerVersionResponse.Merge(m, src)
}
func (m *ChainMakerVersionResponse) XXX_Size() int {
	return m.Size()
}
func (m *ChainMakerVersionResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ChainMakerVersionResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ChainMakerVersionResponse proto.InternalMessageInfo

func (m *ChainMakerVersionResponse) GetCode() int32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *ChainMakerVersionResponse) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *ChainMakerVersionResponse) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

func init() {
	proto.RegisterType((*ChainMakerVersionRequest)(nil), "config.ChainMakerVersionRequest")
	proto.RegisterType((*ChainMakerVersionResponse)(nil), "config.ChainMakerVersionResponse")
}

func init() { proto.RegisterFile("config/chainmaker_server.proto", fileDescriptor_8dfdb570eb1e057d) }

var fileDescriptor_8dfdb570eb1e057d = []byte{
	// 211 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4b, 0xce, 0xcf, 0x4b,
	0xcb, 0x4c, 0xd7, 0x4f, 0xce, 0x48, 0xcc, 0xcc, 0xcb, 0x4d, 0xcc, 0x4e, 0x2d, 0x8a, 0x2f, 0x4e,
	0x2d, 0x2a, 0x4b, 0x2d, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x83, 0xc8, 0x2b, 0x49,
	0x71, 0x49, 0x38, 0x83, 0x94, 0xf8, 0x82, 0x94, 0x84, 0xa5, 0x16, 0x15, 0x67, 0xe6, 0xe7, 0x05,
	0xa5, 0x16, 0x96, 0xa6, 0x16, 0x97, 0x28, 0x25, 0x73, 0x49, 0x62, 0x91, 0x2b, 0x2e, 0xc8, 0xcf,
	0x2b, 0x4e, 0x15, 0x12, 0xe2, 0x62, 0x49, 0xce, 0x4f, 0x49, 0x95, 0x60, 0x54, 0x60, 0xd4, 0x60,
	0x0d, 0x02, 0xb3, 0x85, 0x24, 0xb8, 0xd8, 0x73, 0x53, 0x8b, 0x8b, 0x13, 0xd3, 0x53, 0x25, 0x98,
	0x14, 0x18, 0x35, 0x38, 0x83, 0x60, 0x5c, 0x90, 0x4c, 0x19, 0xc4, 0x00, 0x09, 0x66, 0x88, 0x0c,
	0x94, 0xeb, 0x14, 0x7b, 0xe2, 0x91, 0x1c, 0xe3, 0x85, 0x47, 0x72, 0x8c, 0x0f, 0x1e, 0xc9, 0x31,
	0x4e, 0x78, 0x2c, 0xc7, 0x70, 0xe1, 0xb1, 0x1c, 0xc3, 0x8d, 0xc7, 0x72, 0x0c, 0x5c, 0x12, 0xf9,
	0x45, 0xe9, 0x7a, 0x08, 0xf7, 0xeb, 0x15, 0x24, 0xe9, 0x41, 0x1c, 0x1d, 0xa5, 0x89, 0x24, 0x9a,
	0x5f, 0x84, 0xec, 0x49, 0xfd, 0x82, 0x24, 0xdd, 0xf4, 0x7c, 0xfd, 0x32, 0x23, 0x7d, 0x88, 0xd2,
	0x24, 0x36, 0xb0, 0x77, 0x8d, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x4b, 0x8a, 0xfd, 0xee, 0x10,
	0x01, 0x00, 0x00,
}

func (m *ChainMakerVersionRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ChainMakerVersionRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ChainMakerVersionRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *ChainMakerVersionResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ChainMakerVersionResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ChainMakerVersionResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Version) > 0 {
		i -= len(m.Version)
		copy(dAtA[i:], m.Version)
		i = encodeVarintChainmakerServer(dAtA, i, uint64(len(m.Version)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Message) > 0 {
		i -= len(m.Message)
		copy(dAtA[i:], m.Message)
		i = encodeVarintChainmakerServer(dAtA, i, uint64(len(m.Message)))
		i--
		dAtA[i] = 0x12
	}
	if m.Code != 0 {
		i = encodeVarintChainmakerServer(dAtA, i, uint64(m.Code))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintChainmakerServer(dAtA []byte, offset int, v uint64) int {
	offset -= sovChainmakerServer(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ChainMakerVersionRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *ChainMakerVersionResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Code != 0 {
		n += 1 + sovChainmakerServer(uint64(m.Code))
	}
	l = len(m.Message)
	if l > 0 {
		n += 1 + l + sovChainmakerServer(uint64(l))
	}
	l = len(m.Version)
	if l > 0 {
		n += 1 + l + sovChainmakerServer(uint64(l))
	}
	return n
}

func sovChainmakerServer(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozChainmakerServer(x uint64) (n int) {
	return sovChainmakerServer(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ChainMakerVersionRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowChainmakerServer
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
			return fmt.Errorf("proto: ChainMakerVersionRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ChainMakerVersionRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipChainmakerServer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthChainmakerServer
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
func (m *ChainMakerVersionResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowChainmakerServer
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
			return fmt.Errorf("proto: ChainMakerVersionResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ChainMakerVersionResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Code", wireType)
			}
			m.Code = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChainmakerServer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Code |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Message", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChainmakerServer
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
				return ErrInvalidLengthChainmakerServer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthChainmakerServer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Message = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChainmakerServer
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
				return ErrInvalidLengthChainmakerServer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthChainmakerServer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Version = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipChainmakerServer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthChainmakerServer
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
func skipChainmakerServer(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowChainmakerServer
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
					return 0, ErrIntOverflowChainmakerServer
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
					return 0, ErrIntOverflowChainmakerServer
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
				return 0, ErrInvalidLengthChainmakerServer
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupChainmakerServer
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthChainmakerServer
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthChainmakerServer        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowChainmakerServer          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupChainmakerServer = fmt.Errorf("proto: unexpected end of group")
)
