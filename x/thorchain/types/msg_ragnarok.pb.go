// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/msg_ragnarok.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	common "github.com/switchlyprotocol/switchlynode/v1/common"
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

type MsgRagnarok struct {
	Tx          common.ObservedTx                             `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx"`
	BlockHeight int64                                         `protobuf:"varint,2,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	Signer      github_com_cosmos_cosmos_sdk_types.AccAddress `protobuf:"bytes,3,opt,name=signer,proto3,casttype=github.com/cosmos/cosmos-sdk/types.AccAddress" json:"signer,omitempty"`
}

func (m *MsgRagnarok) Reset()         { *m = MsgRagnarok{} }
func (m *MsgRagnarok) String() string { return proto.CompactTextString(m) }
func (*MsgRagnarok) ProtoMessage()    {}
func (*MsgRagnarok) Descriptor() ([]byte, []int) {
	return fileDescriptor_5afe0b6fb2e8a0c5, []int{0}
}
func (m *MsgRagnarok) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRagnarok) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRagnarok.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRagnarok) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRagnarok.Merge(m, src)
}
func (m *MsgRagnarok) XXX_Size() int {
	return m.Size()
}
func (m *MsgRagnarok) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRagnarok.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRagnarok proto.InternalMessageInfo

func (m *MsgRagnarok) GetTx() common.ObservedTx {
	if m != nil {
		return m.Tx
	}
	return common.ObservedTx{}
}

func (m *MsgRagnarok) GetBlockHeight() int64 {
	if m != nil {
		return m.BlockHeight
	}
	return 0
}

func (m *MsgRagnarok) GetSigner() github_com_cosmos_cosmos_sdk_types.AccAddress {
	if m != nil {
		return m.Signer
	}
	return nil
}

func init() {
	proto.RegisterType((*MsgRagnarok)(nil), "types.MsgRagnarok")
}

func init() { proto.RegisterFile("types/msg_ragnarok.proto", fileDescriptor_5afe0b6fb2e8a0c5) }

var fileDescriptor_5afe0b6fb2e8a0c5 = []byte{
	// 290 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x8f, 0x31, 0x4e, 0xc3, 0x30,
	0x18, 0x85, 0xe3, 0x16, 0x3a, 0x24, 0x9d, 0x02, 0x43, 0x54, 0x21, 0x37, 0x30, 0x65, 0x69, 0x2c,
	0xda, 0x13, 0x34, 0x13, 0x48, 0x20, 0xa4, 0x88, 0x89, 0x25, 0x4a, 0x1c, 0xcb, 0x89, 0xd2, 0xe4,
	0xaf, 0x6c, 0x53, 0x85, 0x5b, 0x70, 0x09, 0xee, 0xd2, 0xb1, 0x23, 0x53, 0x85, 0x92, 0x5b, 0x30,
	0xa1, 0x38, 0x1e, 0x58, 0xfc, 0x3f, 0x7d, 0x7e, 0x7a, 0x4f, 0xcf, 0xf6, 0xd4, 0xc7, 0x9e, 0x49,
	0x52, 0x4b, 0x9e, 0x88, 0x94, 0x37, 0xa9, 0x80, 0x2a, 0xdc, 0x0b, 0x50, 0xe0, 0x5e, 0xea, 0x9f,
	0xc5, 0xcd, 0x68, 0x18, 0xde, 0x04, 0x32, 0xc9, 0xc4, 0x81, 0xe5, 0x89, 0x6a, 0x47, 0xd3, 0xe2,
	0x9a, 0x03, 0x07, 0x2d, 0xc9, 0xa0, 0x0c, 0xbd, 0xa2, 0x50, 0xd7, 0xd0, 0x90, 0xf1, 0x8c, 0xf0,
	0xee, 0x0b, 0xd9, 0xce, 0xb3, 0xe4, 0xb1, 0x69, 0x71, 0x03, 0x7b, 0xa2, 0x5a, 0x0f, 0xf9, 0x28,
	0x70, 0xd6, 0x6e, 0x68, 0xac, 0x2f, 0xa6, 0xe1, 0xb5, 0x8d, 0x2e, 0x8e, 0xe7, 0xa5, 0x15, 0x4f,
	0x54, 0xeb, 0xde, 0xda, 0xf3, 0x6c, 0x07, 0xb4, 0x4a, 0x0a, 0x56, 0xf2, 0x42, 0x79, 0x13, 0x1f,
	0x05, 0xd3, 0xd8, 0xd1, 0xec, 0x41, 0x23, 0xf7, 0xd1, 0x9e, 0xc9, 0x92, 0x37, 0x4c, 0x78, 0x53,
	0x1f, 0x05, 0xf3, 0xe8, 0xfe, 0xf7, 0xbc, 0x5c, 0xf1, 0x52, 0x15, 0xef, 0xd9, 0x10, 0x4d, 0x28,
	0xc8, 0x1a, 0xa4, 0x39, 0x2b, 0x99, 0x57, 0x7a, 0x90, 0x0c, 0xb7, 0x94, 0x6e, 0xf3, 0x5c, 0x30,
	0x29, 0x63, 0x13, 0x10, 0x3d, 0x1d, 0x3b, 0x8c, 0x4e, 0x1d, 0x46, 0x3f, 0x1d, 0x46, 0x9f, 0x3d,
	0xb6, 0x4e, 0x3d, 0xb6, 0xbe, 0x7b, 0x6c, 0xbd, 0xad, 0x79, 0xa9, 0x76, 0xe9, 0x18, 0xa8, 0x0a,
	0x10, 0xb4, 0x48, 0xcb, 0x46, 0xab, 0x06, 0x72, 0x46, 0x0e, 0x1b, 0xd2, 0xfe, 0xe7, 0x43, 0x41,
	0x36, 0xd3, 0xe3, 0x37, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x2f, 0xbf, 0xab, 0x86, 0x68, 0x01,
	0x00, 0x00,
}

func (m *MsgRagnarok) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRagnarok) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRagnarok) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintMsgRagnarok(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0x1a
	}
	if m.BlockHeight != 0 {
		i = encodeVarintMsgRagnarok(dAtA, i, uint64(m.BlockHeight))
		i--
		dAtA[i] = 0x10
	}
	{
		size, err := m.Tx.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMsgRagnarok(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintMsgRagnarok(dAtA []byte, offset int, v uint64) int {
	offset -= sovMsgRagnarok(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgRagnarok) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Tx.Size()
	n += 1 + l + sovMsgRagnarok(uint64(l))
	if m.BlockHeight != 0 {
		n += 1 + sovMsgRagnarok(uint64(m.BlockHeight))
	}
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovMsgRagnarok(uint64(l))
	}
	return n
}

func sovMsgRagnarok(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMsgRagnarok(x uint64) (n int) {
	return sovMsgRagnarok(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgRagnarok) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMsgRagnarok
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
			return fmt.Errorf("proto: MsgRagnarok: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRagnarok: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tx", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRagnarok
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
				return ErrInvalidLengthMsgRagnarok
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRagnarok
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Tx.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockHeight", wireType)
			}
			m.BlockHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRagnarok
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BlockHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRagnarok
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
				return ErrInvalidLengthMsgRagnarok
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRagnarok
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signer = append(m.Signer[:0], dAtA[iNdEx:postIndex]...)
			if m.Signer == nil {
				m.Signer = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMsgRagnarok(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMsgRagnarok
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
func skipMsgRagnarok(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMsgRagnarok
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
					return 0, ErrIntOverflowMsgRagnarok
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
					return 0, ErrIntOverflowMsgRagnarok
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
				return 0, ErrInvalidLengthMsgRagnarok
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMsgRagnarok
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMsgRagnarok
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMsgRagnarok        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMsgRagnarok          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMsgRagnarok = fmt.Errorf("proto: unexpected end of group")
)
