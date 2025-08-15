package conversion

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/switchlyprotocol/switchlynode/v3/cmd"
)

func SetupBech32Prefix() {
	config := sdk.GetConfig()
	// switchly will import go-tss as a library , thus this is not needed, we copy the prefix here to avoid go-tss to import switchly
	config.SetBech32PrefixForAccount(cmd.Bech32PrefixAccAddr, cmd.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(cmd.Bech32PrefixValAddr, cmd.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(cmd.Bech32PrefixConsAddr, cmd.Bech32PrefixConsPub)
}
