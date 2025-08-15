package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

func NewSwitchPool() SwitchPool {
	return SwitchPool{
		ReserveUnits:    cosmos.ZeroUint(),
		PoolUnits:       cosmos.ZeroUint(),
		SwitchDeposited: cosmos.ZeroUint(),
		SwitchWithdrawn: cosmos.ZeroUint(),
	}
}

func (rp SwitchPool) CurrentDeposit() cosmos.Int {
	deposited := cosmos.NewIntFromBigInt(rp.SwitchDeposited.BigInt())
	withdrawn := cosmos.NewIntFromBigInt(rp.SwitchWithdrawn.BigInt())
	return deposited.Sub(withdrawn)
}

func (rp SwitchPool) TotalUnits() cosmos.Uint {
	return rp.ReserveUnits.Add(rp.PoolUnits)
}
