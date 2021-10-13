// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: accesscontrol/policy.proto

package accesscontrol

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

// Policy used to describe how to authenticate a specific action
type Policy struct {
	// rule keywords, e.g., ANY/MAJORITY/ALL/SELF/a number/a rate
	Rule string `protobuf:"bytes,1,opt,name=rule,proto3" json:"rule,omitempty"`
	// org_list describes the organization set included in the authentication
	OrgList []string `protobuf:"bytes,2,rep,name=org_list,json=orgList,proto3" json:"org_list,omitempty"`
	// role_list describes the role set included in the authentication
	// e.g., admin/client/consensus/common
	RoleList []string `protobuf:"bytes,3,rep,name=role_list,json=roleList,proto3" json:"role_list,omitempty"`
}

func (m *Policy) Reset()         { *m = Policy{} }
func (m *Policy) String() string { return proto.CompactTextString(m) }
func (*Policy) ProtoMessage()    {}
func (*Policy) Descriptor() ([]byte, []int) {
	return fileDescriptor_cf8f77c591d51a3a, []int{0}
}
func (m *Policy) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Policy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Policy.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Policy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Policy.Merge(m, src)
}
func (m *Policy) XXX_Size() int {
	return m.Size()
}
func (m *Policy) XXX_DiscardUnknown() {
	xxx_messageInfo_Policy.DiscardUnknown(m)
}

var xxx_messageInfo_Policy proto.InternalMessageInfo

func (m *Policy) GetRule() string {
	if m != nil {
		return m.Rule
	}
	return ""
}

func (m *Policy) GetOrgList() []string {
	if m != nil {
		return m.OrgList
	}
	return nil
}

func (m *Policy) GetRoleList() []string {
	if m != nil {
		return m.RoleList
	}
	return nil
}

func init() {
	proto.RegisterType((*Policy)(nil), "accesscontrol.Policy")
}

func init() { proto.RegisterFile("accesscontrol/policy.proto", fileDescriptor_cf8f77c591d51a3a) }

var fileDescriptor_cf8f77c591d51a3a = []byte{
	// 201 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4a, 0x4c, 0x4e, 0x4e,
	0x2d, 0x2e, 0x4e, 0xce, 0xcf, 0x2b, 0x29, 0xca, 0xcf, 0xd1, 0x2f, 0xc8, 0xcf, 0xc9, 0x4c, 0xae,
	0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x45, 0x91, 0x53, 0x0a, 0xe1, 0x62, 0x0b, 0x00,
	0x4b, 0x0b, 0x09, 0x71, 0xb1, 0x14, 0x95, 0xe6, 0xa4, 0x4a, 0x30, 0x2a, 0x30, 0x6a, 0x70, 0x06,
	0x81, 0xd9, 0x42, 0x92, 0x5c, 0x1c, 0xf9, 0x45, 0xe9, 0xf1, 0x39, 0x99, 0xc5, 0x25, 0x12, 0x4c,
	0x0a, 0xcc, 0x1a, 0x9c, 0x41, 0xec, 0xf9, 0x45, 0xe9, 0x3e, 0x99, 0xc5, 0x25, 0x42, 0xd2, 0x5c,
	0x9c, 0x45, 0xf9, 0x39, 0xa9, 0x10, 0x39, 0x66, 0xb0, 0x1c, 0x07, 0x48, 0x00, 0x24, 0xe9, 0x94,
	0x7d, 0xe2, 0x91, 0x1c, 0xe3, 0x85, 0x47, 0x72, 0x8c, 0x0f, 0x1e, 0xc9, 0x31, 0x4e, 0x78, 0x2c,
	0xc7, 0x70, 0xe1, 0xb1, 0x1c, 0xc3, 0x8d, 0xc7, 0x72, 0x0c, 0x5c, 0xf2, 0xf9, 0x45, 0xe9, 0x7a,
	0xc9, 0x19, 0x89, 0x99, 0x79, 0xb9, 0x89, 0xd9, 0xa9, 0x45, 0x7a, 0x05, 0x49, 0x7a, 0x28, 0x0e,
	0x8a, 0x32, 0x40, 0x92, 0xcc, 0x2f, 0x4a, 0xd7, 0x47, 0x70, 0xf5, 0x0b, 0x92, 0x74, 0xd3, 0xf3,
	0xf5, 0xcb, 0x8c, 0xf4, 0x51, 0x74, 0x24, 0xb1, 0x81, 0x3d, 0x66, 0x0c, 0x08, 0x00, 0x00, 0xff,
	0xff, 0xe8, 0xca, 0x0e, 0x1e, 0xf6, 0x00, 0x00, 0x00,
}

func (m *Policy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Policy) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Policy) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.RoleList) > 0 {
		for iNdEx := len(m.RoleList) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.RoleList[iNdEx])
			copy(dAtA[i:], m.RoleList[iNdEx])
			i = encodeVarintPolicy(dAtA, i, uint64(len(m.RoleList[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.OrgList) > 0 {
		for iNdEx := len(m.OrgList) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.OrgList[iNdEx])
			copy(dAtA[i:], m.OrgList[iNdEx])
			i = encodeVarintPolicy(dAtA, i, uint64(len(m.OrgList[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Rule) > 0 {
		i -= len(m.Rule)
		copy(dAtA[i:], m.Rule)
		i = encodeVarintPolicy(dAtA, i, uint64(len(m.Rule)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintPolicy(dAtA []byte, offset int, v uint64) int {
	offset -= sovPolicy(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Policy) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Rule)
	if l > 0 {
		n += 1 + l + sovPolicy(uint64(l))
	}
	if len(m.OrgList) > 0 {
		for _, s := range m.OrgList {
			l = len(s)
			n += 1 + l + sovPolicy(uint64(l))
		}
	}
	if len(m.RoleList) > 0 {
		for _, s := range m.RoleList {
			l = len(s)
			n += 1 + l + sovPolicy(uint64(l))
		}
	}
	return n
}

func sovPolicy(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPolicy(x uint64) (n int) {
	return sovPolicy(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Policy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPolicy
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
			return fmt.Errorf("proto: Policy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Policy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rule", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicy
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
				return ErrInvalidLengthPolicy
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPolicy
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rule = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OrgList", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicy
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
				return ErrInvalidLengthPolicy
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPolicy
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OrgList = append(m.OrgList, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RoleList", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPolicy
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
				return ErrInvalidLengthPolicy
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPolicy
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RoleList = append(m.RoleList, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPolicy(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthPolicy
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
func skipPolicy(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPolicy
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
					return 0, ErrIntOverflowPolicy
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
					return 0, ErrIntOverflowPolicy
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
				return 0, ErrInvalidLengthPolicy
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupPolicy
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthPolicy
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthPolicy        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPolicy          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupPolicy = fmt.Errorf("proto: unexpected end of group")
)
