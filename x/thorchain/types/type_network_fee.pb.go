// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/type_network_fee.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

// NetworkFee represents the fee rate and typical outbound transaction size. Some chains
// may have simplifid usage to report the fee as the "fee_rate" and set size to 1.
type NetworkFee struct {
	Chain              gitlab_com_thorchain_thornode_v3_common.Chain `protobuf:"bytes,1,opt,name=chain,proto3,casttype=switchlynode/common.Chain" json:"chain,omitempty"`
	TransactionSize    uint64                                        `protobuf:"varint,2,opt,name=transaction_size,json=transactionSize,proto3" json:"transaction_size,omitempty"`
	TransactionFeeRate uint64                                        `protobuf:"varint,3,opt,name=transaction_fee_rate,json=transactionFeeRate,proto3" json:"transaction_fee_rate,omitempty"`
}

func (m *NetworkFee) Reset()         { *m = NetworkFee{} }
func (m *NetworkFee) String() string { return proto.CompactTextString(m) }
func (*NetworkFee) ProtoMessage()    {}
func (*NetworkFee) Descriptor() ([]byte, []int) {
	return fileDescriptor_432789ad71171d83, []int{0}
}
func (m *NetworkFee) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NetworkFee) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NetworkFee.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NetworkFee) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NetworkFee.Merge(m, src)
}
func (m *NetworkFee) XXX_Size() int {
	return m.Size()
}
func (m *NetworkFee) XXX_DiscardUnknown() {
	xxx_messageInfo_NetworkFee.DiscardUnknown(m)
}

var xxx_messageInfo_NetworkFee proto.InternalMessageInfo

func (m *NetworkFee) GetChain() gitlab_com_thorchain_thornode_v3_common.Chain {
	if m != nil {
		return m.Chain
	}
	return ""
}

func (m *NetworkFee) GetTransactionSize() uint64 {
	if m != nil {
		return m.TransactionSize
	}
	return 0
}

func (m *NetworkFee) GetTransactionFeeRate() uint64 {
	if m != nil {
		return m.TransactionFeeRate
	}
	return 0
}

func init() {
	proto.RegisterType((*NetworkFee)(nil), "types.NetworkFee")
}

func init() { proto.RegisterFile("types/type_network_fee.proto", fileDescriptor_432789ad71171d83) }

var fileDescriptor_432789ad71171d83 = []byte{
	// 245 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x29, 0xa9, 0x2c, 0x48,
	0x2d, 0xd6, 0x07, 0x91, 0xf1, 0x79, 0xa9, 0x25, 0xe5, 0xf9, 0x45, 0xd9, 0xf1, 0x69, 0xa9, 0xa9,
	0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0xac, 0x60, 0x59, 0x29, 0x91, 0xf4, 0xfc, 0xf4, 0x7c,
	0xb0, 0x88, 0x3e, 0x88, 0x05, 0x91, 0x54, 0xda, 0xc4, 0xc8, 0xc5, 0xe5, 0x07, 0xd1, 0xe2, 0x96,
	0x9a, 0x2a, 0xe4, 0xce, 0xc5, 0x9a, 0x9c, 0x91, 0x98, 0x99, 0x27, 0xc1, 0xa8, 0xc0, 0xa8, 0xc1,
	0xe9, 0x64, 0xf8, 0xeb, 0x9e, 0xbc, 0x6e, 0x7a, 0x66, 0x49, 0x4e, 0x62, 0x92, 0x5e, 0x72, 0x7e,
	0xae, 0x7e, 0x49, 0x46, 0x7e, 0x11, 0x58, 0x1e, 0xcc, 0xca, 0xcb, 0x4f, 0x49, 0xd5, 0x2f, 0x33,
	0xd6, 0x4f, 0xce, 0xcf, 0xcd, 0xcd, 0xcf, 0xd3, 0x73, 0x06, 0x49, 0x04, 0x41, 0xf4, 0x0b, 0x69,
	0x72, 0x09, 0x94, 0x14, 0x25, 0xe6, 0x15, 0x27, 0x26, 0x97, 0x64, 0xe6, 0xe7, 0xc5, 0x17, 0x67,
	0x56, 0xa5, 0x4a, 0x30, 0x29, 0x30, 0x6a, 0xb0, 0x04, 0xf1, 0x23, 0x89, 0x07, 0x67, 0x56, 0xa5,
	0x0a, 0x19, 0x70, 0x89, 0x20, 0x2b, 0x4d, 0x4b, 0x4d, 0x8d, 0x2f, 0x4a, 0x2c, 0x49, 0x95, 0x60,
	0x06, 0x2b, 0x17, 0x42, 0x92, 0x73, 0x4b, 0x4d, 0x0d, 0x4a, 0x2c, 0x49, 0x75, 0xf2, 0x39, 0xf1,
	0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96, 0x63, 0xb8,
	0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x86, 0x28, 0x23, 0x82, 0x8e, 0xad, 0x40, 0x16, 0x07,
	0x05, 0x4c, 0x12, 0x1b, 0x38, 0x24, 0x8c, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x79, 0xbe, 0x34,
	0xbd, 0x46, 0x01, 0x00, 0x00,
}

func (m *NetworkFee) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NetworkFee) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NetworkFee) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.TransactionFeeRate != 0 {
		i = encodeVarintTypeNetworkFee(dAtA, i, uint64(m.TransactionFeeRate))
		i--
		dAtA[i] = 0x18
	}
	if m.TransactionSize != 0 {
		i = encodeVarintTypeNetworkFee(dAtA, i, uint64(m.TransactionSize))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Chain) > 0 {
		i -= len(m.Chain)
		copy(dAtA[i:], m.Chain)
		i = encodeVarintTypeNetworkFee(dAtA, i, uint64(len(m.Chain)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTypeNetworkFee(dAtA []byte, offset int, v uint64) int {
	offset -= sovTypeNetworkFee(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *NetworkFee) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Chain)
	if l > 0 {
		n += 1 + l + sovTypeNetworkFee(uint64(l))
	}
	if m.TransactionSize != 0 {
		n += 1 + sovTypeNetworkFee(uint64(m.TransactionSize))
	}
	if m.TransactionFeeRate != 0 {
		n += 1 + sovTypeNetworkFee(uint64(m.TransactionFeeRate))
	}
	return n
}

func sovTypeNetworkFee(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTypeNetworkFee(x uint64) (n int) {
	return sovTypeNetworkFee(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *NetworkFee) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypeNetworkFee
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
			return fmt.Errorf("proto: NetworkFee: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NetworkFee: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Chain", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeNetworkFee
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
				return ErrInvalidLengthTypeNetworkFee
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypeNetworkFee
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Chain = gitlab_com_thorchain_thornode_v3_common.Chain(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TransactionSize", wireType)
			}
			m.TransactionSize = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeNetworkFee
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TransactionSize |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TransactionFeeRate", wireType)
			}
			m.TransactionFeeRate = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeNetworkFee
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TransactionFeeRate |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTypeNetworkFee(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypeNetworkFee
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
func skipTypeNetworkFee(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTypeNetworkFee
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
					return 0, ErrIntOverflowTypeNetworkFee
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
					return 0, ErrIntOverflowTypeNetworkFee
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
				return 0, ErrInvalidLengthTypeNetworkFee
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTypeNetworkFee
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTypeNetworkFee
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTypeNetworkFee        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypeNetworkFee          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTypeNetworkFee = fmt.Errorf("proto: unexpected end of group")
)
