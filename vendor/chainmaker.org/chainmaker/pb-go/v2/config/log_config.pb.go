// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: config/log_config.proto

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

// request for log level
type LogLevelsRequest struct {
}

func (m *LogLevelsRequest) Reset()         { *m = LogLevelsRequest{} }
func (m *LogLevelsRequest) String() string { return proto.CompactTextString(m) }
func (*LogLevelsRequest) ProtoMessage()    {}
func (*LogLevelsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_46a0f8fbff6a5479, []int{0}
}
func (m *LogLevelsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LogLevelsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LogLevelsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LogLevelsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogLevelsRequest.Merge(m, src)
}
func (m *LogLevelsRequest) XXX_Size() int {
	return m.Size()
}
func (m *LogLevelsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_LogLevelsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_LogLevelsRequest proto.InternalMessageInfo

// response for log level
type LogLevelsResponse struct {
	// 0 success
	// 1 fail
	Code int32 `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	// failure message
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (m *LogLevelsResponse) Reset()         { *m = LogLevelsResponse{} }
func (m *LogLevelsResponse) String() string { return proto.CompactTextString(m) }
func (*LogLevelsResponse) ProtoMessage()    {}
func (*LogLevelsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_46a0f8fbff6a5479, []int{1}
}
func (m *LogLevelsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LogLevelsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LogLevelsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LogLevelsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogLevelsResponse.Merge(m, src)
}
func (m *LogLevelsResponse) XXX_Size() int {
	return m.Size()
}
func (m *LogLevelsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_LogLevelsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_LogLevelsResponse proto.InternalMessageInfo

func (m *LogLevelsResponse) GetCode() int32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *LogLevelsResponse) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterType((*LogLevelsRequest)(nil), "config.LogLevelsRequest")
	proto.RegisterType((*LogLevelsResponse)(nil), "config.LogLevelsResponse")
}

func init() { proto.RegisterFile("config/log_config.proto", fileDescriptor_46a0f8fbff6a5479) }

var fileDescriptor_46a0f8fbff6a5479 = []byte{
	// 194 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4f, 0xce, 0xcf, 0x4b,
	0xcb, 0x4c, 0xd7, 0xcf, 0xc9, 0x4f, 0x8f, 0x87, 0x30, 0xf5, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2, 0x85,
	0xd8, 0x20, 0x3c, 0x25, 0x21, 0x2e, 0x01, 0x9f, 0xfc, 0x74, 0x9f, 0xd4, 0xb2, 0xd4, 0x9c, 0xe2,
	0xa0, 0xd4, 0xc2, 0xd2, 0xd4, 0xe2, 0x12, 0x25, 0x47, 0x2e, 0x41, 0x24, 0xb1, 0xe2, 0x82, 0xfc,
	0xbc, 0xe2, 0x54, 0x21, 0x21, 0x2e, 0x96, 0xe4, 0xfc, 0x94, 0x54, 0x09, 0x46, 0x05, 0x46, 0x0d,
	0xd6, 0x20, 0x30, 0x5b, 0x48, 0x82, 0x8b, 0x3d, 0x37, 0xb5, 0xb8, 0x38, 0x31, 0x3d, 0x55, 0x82,
	0x49, 0x81, 0x51, 0x83, 0x33, 0x08, 0xc6, 0x75, 0x8a, 0x3d, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23,
	0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96, 0x63, 0xb8, 0xf0, 0x58, 0x8e, 0xe1, 0xc6,
	0x63, 0x39, 0x06, 0x2e, 0x89, 0xfc, 0xa2, 0x74, 0xbd, 0xe4, 0x8c, 0xc4, 0xcc, 0xbc, 0xdc, 0xc4,
	0xec, 0xd4, 0x22, 0xbd, 0x82, 0x24, 0x3d, 0x88, 0x53, 0xa2, 0x34, 0x91, 0x44, 0xf3, 0x8b, 0xd2,
	0xf5, 0x11, 0x5c, 0xfd, 0x82, 0x24, 0xdd, 0xf4, 0x7c, 0xfd, 0x32, 0x23, 0x7d, 0x88, 0xd2, 0x24,
	0x36, 0xb0, 0x27, 0x8c, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0xa2, 0xfb, 0x2f, 0x4d, 0xdf, 0x00,
	0x00, 0x00,
}

func (m *LogLevelsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LogLevelsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LogLevelsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *LogLevelsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LogLevelsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LogLevelsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Message) > 0 {
		i -= len(m.Message)
		copy(dAtA[i:], m.Message)
		i = encodeVarintLogConfig(dAtA, i, uint64(len(m.Message)))
		i--
		dAtA[i] = 0x12
	}
	if m.Code != 0 {
		i = encodeVarintLogConfig(dAtA, i, uint64(m.Code))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintLogConfig(dAtA []byte, offset int, v uint64) int {
	offset -= sovLogConfig(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *LogLevelsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *LogLevelsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Code != 0 {
		n += 1 + sovLogConfig(uint64(m.Code))
	}
	l = len(m.Message)
	if l > 0 {
		n += 1 + l + sovLogConfig(uint64(l))
	}
	return n
}

func sovLogConfig(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozLogConfig(x uint64) (n int) {
	return sovLogConfig(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *LogLevelsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLogConfig
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
			return fmt.Errorf("proto: LogLevelsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LogLevelsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipLogConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthLogConfig
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
func (m *LogLevelsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLogConfig
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
			return fmt.Errorf("proto: LogLevelsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LogLevelsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Code", wireType)
			}
			m.Code = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLogConfig
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
					return ErrIntOverflowLogConfig
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
				return ErrInvalidLengthLogConfig
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLogConfig
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Message = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLogConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthLogConfig
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
func skipLogConfig(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLogConfig
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
					return 0, ErrIntOverflowLogConfig
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
					return 0, ErrIntOverflowLogConfig
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
				return 0, ErrInvalidLengthLogConfig
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupLogConfig
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthLogConfig
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthLogConfig        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowLogConfig          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupLogConfig = fmt.Errorf("proto: unexpected end of group")
)
