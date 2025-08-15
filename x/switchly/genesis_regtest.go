//go:build regtest
// +build regtest

package switchly

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/keeper"
)

func InitGenesis(ctx cosmos.Context, keeper keeper.Keeper, data GenesisState) []abci.ValidatorUpdate {
	validators := initGenesis(ctx, keeper, data)
	return validators[:1]
}
