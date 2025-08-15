package types

import (
	"errors"

	"cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
)

func NewSWCYStaker(address common.Address, amount math.Uint) SWCYStaker {
	return SWCYStaker{
		Address: address,
		Amount:  amount,
	}
}

func (t *SWCYStaker) Valid() error {
	if t.Address.IsEmpty() {
		return errors.New("address is empty")
	}
	if t.Amount.IsZero() {
		return errors.New("staking amount is zero")
	}
	return nil
}

func (t *SWCYStaker) IsEmpty() bool {
	return t.Address.IsEmpty() && t.Amount.IsZero()
}
