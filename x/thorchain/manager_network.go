package thorchain

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/x/thorchain/keeper"
)

// const values used to emit events
const (
	EventTypeActiveVault   = "ActiveVault"
	EventTypeInactiveVault = "InactiveVault"
)

func getTotalActiveNodeWithBond(ctx cosmos.Context, k keeper.Keeper) (int64, error) {
	nas, err := k.ListActiveValidators(ctx)
	if err != nil {
		return 0, fmt.Errorf("fail to get active node accounts: %w", err)
	}
	var total int64
	for _, item := range nas {
		if !item.Bond.IsZero() {
			total++
		}
	}
	return total, nil
}
