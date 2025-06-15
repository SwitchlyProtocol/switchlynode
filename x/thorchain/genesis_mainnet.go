//go:build !regtest
// +build !regtest

package thorchain

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

func InitGenesis(ctx cosmos.Context, keeper keeper.Keeper, data GenesisState) []abci.ValidatorUpdate {
	return initGenesis(ctx, keeper, data)
}
