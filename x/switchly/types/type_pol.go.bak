package types

import (
	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

// NewProtocolOwnedLiquidity create a new instance ProtocolOwnedLiquidity it is empty though
func NewProtocolOwnedLiquidity() ProtocolOwnedLiquidity {
	return ProtocolOwnedLiquidity{
		SwitchDeposited: cosmos.ZeroUint(),
		SwitchWithdrawn: cosmos.ZeroUint(),
	}
}

func (pol ProtocolOwnedLiquidity) CurrentDeposit() cosmos.Int {
	deposited := cosmos.NewIntFromBigInt(pol.SwitchDeposited.BigInt())
	withdrawn := cosmos.NewIntFromBigInt(pol.SwitchWithdrawn.BigInt())
	return deposited.Sub(withdrawn)
}

// PnL - Profit and Loss
func (pol ProtocolOwnedLiquidity) PnL(value cosmos.Uint) cosmos.Int {
	deposited := cosmos.NewIntFromBigInt(pol.SwitchDeposited.BigInt())
	withdrawn := cosmos.NewIntFromBigInt(pol.SwitchWithdrawn.BigInt())
	v := cosmos.NewIntFromBigInt(value.BigInt())
	return withdrawn.Sub(deposited).Add(v)
}
