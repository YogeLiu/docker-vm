// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: net/net.proto

package net

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

// specific net message types
type NetMsg_MsgType int32

const (
	NetMsg_INVALID_MSG    NetMsg_MsgType = 0
	NetMsg_TX             NetMsg_MsgType = 1
	NetMsg_TXS            NetMsg_MsgType = 2
	NetMsg_BLOCK          NetMsg_MsgType = 3
	NetMsg_BLOCKS         NetMsg_MsgType = 4
	NetMsg_CONSENSUS_MSG  NetMsg_MsgType = 5
	NetMsg_SYNC_BLOCK_MSG NetMsg_MsgType = 6
)

var NetMsg_MsgType_name = map[int32]string{
	0: "INVALID_MSG",
	1: "TX",
	2: "TXS",
	3: "BLOCK",
	4: "BLOCKS",
	5: "CONSENSUS_MSG",
	6: "SYNC_BLOCK_MSG",
}

var NetMsg_MsgType_value = map[string]int32{
	"INVALID_MSG":    0,
	"TX":             1,
	"TXS":            2,
	"BLOCK":          3,
	"BLOCKS":         4,
	"CONSENSUS_MSG":  5,
	"SYNC_BLOCK_MSG": 6,
}

func (x NetMsg_MsgType) String() string {
	return proto.EnumName(NetMsg_MsgType_name, int32(x))
}

func (NetMsg_MsgType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8b79b5a127a76ba1, []int{1, 0}
}

// wrapped network message
type Msg struct {
	Msg     *NetMsg `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	ChainId string  `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	// 属于那个模块，判断消息类型
	Flag string `protobuf:"bytes,3,opt,name=flag,proto3" json:"flag,omitempty"`
}

func (m *Msg) Reset()         { *m = Msg{} }
func (m *Msg) String() string { return proto.CompactTextString(m) }
func (*Msg) ProtoMessage()    {}
func (*Msg) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b79b5a127a76ba1, []int{0}
}
func (m *Msg) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Msg) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Msg.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Msg) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Msg.Merge(m, src)
}
func (m *Msg) XXX_Size() int {
	return m.Size()
}
func (m *Msg) XXX_DiscardUnknown() {
	xxx_messageInfo_Msg.DiscardUnknown(m)
}

var xxx_messageInfo_Msg proto.InternalMessageInfo

func (m *Msg) GetMsg() *NetMsg {
	if m != nil {
		return m.Msg
	}
	return nil
}

func (m *Msg) GetChainId() string {
	if m != nil {
		return m.ChainId
	}
	return ""
}

func (m *Msg) GetFlag() string {
	if m != nil {
		return m.Flag
	}
	return ""
}

// net message
type NetMsg struct {
	// payload of the message
	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	// message type
	Type NetMsg_MsgType `protobuf:"varint,2,opt,name=type,proto3,enum=net.NetMsg_MsgType" json:"type,omitempty"`
	// nodeId
	To string `protobuf:"bytes,3,opt,name=to,proto3" json:"to,omitempty"`
}

func (m *NetMsg) Reset()         { *m = NetMsg{} }
func (m *NetMsg) String() string { return proto.CompactTextString(m) }
func (*NetMsg) ProtoMessage()    {}
func (*NetMsg) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b79b5a127a76ba1, []int{1}
}
func (m *NetMsg) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NetMsg) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NetMsg.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NetMsg) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NetMsg.Merge(m, src)
}
func (m *NetMsg) XXX_Size() int {
	return m.Size()
}
func (m *NetMsg) XXX_DiscardUnknown() {
	xxx_messageInfo_NetMsg.DiscardUnknown(m)
}

var xxx_messageInfo_NetMsg proto.InternalMessageInfo

func (m *NetMsg) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *NetMsg) GetType() NetMsg_MsgType {
	if m != nil {
		return m.Type
	}
	return NetMsg_INVALID_MSG
}

func (m *NetMsg) GetTo() string {
	if m != nil {
		return m.To
	}
	return ""
}

func init() {
	proto.RegisterEnum("net.NetMsg_MsgType", NetMsg_MsgType_name, NetMsg_MsgType_value)
	proto.RegisterType((*Msg)(nil), "net.Msg")
	proto.RegisterType((*NetMsg)(nil), "net.NetMsg")
}

func init() { proto.RegisterFile("net/net.proto", fileDescriptor_8b79b5a127a76ba1) }

var fileDescriptor_8b79b5a127a76ba1 = []byte{
	// 331 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x91, 0xc1, 0x4e, 0xea, 0x40,
	0x14, 0x86, 0x3b, 0x2d, 0xb4, 0x97, 0xc3, 0x85, 0xdb, 0x7b, 0x8c, 0x49, 0x5d, 0xd8, 0x10, 0x16,
	0xca, 0xc6, 0x36, 0xc1, 0x27, 0x10, 0x34, 0x86, 0x48, 0x4b, 0xd2, 0x41, 0x45, 0x37, 0xa4, 0xc8,
	0x38, 0x12, 0xa1, 0xd3, 0xb4, 0x13, 0x13, 0xde, 0xc2, 0xa7, 0x32, 0x2e, 0x59, 0xba, 0x34, 0xf0,
	0x22, 0xa6, 0x23, 0x46, 0x76, 0xe7, 0xff, 0xf3, 0x9d, 0x7f, 0x32, 0xe7, 0x87, 0x5a, 0xc2, 0xa4,
	0x9f, 0x30, 0xe9, 0xa5, 0x99, 0x90, 0x02, 0x8d, 0x84, 0xc9, 0x26, 0x05, 0x23, 0xc8, 0x39, 0x1e,
	0x82, 0xb1, 0xc8, 0xb9, 0x43, 0x1a, 0xa4, 0x55, 0x6d, 0x57, 0xbd, 0x02, 0x0a, 0x99, 0x0c, 0x72,
	0x1e, 0x15, 0x3e, 0x1e, 0xc0, 0x9f, 0x87, 0xa7, 0x78, 0x96, 0x8c, 0x67, 0x53, 0x47, 0x6f, 0x90,
	0x56, 0x25, 0xb2, 0x94, 0xee, 0x4d, 0x11, 0xa1, 0xf4, 0x38, 0x8f, 0xb9, 0x63, 0x28, 0x5b, 0xcd,
	0xcd, 0x37, 0x02, 0xe6, 0xf7, 0x3a, 0x3a, 0x60, 0xa5, 0xf1, 0x72, 0x2e, 0xe2, 0xa9, 0x0a, 0xff,
	0x1b, 0xfd, 0x48, 0x3c, 0x86, 0x92, 0x5c, 0xa6, 0x4c, 0xe5, 0xd5, 0xdb, 0x7b, 0x3b, 0x6f, 0x7a,
	0x41, 0xce, 0x87, 0xcb, 0x94, 0x45, 0x0a, 0xc0, 0x3a, 0xe8, 0x52, 0x6c, 0xf3, 0x75, 0x29, 0x9a,
	0x33, 0xb0, 0xb6, 0x00, 0xfe, 0x83, 0x6a, 0x2f, 0xbc, 0x39, 0xeb, 0xf7, 0xce, 0xc7, 0x01, 0xbd,
	0xb4, 0x35, 0x34, 0x41, 0x1f, 0x8e, 0x6c, 0x82, 0x16, 0x18, 0xc3, 0x11, 0xb5, 0x75, 0xac, 0x40,
	0xb9, 0xd3, 0x1f, 0x74, 0xaf, 0x6c, 0x03, 0x01, 0x4c, 0x35, 0x52, 0xbb, 0x84, 0xff, 0xa1, 0xd6,
	0x1d, 0x84, 0xf4, 0x22, 0xa4, 0xd7, 0x54, 0xad, 0x96, 0x11, 0xa1, 0x4e, 0xef, 0xc2, 0xee, 0x58,
	0x31, 0xca, 0x33, 0x3b, 0xb7, 0xef, 0x6b, 0x97, 0xac, 0xd6, 0x2e, 0xf9, 0x5c, 0xbb, 0xe4, 0x75,
	0xe3, 0x6a, 0xab, 0x8d, 0xab, 0x7d, 0x6c, 0x5c, 0x0d, 0xf6, 0x45, 0xc6, 0x3d, 0x75, 0x83, 0x45,
	0xfc, 0xcc, 0x32, 0x2f, 0x9d, 0x14, 0x1f, 0xb8, 0x3f, 0xda, 0xb1, 0x44, 0xc6, 0xfd, 0x5f, 0xe9,
	0xa7, 0x93, 0x13, 0x2e, 0xfc, 0x97, 0x76, 0xd1, 0xc0, 0xc4, 0x54, 0x15, 0x9c, 0x7e, 0x05, 0x00,
	0x00, 0xff, 0xff, 0xdb, 0xc3, 0xd4, 0xe3, 0x93, 0x01, 0x00, 0x00,
}

func (m *Msg) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Msg) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Msg) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Flag) > 0 {
		i -= len(m.Flag)
		copy(dAtA[i:], m.Flag)
		i = encodeVarintNet(dAtA, i, uint64(len(m.Flag)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ChainId) > 0 {
		i -= len(m.ChainId)
		copy(dAtA[i:], m.ChainId)
		i = encodeVarintNet(dAtA, i, uint64(len(m.ChainId)))
		i--
		dAtA[i] = 0x12
	}
	if m.Msg != nil {
		{
			size, err := m.Msg.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintNet(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *NetMsg) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NetMsg) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NetMsg) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.To) > 0 {
		i -= len(m.To)
		copy(dAtA[i:], m.To)
		i = encodeVarintNet(dAtA, i, uint64(len(m.To)))
		i--
		dAtA[i] = 0x1a
	}
	if m.Type != 0 {
		i = encodeVarintNet(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Payload) > 0 {
		i -= len(m.Payload)
		copy(dAtA[i:], m.Payload)
		i = encodeVarintNet(dAtA, i, uint64(len(m.Payload)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintNet(dAtA []byte, offset int, v uint64) int {
	offset -= sovNet(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Msg) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Msg != nil {
		l = m.Msg.Size()
		n += 1 + l + sovNet(uint64(l))
	}
	l = len(m.ChainId)
	if l > 0 {
		n += 1 + l + sovNet(uint64(l))
	}
	l = len(m.Flag)
	if l > 0 {
		n += 1 + l + sovNet(uint64(l))
	}
	return n
}

func (m *NetMsg) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Payload)
	if l > 0 {
		n += 1 + l + sovNet(uint64(l))
	}
	if m.Type != 0 {
		n += 1 + sovNet(uint64(m.Type))
	}
	l = len(m.To)
	if l > 0 {
		n += 1 + l + sovNet(uint64(l))
	}
	return n
}

func sovNet(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozNet(x uint64) (n int) {
	return sovNet(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Msg) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowNet
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
			return fmt.Errorf("proto: Msg: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Msg: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Msg", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNet
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
				return ErrInvalidLengthNet
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthNet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Msg == nil {
				m.Msg = &NetMsg{}
			}
			if err := m.Msg.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNet
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
				return ErrInvalidLengthNet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthNet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChainId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Flag", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNet
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
				return ErrInvalidLengthNet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthNet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Flag = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipNet(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthNet
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
func (m *NetMsg) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowNet
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
			return fmt.Errorf("proto: NetMsg: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NetMsg: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthNet
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthNet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Payload = append(m.Payload[:0], dAtA[iNdEx:postIndex]...)
			if m.Payload == nil {
				m.Payload = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNet
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= NetMsg_MsgType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field To", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowNet
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
				return ErrInvalidLengthNet
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthNet
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.To = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipNet(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthNet
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
func skipNet(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowNet
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
					return 0, ErrIntOverflowNet
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
					return 0, ErrIntOverflowNet
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
				return 0, ErrInvalidLengthNet
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupNet
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthNet
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthNet        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowNet          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupNet = fmt.Errorf("proto: unexpected end of group")
)
