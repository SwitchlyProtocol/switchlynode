package types

import (
	"errors"

	"cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
)

func NewSWCYClaimer(l1Address common.Address, asset common.Asset, amount math.Uint) SWCYClaimer {
	return SWCYClaimer{
		L1Address: l1Address,
		Asset:     asset,
		Amount:    amount,
	}
}

func (t *SWCYClaimer) Valid() error {
	if t.L1Address.IsEmpty() {
		return errors.New("L1 address is empty")
	}
	if t.Amount.IsZero() {
		return errors.New("claim amount is zero")
	}
	if t.Asset.IsEmpty() {
		return errors.New("asset is empty")
	}
	return nil
}

func (t *SWCYClaimer) IsEmpty() bool {
	return t.L1Address.IsEmpty() && t.Amount.IsZero() && t.Asset.IsEmpty()
}
