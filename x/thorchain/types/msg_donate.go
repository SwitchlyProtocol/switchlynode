package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

var (
	_ sdk.Msg              = &MsgDonate{}
	_ sdk.HasValidateBasic = &MsgDonate{}
	_ sdk.LegacyMsg        = &MsgDonate{}
)

// NewMsgDonate is a constructor function for MsgDonate
func NewMsgDonate(tx common.Tx, asset common.Asset, r, amount cosmos.Uint, signer cosmos.AccAddress) *MsgDonate {
	return &MsgDonate{
		Asset:       asset,
		AssetAmount: amount,
		RuneAmount:  r,
		Tx:          tx,
		Signer:      signer,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgDonate) ValidateBasic() error {
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress(m.Signer.String())
	}
	if m.Asset.IsEmpty() {
		return cosmos.ErrUnknownRequest("donate asset cannot be empty")
	}
	if m.Asset.IsSwitch() {
		return cosmos.ErrUnknownRequest("asset cannot be switch")
	}
	if m.RuneAmount.IsZero() && m.AssetAmount.IsZero() {
		return errors.New("rune and asset amount cannot be zero")
	}
	if err := m.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	return nil
}

// GetSigners defines whose signature is required
func (m *MsgDonate) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
