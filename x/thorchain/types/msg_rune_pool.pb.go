// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: thorchain/v1/x/thorchain/types/msg_rune_pool.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	common "gitlab.com/thorchain/thornode/common"
	gitlab_com_thorchain_thornode_common "gitlab.com/thorchain/thornode/common"
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

type MsgRunePoolDeposit struct {
	Signer github_com_cosmos_cosmos_sdk_types.AccAddress `protobuf:"bytes,1,opt,name=signer,proto3,casttype=github.com/cosmos/cosmos-sdk/types.AccAddress" json:"signer,omitempty"`
	Tx     common.Tx                                     `protobuf:"bytes,2,opt,name=tx,proto3" json:"tx"`
}

func (m *MsgRunePoolDeposit) Reset()         { *m = MsgRunePoolDeposit{} }
func (m *MsgRunePoolDeposit) String() string { return proto.CompactTextString(m) }
func (*MsgRunePoolDeposit) ProtoMessage()    {}
func (*MsgRunePoolDeposit) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc2ca6e098490372, []int{0}
}
func (m *MsgRunePoolDeposit) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRunePoolDeposit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRunePoolDeposit.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRunePoolDeposit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRunePoolDeposit.Merge(m, src)
}
func (m *MsgRunePoolDeposit) XXX_Size() int {
	return m.Size()
}
func (m *MsgRunePoolDeposit) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRunePoolDeposit.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRunePoolDeposit proto.InternalMessageInfo

func (m *MsgRunePoolDeposit) GetSigner() github_com_cosmos_cosmos_sdk_types.AccAddress {
	if m != nil {
		return m.Signer
	}
	return nil
}

func (m *MsgRunePoolDeposit) GetTx() common.Tx {
	if m != nil {
		return m.Tx
	}
	return common.Tx{}
}

type MsgRunePoolWithdraw struct {
	Signer               github_com_cosmos_cosmos_sdk_types.AccAddress `protobuf:"bytes,1,opt,name=signer,proto3,casttype=github.com/cosmos/cosmos-sdk/types.AccAddress" json:"signer,omitempty"`
	Tx                   common.Tx                                     `protobuf:"bytes,2,opt,name=tx,proto3" json:"tx"`
	BasisPoints          github_com_cosmos_cosmos_sdk_types.Uint       `protobuf:"bytes,3,opt,name=basis_points,json=basisPoints,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Uint" json:"basis_points"`
	AffiliateAddress     gitlab_com_thorchain_thornode_common.Address  `protobuf:"bytes,4,opt,name=affiliate_address,json=affiliateAddress,proto3,casttype=gitlab.com/thorchain/thornode/common.Address" json:"affiliate_address,omitempty"`
	AffiliateBasisPoints github_com_cosmos_cosmos_sdk_types.Uint       `protobuf:"bytes,5,opt,name=affiliate_basis_points,json=affiliateBasisPoints,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Uint" json:"affiliate_basis_points"`
}

func (m *MsgRunePoolWithdraw) Reset()         { *m = MsgRunePoolWithdraw{} }
func (m *MsgRunePoolWithdraw) String() string { return proto.CompactTextString(m) }
func (*MsgRunePoolWithdraw) ProtoMessage()    {}
func (*MsgRunePoolWithdraw) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc2ca6e098490372, []int{1}
}
func (m *MsgRunePoolWithdraw) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRunePoolWithdraw) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRunePoolWithdraw.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRunePoolWithdraw) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRunePoolWithdraw.Merge(m, src)
}
func (m *MsgRunePoolWithdraw) XXX_Size() int {
	return m.Size()
}
func (m *MsgRunePoolWithdraw) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRunePoolWithdraw.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRunePoolWithdraw proto.InternalMessageInfo

func (m *MsgRunePoolWithdraw) GetSigner() github_com_cosmos_cosmos_sdk_types.AccAddress {
	if m != nil {
		return m.Signer
	}
	return nil
}

func (m *MsgRunePoolWithdraw) GetTx() common.Tx {
	if m != nil {
		return m.Tx
	}
	return common.Tx{}
}

func (m *MsgRunePoolWithdraw) GetAffiliateAddress() gitlab_com_thorchain_thornode_common.Address {
	if m != nil {
		return m.AffiliateAddress
	}
	return ""
}

func init() {
	proto.RegisterType((*MsgRunePoolDeposit)(nil), "types.MsgRunePoolDeposit")
	proto.RegisterType((*MsgRunePoolWithdraw)(nil), "types.MsgRunePoolWithdraw")
}

func init() {
	proto.RegisterFile("thorchain/v1/x/thorchain/types/msg_rune_pool.proto", fileDescriptor_dc2ca6e098490372)
}

var fileDescriptor_dc2ca6e098490372 = []byte{
	// 381 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x92, 0xcf, 0xca, 0xd3, 0x40,
	0x10, 0xc0, 0xb3, 0xfd, 0x07, 0x6e, 0x7b, 0xd0, 0x58, 0x24, 0xf4, 0x90, 0x84, 0x5e, 0xec, 0xc1,
	0x66, 0x6d, 0x7d, 0x82, 0x06, 0x2f, 0x3d, 0x08, 0x25, 0x28, 0x82, 0x20, 0x21, 0x4d, 0xb6, 0xc9,
	0x62, 0xb2, 0x13, 0xb2, 0x5b, 0x8d, 0x47, 0xdf, 0xc0, 0x07, 0xf0, 0x81, 0x7a, 0xec, 0x51, 0x3c,
	0x04, 0x69, 0xdf, 0xa2, 0x27, 0x69, 0x92, 0xfe, 0x43, 0xf8, 0xf8, 0xf8, 0x0e, 0xdf, 0x69, 0x26,
	0x33, 0xc9, 0x2f, 0xbf, 0x61, 0x06, 0x4f, 0x65, 0x04, 0x99, 0x1f, 0x79, 0x8c, 0x93, 0xaf, 0x13,
	0x92, 0x93, 0xcb, 0xa3, 0xfc, 0x9e, 0x52, 0x41, 0x12, 0x11, 0xba, 0xd9, 0x9a, 0x53, 0x37, 0x05,
	0x88, 0xad, 0x34, 0x03, 0x09, 0x6a, 0xbb, 0x6c, 0x0d, 0xcc, 0x9b, 0x4f, 0x7d, 0x48, 0x12, 0xe0,
	0x75, 0xa8, 0x5e, 0x1c, 0xf4, 0x43, 0x08, 0xa1, 0x4c, 0xc9, 0x31, 0xab, 0xaa, 0xc3, 0x1f, 0x08,
	0xab, 0xef, 0x44, 0xe8, 0xac, 0x39, 0x5d, 0x00, 0xc4, 0x6f, 0x69, 0x0a, 0x82, 0x49, 0x75, 0x8e,
	0x3b, 0x82, 0x85, 0x9c, 0x66, 0x1a, 0x32, 0xd1, 0xa8, 0x67, 0x4f, 0x0e, 0x85, 0x31, 0x0e, 0x99,
	0x8c, 0xd6, 0x4b, 0xcb, 0x87, 0x84, 0xf8, 0x20, 0x12, 0x10, 0x75, 0x18, 0x8b, 0xe0, 0x4b, 0x65,
	0x68, 0xcd, 0x7c, 0x7f, 0x16, 0x04, 0x19, 0x15, 0xc2, 0xa9, 0x01, 0xaa, 0x89, 0x1b, 0x32, 0xd7,
	0x1a, 0x26, 0x1a, 0x75, 0xa7, 0xd8, 0xaa, 0x95, 0xde, 0xe7, 0x76, 0x6b, 0x53, 0x18, 0x8a, 0xd3,
	0x90, 0xf9, 0xf0, 0x57, 0x13, 0x3f, 0xbf, 0x72, 0xf8, 0xc8, 0x64, 0x14, 0x64, 0xde, 0xb7, 0x47,
	0x95, 0x50, 0x1d, 0xdc, 0x5b, 0x7a, 0x82, 0x09, 0x37, 0x05, 0xc6, 0xa5, 0xd0, 0x9a, 0x26, 0x1a,
	0x3d, 0xb1, 0xc9, 0xb1, 0xff, 0xa7, 0x30, 0x5e, 0xde, 0xe3, 0xb7, 0x1f, 0x18, 0x97, 0x4e, 0xb7,
	0x84, 0x2c, 0x4a, 0x86, 0xfa, 0x19, 0x3f, 0xf3, 0x56, 0x2b, 0x16, 0x33, 0x4f, 0x52, 0xd7, 0xab,
	0x94, 0xb4, 0x56, 0x09, 0x7e, 0x7d, 0x28, 0x8c, 0x57, 0x21, 0x93, 0xb1, 0x57, 0x41, 0xaf, 0x56,
	0x1d, 0x41, 0xc6, 0x21, 0xa0, 0xa7, 0xed, 0x9d, 0x46, 0x79, 0x7a, 0x46, 0xd5, 0x15, 0x95, 0xe2,
	0x17, 0x17, 0xfc, 0x8d, 0x7c, 0xfb, 0x61, 0xf2, 0xfd, 0x33, 0xce, 0xbe, 0x4c, 0x61, 0xcf, 0x37,
	0x3b, 0x1d, 0x6d, 0x77, 0x3a, 0xfa, 0xbb, 0xd3, 0xd1, 0xcf, 0xbd, 0xae, 0x6c, 0xf7, 0xba, 0xf2,
	0x7b, 0xaf, 0x2b, 0x9f, 0xc8, 0xdd, 0x03, 0xfc, 0x77, 0xc0, 0xcb, 0x4e, 0x79, 0x74, 0x6f, 0xfe,
	0x05, 0x00, 0x00, 0xff, 0xff, 0x5a, 0x53, 0x74, 0x48, 0xe9, 0x02, 0x00, 0x00,
}

func (m *MsgRunePoolDeposit) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRunePoolDeposit) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRunePoolDeposit) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Tx.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMsgRunePool(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintMsgRunePool(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgRunePoolWithdraw) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRunePoolWithdraw) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRunePoolWithdraw) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.AffiliateBasisPoints.Size()
		i -= size
		if _, err := m.AffiliateBasisPoints.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintMsgRunePool(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if len(m.AffiliateAddress) > 0 {
		i -= len(m.AffiliateAddress)
		copy(dAtA[i:], m.AffiliateAddress)
		i = encodeVarintMsgRunePool(dAtA, i, uint64(len(m.AffiliateAddress)))
		i--
		dAtA[i] = 0x22
	}
	{
		size := m.BasisPoints.Size()
		i -= size
		if _, err := m.BasisPoints.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintMsgRunePool(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	{
		size, err := m.Tx.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMsgRunePool(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintMsgRunePool(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMsgRunePool(dAtA []byte, offset int, v uint64) int {
	offset -= sovMsgRunePool(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgRunePoolDeposit) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovMsgRunePool(uint64(l))
	}
	l = m.Tx.Size()
	n += 1 + l + sovMsgRunePool(uint64(l))
	return n
}

func (m *MsgRunePoolWithdraw) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovMsgRunePool(uint64(l))
	}
	l = m.Tx.Size()
	n += 1 + l + sovMsgRunePool(uint64(l))
	l = m.BasisPoints.Size()
	n += 1 + l + sovMsgRunePool(uint64(l))
	l = len(m.AffiliateAddress)
	if l > 0 {
		n += 1 + l + sovMsgRunePool(uint64(l))
	}
	l = m.AffiliateBasisPoints.Size()
	n += 1 + l + sovMsgRunePool(uint64(l))
	return n
}

func sovMsgRunePool(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMsgRunePool(x uint64) (n int) {
	return sovMsgRunePool(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgRunePoolDeposit) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMsgRunePool
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
			return fmt.Errorf("proto: MsgRunePoolDeposit: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRunePoolDeposit: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signer = append(m.Signer[:0], dAtA[iNdEx:postIndex]...)
			if m.Signer == nil {
				m.Signer = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tx", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Tx.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMsgRunePool(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMsgRunePool
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
func (m *MsgRunePoolWithdraw) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMsgRunePool
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
			return fmt.Errorf("proto: MsgRunePoolWithdraw: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRunePoolWithdraw: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signer = append(m.Signer[:0], dAtA[iNdEx:postIndex]...)
			if m.Signer == nil {
				m.Signer = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tx", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Tx.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BasisPoints", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.BasisPoints.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AffiliateAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AffiliateAddress = gitlab_com_thorchain_thornode_common.Address(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AffiliateBasisPoints", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMsgRunePool
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
				return ErrInvalidLengthMsgRunePool
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.AffiliateBasisPoints.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMsgRunePool(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMsgRunePool
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMsgRunePool
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
func skipMsgRunePool(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMsgRunePool
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
					return 0, ErrIntOverflowMsgRunePool
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
					return 0, ErrIntOverflowMsgRunePool
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
				return 0, ErrInvalidLengthMsgRunePool
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMsgRunePool
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMsgRunePool
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMsgRunePool        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMsgRunePool          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMsgRunePool = fmt.Errorf("proto: unexpected end of group")
)
