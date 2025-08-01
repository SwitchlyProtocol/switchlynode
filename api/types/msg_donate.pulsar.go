// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package types

import (
	fmt "fmt"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/gogoproto/gogoproto"
	common "github.com/switchlyprotocol/switchlynode/v1/api/common"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	io "io"
	reflect "reflect"
	sync "sync"
)

var (
	md_MsgDonate              protoreflect.MessageDescriptor
	fd_MsgDonate_asset        protoreflect.FieldDescriptor
	fd_MsgDonate_asset_amount protoreflect.FieldDescriptor
	fd_MsgDonate_rune_amount  protoreflect.FieldDescriptor
	fd_MsgDonate_tx           protoreflect.FieldDescriptor
	fd_MsgDonate_signer       protoreflect.FieldDescriptor
)

func init() {
	file_types_msg_donate_proto_init()
	md_MsgDonate = File_types_msg_donate_proto.Messages().ByName("MsgDonate")
	fd_MsgDonate_asset = md_MsgDonate.Fields().ByName("asset")
	fd_MsgDonate_asset_amount = md_MsgDonate.Fields().ByName("asset_amount")
	fd_MsgDonate_rune_amount = md_MsgDonate.Fields().ByName("rune_amount")
	fd_MsgDonate_tx = md_MsgDonate.Fields().ByName("tx")
	fd_MsgDonate_signer = md_MsgDonate.Fields().ByName("signer")
}

var _ protoreflect.Message = (*fastReflection_MsgDonate)(nil)

type fastReflection_MsgDonate MsgDonate

func (x *MsgDonate) ProtoReflect() protoreflect.Message {
	return (*fastReflection_MsgDonate)(x)
}

func (x *MsgDonate) slowProtoReflect() protoreflect.Message {
	mi := &file_types_msg_donate_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_MsgDonate_messageType fastReflection_MsgDonate_messageType
var _ protoreflect.MessageType = fastReflection_MsgDonate_messageType{}

type fastReflection_MsgDonate_messageType struct{}

func (x fastReflection_MsgDonate_messageType) Zero() protoreflect.Message {
	return (*fastReflection_MsgDonate)(nil)
}
func (x fastReflection_MsgDonate_messageType) New() protoreflect.Message {
	return new(fastReflection_MsgDonate)
}
func (x fastReflection_MsgDonate_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_MsgDonate
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_MsgDonate) Descriptor() protoreflect.MessageDescriptor {
	return md_MsgDonate
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_MsgDonate) Type() protoreflect.MessageType {
	return _fastReflection_MsgDonate_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_MsgDonate) New() protoreflect.Message {
	return new(fastReflection_MsgDonate)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_MsgDonate) Interface() protoreflect.ProtoMessage {
	return (*MsgDonate)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_MsgDonate) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.Asset != nil {
		value := protoreflect.ValueOfMessage(x.Asset.ProtoReflect())
		if !f(fd_MsgDonate_asset, value) {
			return
		}
	}
	if x.AssetAmount != "" {
		value := protoreflect.ValueOfString(x.AssetAmount)
		if !f(fd_MsgDonate_asset_amount, value) {
			return
		}
	}
	if x.RuneAmount != "" {
		value := protoreflect.ValueOfString(x.RuneAmount)
		if !f(fd_MsgDonate_rune_amount, value) {
			return
		}
	}
	if x.Tx != nil {
		value := protoreflect.ValueOfMessage(x.Tx.ProtoReflect())
		if !f(fd_MsgDonate_tx, value) {
			return
		}
	}
	if len(x.Signer) != 0 {
		value := protoreflect.ValueOfBytes(x.Signer)
		if !f(fd_MsgDonate_signer, value) {
			return
		}
	}
}

// Has reports whether a field is populated.
//
// Some fields have the property of nullability where it is possible to
// distinguish between the default value of a field and whether the field
// was explicitly populated with the default value. Singular message fields,
// member fields of a oneof, and proto2 scalar fields are nullable. Such
// fields are populated only if explicitly set.
//
// In other cases (aside from the nullable cases above),
// a proto3 scalar field is populated if it contains a non-zero value, and
// a repeated field is populated if it is non-empty.
func (x *fastReflection_MsgDonate) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "types.MsgDonate.asset":
		return x.Asset != nil
	case "types.MsgDonate.asset_amount":
		return x.AssetAmount != ""
	case "types.MsgDonate.rune_amount":
		return x.RuneAmount != ""
	case "types.MsgDonate.tx":
		return x.Tx != nil
	case "types.MsgDonate.signer":
		return len(x.Signer) != 0
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: types.MsgDonate"))
		}
		panic(fmt.Errorf("message types.MsgDonate does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_MsgDonate) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "types.MsgDonate.asset":
		x.Asset = nil
	case "types.MsgDonate.asset_amount":
		x.AssetAmount = ""
	case "types.MsgDonate.rune_amount":
		x.RuneAmount = ""
	case "types.MsgDonate.tx":
		x.Tx = nil
	case "types.MsgDonate.signer":
		x.Signer = nil
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: types.MsgDonate"))
		}
		panic(fmt.Errorf("message types.MsgDonate does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_MsgDonate) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "types.MsgDonate.asset":
		value := x.Asset
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	case "types.MsgDonate.asset_amount":
		value := x.AssetAmount
		return protoreflect.ValueOfString(value)
	case "types.MsgDonate.rune_amount":
		value := x.RuneAmount
		return protoreflect.ValueOfString(value)
	case "types.MsgDonate.tx":
		value := x.Tx
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	case "types.MsgDonate.signer":
		value := x.Signer
		return protoreflect.ValueOfBytes(value)
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: types.MsgDonate"))
		}
		panic(fmt.Errorf("message types.MsgDonate does not contain field %s", descriptor.FullName()))
	}
}

// Set stores the value for a field.
//
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType.
// When setting a composite type, it is unspecified whether the stored value
// aliases the source's memory in any way. If the composite value is an
// empty, read-only value, then it panics.
//
// Set is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_MsgDonate) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "types.MsgDonate.asset":
		x.Asset = value.Message().Interface().(*common.Asset)
	case "types.MsgDonate.asset_amount":
		x.AssetAmount = value.Interface().(string)
	case "types.MsgDonate.rune_amount":
		x.RuneAmount = value.Interface().(string)
	case "types.MsgDonate.tx":
		x.Tx = value.Message().Interface().(*common.Tx)
	case "types.MsgDonate.signer":
		x.Signer = value.Bytes()
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: types.MsgDonate"))
		}
		panic(fmt.Errorf("message types.MsgDonate does not contain field %s", fd.FullName()))
	}
}

// Mutable returns a mutable reference to a composite type.
//
// If the field is unpopulated, it may allocate a composite value.
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType
// if not already stored.
// It panics if the field does not contain a composite type.
//
// Mutable is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_MsgDonate) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "types.MsgDonate.asset":
		if x.Asset == nil {
			x.Asset = new(common.Asset)
		}
		return protoreflect.ValueOfMessage(x.Asset.ProtoReflect())
	case "types.MsgDonate.tx":
		if x.Tx == nil {
			x.Tx = new(common.Tx)
		}
		return protoreflect.ValueOfMessage(x.Tx.ProtoReflect())
	case "types.MsgDonate.asset_amount":
		panic(fmt.Errorf("field asset_amount of message types.MsgDonate is not mutable"))
	case "types.MsgDonate.rune_amount":
		panic(fmt.Errorf("field rune_amount of message types.MsgDonate is not mutable"))
	case "types.MsgDonate.signer":
		panic(fmt.Errorf("field signer of message types.MsgDonate is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: types.MsgDonate"))
		}
		panic(fmt.Errorf("message types.MsgDonate does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_MsgDonate) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "types.MsgDonate.asset":
		m := new(common.Asset)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	case "types.MsgDonate.asset_amount":
		return protoreflect.ValueOfString("")
	case "types.MsgDonate.rune_amount":
		return protoreflect.ValueOfString("")
	case "types.MsgDonate.tx":
		m := new(common.Tx)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	case "types.MsgDonate.signer":
		return protoreflect.ValueOfBytes(nil)
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: types.MsgDonate"))
		}
		panic(fmt.Errorf("message types.MsgDonate does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_MsgDonate) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in types.MsgDonate", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_MsgDonate) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_MsgDonate) SetUnknown(fields protoreflect.RawFields) {
	x.unknownFields = fields
}

// IsValid reports whether the message is valid.
//
// An invalid message is an empty, read-only value.
//
// An invalid message often corresponds to a nil pointer of the concrete
// message type, but the details are implementation dependent.
// Validity is not part of the protobuf data model, and may not
// be preserved in marshaling or other operations.
func (x *fastReflection_MsgDonate) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_MsgDonate) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*MsgDonate)
		if x == nil {
			return protoiface.SizeOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Size:              0,
			}
		}
		options := runtime.SizeInputToOptions(input)
		_ = options
		var n int
		var l int
		_ = l
		if x.Asset != nil {
			l = options.Size(x.Asset)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.AssetAmount)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.RuneAmount)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.Tx != nil {
			l = options.Size(x.Tx)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.Signer)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.unknownFields != nil {
			n += len(x.unknownFields)
		}
		return protoiface.SizeOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Size:              n,
		}
	}

	marshal := func(input protoiface.MarshalInput) (protoiface.MarshalOutput, error) {
		x := input.Message.Interface().(*MsgDonate)
		if x == nil {
			return protoiface.MarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Buf:               input.Buf,
			}, nil
		}
		options := runtime.MarshalInputToOptions(input)
		_ = options
		size := options.Size(x)
		dAtA := make([]byte, size)
		i := len(dAtA)
		_ = i
		var l int
		_ = l
		if x.unknownFields != nil {
			i -= len(x.unknownFields)
			copy(dAtA[i:], x.unknownFields)
		}
		if len(x.Signer) > 0 {
			i -= len(x.Signer)
			copy(dAtA[i:], x.Signer)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.Signer)))
			i--
			dAtA[i] = 0x2a
		}
		if x.Tx != nil {
			encoded, err := options.Marshal(x.Tx)
			if err != nil {
				return protoiface.MarshalOutput{
					NoUnkeyedLiterals: input.NoUnkeyedLiterals,
					Buf:               input.Buf,
				}, err
			}
			i -= len(encoded)
			copy(dAtA[i:], encoded)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
			i--
			dAtA[i] = 0x22
		}
		if len(x.RuneAmount) > 0 {
			i -= len(x.RuneAmount)
			copy(dAtA[i:], x.RuneAmount)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.RuneAmount)))
			i--
			dAtA[i] = 0x1a
		}
		if len(x.AssetAmount) > 0 {
			i -= len(x.AssetAmount)
			copy(dAtA[i:], x.AssetAmount)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.AssetAmount)))
			i--
			dAtA[i] = 0x12
		}
		if x.Asset != nil {
			encoded, err := options.Marshal(x.Asset)
			if err != nil {
				return protoiface.MarshalOutput{
					NoUnkeyedLiterals: input.NoUnkeyedLiterals,
					Buf:               input.Buf,
				}, err
			}
			i -= len(encoded)
			copy(dAtA[i:], encoded)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
			i--
			dAtA[i] = 0xa
		}
		if input.Buf != nil {
			input.Buf = append(input.Buf, dAtA...)
		} else {
			input.Buf = dAtA
		}
		return protoiface.MarshalOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Buf:               input.Buf,
		}, nil
	}
	unmarshal := func(input protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
		x := input.Message.Interface().(*MsgDonate)
		if x == nil {
			return protoiface.UnmarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Flags:             input.Flags,
			}, nil
		}
		options := runtime.UnmarshalInputToOptions(input)
		_ = options
		dAtA := input.Buf
		l := len(dAtA)
		iNdEx := 0
		for iNdEx < l {
			preIndex := iNdEx
			var wire uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
				}
				if iNdEx >= l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: MsgDonate: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: MsgDonate: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Asset", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if x.Asset == nil {
					x.Asset = &common.Asset{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.Asset); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field AssetAmount", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.AssetAmount = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 3:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field RuneAmount", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.RuneAmount = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 4:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Tx", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if x.Tx == nil {
					x.Tx = &common.Tx{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.Tx); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			case 5:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
				}
				var byteLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					byteLen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if byteLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + byteLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.Signer = append(x.Signer[:0], dAtA[iNdEx:postIndex]...)
				if x.Signer == nil {
					x.Signer = []byte{}
				}
				iNdEx = postIndex
			default:
				iNdEx = preIndex
				skippy, err := runtime.Skip(dAtA[iNdEx:])
				if err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				if (skippy < 0) || (iNdEx+skippy) < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if (iNdEx + skippy) > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if !options.DiscardUnknown {
					x.unknownFields = append(x.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
				}
				iNdEx += skippy
			}
		}

		if iNdEx > l {
			return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
		}
		return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, nil
	}
	return &protoiface.Methods{
		NoUnkeyedLiterals: struct{}{},
		Flags:             protoiface.SupportMarshalDeterministic | protoiface.SupportUnmarshalDiscardUnknown,
		Size:              size,
		Marshal:           marshal,
		Unmarshal:         unmarshal,
		Merge:             nil,
		CheckInitialized:  nil,
	}
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.0
// 	protoc        (unknown)
// source: types/msg_donate.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MsgDonate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Asset       *common.Asset `protobuf:"bytes,1,opt,name=asset,proto3" json:"asset,omitempty"`
	AssetAmount string        `protobuf:"bytes,2,opt,name=asset_amount,json=assetAmount,proto3" json:"asset_amount,omitempty"`
	RuneAmount  string        `protobuf:"bytes,3,opt,name=rune_amount,json=runeAmount,proto3" json:"rune_amount,omitempty"`
	Tx          *common.Tx    `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
	Signer      []byte        `protobuf:"bytes,5,opt,name=signer,proto3" json:"signer,omitempty"`
}

func (x *MsgDonate) Reset() {
	*x = MsgDonate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_msg_donate_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MsgDonate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MsgDonate) ProtoMessage() {}

// Deprecated: Use MsgDonate.ProtoReflect.Descriptor instead.
func (*MsgDonate) Descriptor() ([]byte, []int) {
	return file_types_msg_donate_proto_rawDescGZIP(), []int{0}
}

func (x *MsgDonate) GetAsset() *common.Asset {
	if x != nil {
		return x.Asset
	}
	return nil
}

func (x *MsgDonate) GetAssetAmount() string {
	if x != nil {
		return x.AssetAmount
	}
	return ""
}

func (x *MsgDonate) GetRuneAmount() string {
	if x != nil {
		return x.RuneAmount
	}
	return ""
}

func (x *MsgDonate) GetTx() *common.Tx {
	if x != nil {
		return x.Tx
	}
	return nil
}

func (x *MsgDonate) GetSigner() []byte {
	if x != nil {
		return x.Signer
	}
	return nil
}

var File_types_msg_donate_proto protoreflect.FileDescriptor

var file_types_msg_donate_proto_rawDesc = []byte{
	0x0a, 0x16, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x6d, 0x73, 0x67, 0x5f, 0x64, 0x6f, 0x6e, 0x61,
	0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a,
	0x13, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x14, 0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd8, 0x02, 0x0a, 0x09, 0x4d,
	0x73, 0x67, 0x44, 0x6f, 0x6e, 0x61, 0x74, 0x65, 0x12, 0x5a, 0x0a, 0x05, 0x61, 0x73, 0x73, 0x65,
	0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x42, 0x35, 0xc8, 0xde, 0x1f, 0x00, 0xda, 0xde, 0x1f, 0x2d,
	0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x68, 0x6f, 0x72, 0x63,
	0x68, 0x61, 0x69, 0x6e, 0x2f, 0x74, 0x68, 0x6f, 0x72, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x76, 0x33,
	0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x52, 0x05, 0x61,
	0x73, 0x73, 0x65, 0x74, 0x12, 0x41, 0x0a, 0x0c, 0x61, 0x73, 0x73, 0x65, 0x74, 0x5f, 0x61, 0x6d,
	0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x1e, 0xc8, 0xde, 0x1f, 0x00,
	0xda, 0xde, 0x1f, 0x16, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f,
	0x2f, 0x6d, 0x61, 0x74, 0x68, 0x2e, 0x55, 0x69, 0x6e, 0x74, 0x52, 0x0b, 0x61, 0x73, 0x73, 0x65,
	0x74, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x3f, 0x0a, 0x0b, 0x72, 0x75, 0x6e, 0x65, 0x5f,
	0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x42, 0x1e, 0xc8, 0xde,
	0x1f, 0x00, 0xda, 0xde, 0x1f, 0x16, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e,
	0x69, 0x6f, 0x2f, 0x6d, 0x61, 0x74, 0x68, 0x2e, 0x55, 0x69, 0x6e, 0x74, 0x52, 0x0a, 0x72, 0x75,
	0x6e, 0x65, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x20, 0x0a, 0x02, 0x74, 0x78, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x54, 0x78,
	0x42, 0x04, 0xc8, 0xde, 0x1f, 0x00, 0x52, 0x02, 0x74, 0x78, 0x12, 0x49, 0x0a, 0x06, 0x73, 0x69,
	0x67, 0x6e, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x42, 0x31, 0xfa, 0xde, 0x1f, 0x2d,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f,
	0x73, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2d, 0x73, 0x64, 0x6b, 0x2f, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x41, 0x63, 0x63, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x52, 0x06, 0x73,
	0x69, 0x67, 0x6e, 0x65, 0x72, 0x42, 0x86, 0x01, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x2e, 0x74, 0x79,
	0x70, 0x65, 0x73, 0x42, 0x0e, 0x4d, 0x73, 0x67, 0x44, 0x6f, 0x6e, 0x61, 0x74, 0x65, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x73, 0x77, 0x69, 0x74, 0x63, 0x68, 0x6c, 0x79, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6c, 0x2f, 0x73, 0x77, 0x69, 0x74, 0x63, 0x68, 0x6c, 0x79, 0x6e, 0x6f, 0x64, 0x65, 0x2f,
	0x76, 0x31, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0xa2, 0x02, 0x03, 0x54,
	0x58, 0x58, 0xaa, 0x02, 0x05, 0x54, 0x79, 0x70, 0x65, 0x73, 0xca, 0x02, 0x05, 0x54, 0x79, 0x70,
	0x65, 0x73, 0xe2, 0x02, 0x11, 0x54, 0x79, 0x70, 0x65, 0x73, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x05, 0x54, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_types_msg_donate_proto_rawDescOnce sync.Once
	file_types_msg_donate_proto_rawDescData = file_types_msg_donate_proto_rawDesc
)

func file_types_msg_donate_proto_rawDescGZIP() []byte {
	file_types_msg_donate_proto_rawDescOnce.Do(func() {
		file_types_msg_donate_proto_rawDescData = protoimpl.X.CompressGZIP(file_types_msg_donate_proto_rawDescData)
	})
	return file_types_msg_donate_proto_rawDescData
}

var file_types_msg_donate_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_types_msg_donate_proto_goTypes = []interface{}{
	(*MsgDonate)(nil),    // 0: types.MsgDonate
	(*common.Asset)(nil), // 1: common.Asset
	(*common.Tx)(nil),    // 2: common.Tx
}
var file_types_msg_donate_proto_depIdxs = []int32{
	1, // 0: types.MsgDonate.asset:type_name -> common.Asset
	2, // 1: types.MsgDonate.tx:type_name -> common.Tx
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_types_msg_donate_proto_init() }
func file_types_msg_donate_proto_init() {
	if File_types_msg_donate_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_types_msg_donate_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MsgDonate); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_types_msg_donate_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_types_msg_donate_proto_goTypes,
		DependencyIndexes: file_types_msg_donate_proto_depIdxs,
		MessageInfos:      file_types_msg_donate_proto_msgTypes,
	}.Build()
	File_types_msg_donate_proto = out.File
	file_types_msg_donate_proto_rawDesc = nil
	file_types_msg_donate_proto_goTypes = nil
	file_types_msg_donate_proto_depIdxs = nil
}
