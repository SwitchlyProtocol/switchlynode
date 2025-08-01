package keeperv1

import (
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

// AddPoolFeeToReserve add fee to reserve, the fee is always in RUNE
func (k KVStore) AddPoolFeeToReserve(ctx cosmos.Context, fee cosmos.Uint) error {
	coin := common.NewCoin(common.SwitchNative, fee)
	sdkErr := k.SendFromModuleToModule(ctx, AsgardName, ReserveName, common.NewCoins(coin))
	if sdkErr != nil {
		return dbError(ctx, "fail to send pool fee to reserve", sdkErr)
	}
	return nil
}

// AddBondFeeToReserve add fee to reserve, the fee is always in RUNE
func (k KVStore) AddBondFeeToReserve(ctx cosmos.Context, fee cosmos.Uint) error {
	coin := common.NewCoin(common.SwitchNative, fee)
	sdkErr := k.SendFromModuleToModule(ctx, BondName, ReserveName, common.NewCoins(coin))
	if sdkErr != nil {
		return dbError(ctx, "fail to send bond fee to reserve", sdkErr)
	}
	return nil
}
