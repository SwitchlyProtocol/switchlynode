package params

import (
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/x/tx/signing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	switchlytypes "github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
func MakeEncodingConfig() EncodingConfig {
	amino := codec.NewLegacyAmino()

	interfaceRegistrySigningOptions := signing.Options{
		AddressCodec: address.Bech32Codec{
			Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
		},
		ValidatorAddressCodec: address.Bech32Codec{
			Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
		},
	}
	switchlytypes.DefineCustomGetSigners(&interfaceRegistrySigningOptions)
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: interfaceRegistrySigningOptions,
	})
	if err != nil {
		panic(err)
	}

	marshaler := codec.NewProtoCodec(interfaceRegistry)

	txSigningOptions, err := tx.NewDefaultSigningOptions()
	if err != nil {
		panic(err)
	}
	switchlytypes.DefineCustomGetSigners(txSigningOptions)
	txCfg, err := TxConfig(marshaler, nil)
	if err != nil {
		panic(err)
	}
	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             marshaler,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}
