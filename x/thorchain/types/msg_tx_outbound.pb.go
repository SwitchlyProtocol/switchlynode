// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/msg_tx_outbound.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	common "github.com/switchlyprotocol/switchlynode/v1/common"
	gitlab_com_thorchain_thornode_v3_common "github.com/switchlyprotocol/switchlynode/v1/common"
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

type MsgOutboundTx struct {
	Tx     common.ObservedTx                             `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx"`
	InTxID gitlab_com_thorchain_thornode_v3_common.TxID  `protobuf:"bytes,2,opt,name=in_tx_id,json=inTxId,proto3,casttype=switchlynode/common.TxID" json:"in_tx_id,omitempty"`
	Signer github_com_cosmos_cosmos_sdk_types.AccAddress `protobuf:"bytes,3,opt,name=signer,proto3,casttype=github.com/cosmos/cosmos-sdk/types.AccAddress" json:"signer,omitempty"`
}

func (m *MsgOutboundTx) Reset()         { *m = MsgOutboundTx{} }
func (m *MsgOutboundTx) String() string { return proto.CompactTextString(m) }
func (*MsgOutboundTx) ProtoMessage()    {}
func (*MsgOutboundTx) Descriptor() ([]byte, []int) {
	return fileDescriptor_f7355c25dce1dc28, []int{0}
}
func (m *MsgOutboundTx) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgOutboundTx) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgOutboundTx.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgOutboundTx) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgOutboundTx.Merge(m, src)
}
func (m *MsgOutboundTx) XXX_Size() int {
	return m.Size()
}
func (m *MsgOutboundTx) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgOutboundTx.DiscardUnknown(m)
}

var xxx_messageInfo_MsgOutboundTx proto.InternalMessageInfo

func (m *MsgOutboundTx) GetTx() common.ObservedTx {
	if m != nil {
		return m.Tx
	}
	return common.ObservedTx{}
}

func (m *MsgOutboundTx) GetInTxID() gitlab_com_thorchain_thornode_v3_common.TxID {
	if m != nil {
		return m.InTxID
	}
	return ""
}

func (m *MsgOutboundTx) GetSigner() github_com_cosmos_cosmos_sdk_types.AccAddress {
	if m != nil {
		return m.Signer
	}
	return nil
}

func init() {
	proto.RegisterType((*MsgOutboundTx)(nil), "types.MsgOutboundTx")
}

func init() { proto.RegisterFile("types/msg_tx_outbound.proto", fileDescriptor_f7355c25dce1dc28) }

var fileDescriptor_f7355c25dce1dc28 = []byte{
	// 312 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x50, 0x31, 0x4f, 0xf3, 0x30,
	0x14, 0x8c, 0xfb, 0x7d, 0x44, 0x10, 0x60, 0x09, 0x0c, 0x55, 0x41, 0x4e, 0xc5, 0x94, 0x81, 0x36,
	0xa2, 0xdd, 0xd8, 0x1a, 0xb1, 0x54, 0x02, 0x55, 0xaa, 0x32, 0xb1, 0x44, 0x4d, 0x6c, 0xb9, 0x16,
	0xc4, 0xaf, 0x8a, 0xdd, 0xca, 0xfc, 0x0b, 0x7e, 0x56, 0xc7, 0x8e, 0x88, 0x21, 0x42, 0xe9, 0xbf,
	0xe8, 0x84, 0x9c, 0x18, 0x89, 0x8d, 0xc5, 0xef, 0x74, 0xbe, 0x77, 0x77, 0x7a, 0xde, 0x95, 0x7a,
	0x5b, 0x51, 0x19, 0x15, 0x92, 0xa5, 0x4a, 0xa7, 0xb0, 0x56, 0x19, 0xac, 0x05, 0x19, 0xae, 0x4a,
	0x50, 0xe0, 0x1f, 0x35, 0x9f, 0xbd, 0xeb, 0x56, 0x63, 0xde, 0x14, 0x32, 0x49, 0xcb, 0x0d, 0x25,
	0xa9, 0xd2, 0xad, 0xa8, 0x77, 0xc9, 0x80, 0x41, 0x03, 0x23, 0x83, 0x2c, 0x7b, 0x91, 0x43, 0x51,
	0x80, 0x88, 0xda, 0xd1, 0x92, 0x37, 0x9f, 0xc8, 0x3b, 0x7f, 0x92, 0x6c, 0x66, 0x53, 0x12, 0xed,
	0x87, 0x5e, 0x47, 0xe9, 0x2e, 0xea, 0xa3, 0xf0, 0x74, 0xe4, 0x0f, 0xad, 0x78, 0x66, 0x33, 0x12,
	0x1d, 0xff, 0xdf, 0x56, 0x81, 0x33, 0xef, 0x28, 0xed, 0x27, 0xde, 0x31, 0x17, 0xa6, 0x23, 0x27,
	0xdd, 0x4e, 0x1f, 0x85, 0x27, 0xf1, 0x7d, 0x5d, 0x05, 0xee, 0x54, 0x24, 0x7a, 0xfa, 0x70, 0xa8,
	0x82, 0x5b, 0xc6, 0xd5, 0xeb, 0x22, 0x33, 0x1e, 0x91, 0x5a, 0x42, 0x99, 0x2f, 0x17, 0x5c, 0x34,
	0x48, 0x00, 0xa1, 0xd1, 0x66, 0xfc, 0x53, 0xc5, 0xe8, 0xe7, 0x2e, 0x37, 0x7b, 0xc4, 0x9f, 0x7a,
	0xae, 0xe4, 0x4c, 0xd0, 0xb2, 0xfb, 0xaf, 0x8f, 0xc2, 0xb3, 0xf8, 0xee, 0x50, 0x05, 0x03, 0xc6,
	0xd5, 0x72, 0xdd, 0x3a, 0xe5, 0x20, 0x0b, 0x90, 0x76, 0x0c, 0x24, 0x79, 0x69, 0xae, 0x20, 0x87,
	0x93, 0x3c, 0x9f, 0x10, 0x52, 0x52, 0x29, 0xe7, 0xd6, 0x20, 0x7e, 0xdc, 0xd6, 0x18, 0xed, 0x6a,
	0x8c, 0xbe, 0x6a, 0x8c, 0xde, 0xf7, 0xd8, 0xd9, 0xed, 0xb1, 0xf3, 0xb1, 0xc7, 0xce, 0xf3, 0xe8,
	0xcf, 0x6a, 0xfa, 0x37, 0x6f, 0x02, 0x32, 0xb7, 0xb9, 0xd8, 0xf8, 0x3b, 0x00, 0x00, 0xff, 0xff,
	0xf4, 0x8e, 0x0f, 0xb6, 0xa0, 0x01, 0x00, 0x00,
}

func (m *MsgOutboundTx) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgOutboundTx) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgOutboundTx) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintMsgTxOutbound(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.InTxID) > 0 {
		i -= len(m.InTxID)
		copy(dAtA[i:], m.InTxID)
		i = encodeVarintMsgTxOutbound(dAtA, i, uint64(len(m.InTxID)))
		i--
		dAtA[i] = 0x12
	}
	{
		size, err := m.Tx.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMsgTxOutbound(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintMsgTxOutbound(dAtA []byte, offset int, v uint64) int {
	offset -= sovMsgTxOutbound(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgOutboundTx) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Tx.Size()
	n += 1 + l + sovMsgTxOutbound(uint64(l))
	l = len(m.InTxID)
	if l > 0 {
		n += 1 + l + sovMsgTxOutbound(uint64(l))
	}
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovMsgTxOutbound(uint64(l))
	}
	return n
}

func sovMsgTxOutbound(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMsgTxOutbound(x uint64) (n int) {
	return sovMsgTxOutbound(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgOutboundTx) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMsgTxOutbound
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
			return fmt.Errorf("proto: MsgOutboundTx: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgOutboundTx: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tx", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgTxOutbound
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
				return ErrInvalidLengthMsgTxOutbound
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMsgTxOutbound
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Tx.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InTxID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgTxOutbound
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
				return ErrInvalidLengthMsgTxOutbound
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgTxOutbound
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InTxID = gitlab_com_thorchain_thornode_v3_common.TxID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgTxOutbound
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
				return ErrInvalidLengthMsgTxOutbound
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgTxOutbound
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
			skippy, err := skipMsgTxOutbound(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMsgTxOutbound
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
func skipMsgTxOutbound(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMsgTxOutbound
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
					return 0, ErrIntOverflowMsgTxOutbound
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
					return 0, ErrIntOverflowMsgTxOutbound
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
				return 0, ErrInvalidLengthMsgTxOutbound
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMsgTxOutbound
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMsgTxOutbound
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMsgTxOutbound        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMsgTxOutbound          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMsgTxOutbound = fmt.Errorf("proto: unexpected end of group")
)
