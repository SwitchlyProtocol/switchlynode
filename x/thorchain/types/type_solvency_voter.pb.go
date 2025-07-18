// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/type_solvency_voter.proto

package types

import (
	fmt "fmt"
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

type SolvencyVoter struct {
	Id                   gitlab_com_thorchain_thornode_v3_common.TxID   `protobuf:"bytes,1,opt,name=id,proto3,casttype=switchlynode/common.TxID" json:"id,omitempty"`
	Chain                gitlab_com_thorchain_thornode_v3_common.Chain  `protobuf:"bytes,2,opt,name=chain,proto3,casttype=switchlynode/common.Chain" json:"chain,omitempty"`
	PubKey               gitlab_com_thorchain_thornode_v3_common.PubKey `protobuf:"bytes,3,opt,name=pub_key,json=pubKey,proto3,casttype=switchlynode/common.PubKey" json:"pub_key,omitempty"`
	Coins                gitlab_com_thorchain_thornode_v3_common.Coins  `protobuf:"bytes,4,rep,name=coins,proto3,castrepeated=switchlynode/common.Coins" json:"coins"`
	Height               int64                                          `protobuf:"varint,5,opt,name=height,proto3" json:"height,omitempty"`
	ConsensusBlockHeight int64                                          `protobuf:"varint,6,opt,name=consensus_block_height,json=consensusBlockHeight,proto3" json:"consensus_block_height,omitempty"`
	Signers              []string                                       `protobuf:"bytes,7,rep,name=signers,proto3" json:"signers,omitempty"`
}

func (m *SolvencyVoter) Reset()      { *m = SolvencyVoter{} }
func (*SolvencyVoter) ProtoMessage() {}
func (*SolvencyVoter) Descriptor() ([]byte, []int) {
	return fileDescriptor_6f0a8e59c04c645d, []int{0}
}
func (m *SolvencyVoter) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SolvencyVoter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SolvencyVoter.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SolvencyVoter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SolvencyVoter.Merge(m, src)
}
func (m *SolvencyVoter) XXX_Size() int {
	return m.Size()
}
func (m *SolvencyVoter) XXX_DiscardUnknown() {
	xxx_messageInfo_SolvencyVoter.DiscardUnknown(m)
}

var xxx_messageInfo_SolvencyVoter proto.InternalMessageInfo

func init() {
	proto.RegisterType((*SolvencyVoter)(nil), "types.SolvencyVoter")
}

func init() { proto.RegisterFile("types/type_solvency_voter.proto", fileDescriptor_6f0a8e59c04c645d) }

var fileDescriptor_6f0a8e59c04c645d = []byte{
	// 373 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x4f, 0x6f, 0xa2, 0x40,
	0x18, 0xc6, 0x41, 0x16, 0x8c, 0xb3, 0xbb, 0x17, 0xd6, 0x18, 0xe2, 0x61, 0x20, 0x7b, 0xe2, 0xb0,
	0x0b, 0xad, 0xb6, 0xf7, 0x86, 0x36, 0x69, 0x1b, 0x2f, 0x0d, 0x6d, 0x9a, 0xb6, 0x17, 0x22, 0x38,
	0x81, 0x89, 0x3a, 0x43, 0x18, 0x34, 0x72, 0xeb, 0x47, 0xe8, 0xe7, 0xf0, 0x93, 0x78, 0xf4, 0xe8,
	0x89, 0x56, 0xfc, 0x16, 0x9e, 0x1a, 0x06, 0x6d, 0x7a, 0xab, 0xbd, 0xcc, 0xfb, 0xe7, 0x79, 0xde,
	0xdf, 0xcc, 0x24, 0x2f, 0xd0, 0xd3, 0x2c, 0x46, 0xcc, 0x2e, 0x4f, 0x8f, 0xd1, 0xd1, 0x14, 0x91,
	0x20, 0xf3, 0xa6, 0x34, 0x45, 0x89, 0x15, 0x27, 0x34, 0xa5, 0xaa, 0xcc, 0x0d, 0xed, 0x66, 0x48,
	0x43, 0xca, 0x3b, 0x76, 0x99, 0x55, 0x62, 0xfb, 0x4f, 0x40, 0xc7, 0x63, 0x4a, 0xec, 0x2a, 0x54,
	0xcd, 0xbf, 0x73, 0x09, 0xfc, 0xbe, 0xdd, 0xa1, 0xee, 0x4b, 0x92, 0x7a, 0x06, 0x6a, 0x78, 0xa0,
	0x89, 0x86, 0x68, 0x36, 0x9c, 0xa3, 0x6d, 0xae, 0xff, 0x0b, 0x71, 0x3a, 0xea, 0xfb, 0x56, 0x40,
	0xc7, 0x76, 0x1a, 0xd1, 0x24, 0x88, 0xfa, 0x98, 0xf0, 0x8c, 0xd0, 0x01, 0xb2, 0xa7, 0xdd, 0x3d,
	0xf0, 0x6e, 0x76, 0x7d, 0xe1, 0xd6, 0xf0, 0x40, 0xbd, 0x04, 0x32, 0x37, 0x69, 0x35, 0x0e, 0x39,
	0xde, 0xe6, 0xfa, 0xff, 0x43, 0x21, 0xe7, 0xa5, 0xe0, 0x56, 0xf3, 0x6a, 0x0f, 0xd4, 0xe3, 0x89,
	0xef, 0x0d, 0x51, 0xa6, 0x49, 0x1c, 0xd5, 0xd9, 0xe6, 0xba, 0x75, 0x28, 0xea, 0x66, 0xe2, 0xf7,
	0x50, 0xe6, 0x2a, 0x31, 0x8f, 0xea, 0x23, 0x90, 0x03, 0x8a, 0x09, 0xd3, 0x7e, 0x18, 0x92, 0xf9,
	0xb3, 0xf3, 0xcb, 0xda, 0xdf, 0x48, 0x31, 0x71, 0x4e, 0x17, 0xb9, 0x2e, 0xcc, 0x5f, 0xbf, 0xf1,
	0xce, 0x12, 0xe5, 0x56, 0x44, 0xb5, 0x05, 0x94, 0x08, 0xe1, 0x30, 0x4a, 0x35, 0xd9, 0x10, 0x4d,
	0xc9, 0xdd, 0x55, 0xea, 0x09, 0x68, 0x05, 0x94, 0x30, 0x44, 0xd8, 0x84, 0x79, 0xfe, 0x88, 0x06,
	0x43, 0x6f, 0xe7, 0x53, 0xb8, 0xaf, 0xf9, 0xa1, 0x3a, 0xa5, 0x78, 0x55, 0x4d, 0x69, 0xa0, 0xce,
	0x70, 0x48, 0x50, 0xc2, 0xb4, 0xba, 0x21, 0x99, 0x0d, 0x77, 0x5f, 0x3a, 0x0f, 0x8b, 0x35, 0x14,
	0x56, 0x6b, 0x28, 0x3c, 0x17, 0x50, 0x58, 0x14, 0x50, 0x5c, 0x16, 0x50, 0x7c, 0x2b, 0xa0, 0xf8,
	0xb2, 0x81, 0xc2, 0x72, 0x03, 0x85, 0xd5, 0x06, 0x0a, 0x4f, 0x9d, 0x2f, 0xff, 0x30, 0xfb, 0xdc,
	0x2f, 0x37, 0xc6, 0x57, 0xf8, 0x36, 0x74, 0xdf, 0x03, 0x00, 0x00, 0xff, 0xff, 0x96, 0xe9, 0xf8,
	0x3f, 0x62, 0x02, 0x00, 0x00,
}

func (m *SolvencyVoter) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SolvencyVoter) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SolvencyVoter) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signers) > 0 {
		for iNdEx := len(m.Signers) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Signers[iNdEx])
			copy(dAtA[i:], m.Signers[iNdEx])
			i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(len(m.Signers[iNdEx])))
			i--
			dAtA[i] = 0x3a
		}
	}
	if m.ConsensusBlockHeight != 0 {
		i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(m.ConsensusBlockHeight))
		i--
		dAtA[i] = 0x30
	}
	if m.Height != 0 {
		i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x28
	}
	if len(m.Coins) > 0 {
		for iNdEx := len(m.Coins) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Coins[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.PubKey) > 0 {
		i -= len(m.PubKey)
		copy(dAtA[i:], m.PubKey)
		i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(len(m.PubKey)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Chain) > 0 {
		i -= len(m.Chain)
		copy(dAtA[i:], m.Chain)
		i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(len(m.Chain)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintTypeSolvencyVoter(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTypeSolvencyVoter(dAtA []byte, offset int, v uint64) int {
	offset -= sovTypeSolvencyVoter(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *SolvencyVoter) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovTypeSolvencyVoter(uint64(l))
	}
	l = len(m.Chain)
	if l > 0 {
		n += 1 + l + sovTypeSolvencyVoter(uint64(l))
	}
	l = len(m.PubKey)
	if l > 0 {
		n += 1 + l + sovTypeSolvencyVoter(uint64(l))
	}
	if len(m.Coins) > 0 {
		for _, e := range m.Coins {
			l = e.Size()
			n += 1 + l + sovTypeSolvencyVoter(uint64(l))
		}
	}
	if m.Height != 0 {
		n += 1 + sovTypeSolvencyVoter(uint64(m.Height))
	}
	if m.ConsensusBlockHeight != 0 {
		n += 1 + sovTypeSolvencyVoter(uint64(m.ConsensusBlockHeight))
	}
	if len(m.Signers) > 0 {
		for _, s := range m.Signers {
			l = len(s)
			n += 1 + l + sovTypeSolvencyVoter(uint64(l))
		}
	}
	return n
}

func sovTypeSolvencyVoter(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTypeSolvencyVoter(x uint64) (n int) {
	return sovTypeSolvencyVoter(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *SolvencyVoter) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypeSolvencyVoter
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
			return fmt.Errorf("proto: SolvencyVoter: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SolvencyVoter: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
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
				return ErrInvalidLengthTypeSolvencyVoter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypeSolvencyVoter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = gitlab_com_thorchain_thornode_v3_common.TxID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Chain", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
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
				return ErrInvalidLengthTypeSolvencyVoter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypeSolvencyVoter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Chain = gitlab_com_thorchain_thornode_v3_common.Chain(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PubKey", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
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
				return ErrInvalidLengthTypeSolvencyVoter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypeSolvencyVoter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PubKey = gitlab_com_thorchain_thornode_v3_common.PubKey(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Coins", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
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
				return ErrInvalidLengthTypeSolvencyVoter
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypeSolvencyVoter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Coins = append(m.Coins, common.Coin{})
			if err := m.Coins[len(m.Coins)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConsensusBlockHeight", wireType)
			}
			m.ConsensusBlockHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ConsensusBlockHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signers", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypeSolvencyVoter
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
				return ErrInvalidLengthTypeSolvencyVoter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypeSolvencyVoter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signers = append(m.Signers, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTypeSolvencyVoter(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypeSolvencyVoter
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
func skipTypeSolvencyVoter(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTypeSolvencyVoter
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
					return 0, ErrIntOverflowTypeSolvencyVoter
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
					return 0, ErrIntOverflowTypeSolvencyVoter
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
				return 0, ErrInvalidLengthTypeSolvencyVoter
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTypeSolvencyVoter
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTypeSolvencyVoter
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTypeSolvencyVoter        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypeSolvencyVoter          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTypeSolvencyVoter = fmt.Errorf("proto: unexpected end of group")
)
