package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

// MaxWithdrawBasisPoints basis points for withdrawals
const MaxWithdrawBasisPoints = 10_000

var (
	_ sdk.Msg              = &MsgWithdrawLiquidity{}
	_ sdk.HasValidateBasic = &MsgWithdrawLiquidity{}
	_ sdk.LegacyMsg        = &MsgWithdrawLiquidity{}
)

// NewMsgWithdrawLiquidity is a constructor function for MsgWithdrawLiquidity
func NewMsgWithdrawLiquidity(tx common.Tx, withdrawAddress common.Address, withdrawBasisPoints cosmos.Uint, asset, withdrawalAsset common.Asset, signer cosmos.AccAddress) *MsgWithdrawLiquidity {
	return &MsgWithdrawLiquidity{
		Tx:              tx,
		WithdrawAddress: withdrawAddress,
		BasisPoints:     withdrawBasisPoints,
		Asset:           asset,
		WithdrawalAsset: withdrawalAsset,
		Signer:          signer,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgWithdrawLiquidity) ValidateBasic() error {
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress(m.Signer.String())
	}
	// here we can't call m.Tx.Valid , because we allow user to send withdraw request without any coins in it
	// m.Tx.Valid will reject this kind request , which result withdraw to fail
	if m.Tx.ID.IsEmpty() {
		return cosmos.ErrInvalidAddress("tx id cannot be empty")
	}
	if m.Asset.IsEmpty() {
		return cosmos.ErrUnknownRequest("pool asset cannot be empty")
	}
	if m.Asset.IsSwitch() {
		return cosmos.ErrUnknownRequest("asset cannot be switch")
	}
	if m.WithdrawAddress.IsEmpty() {
		return cosmos.ErrUnknownRequest("address cannot be empty")
	}
	if m.BasisPoints.IsZero() {
		return cosmos.ErrUnknownRequest("basis points can't be zero")
	}
	if m.BasisPoints.GT(cosmos.NewUint(MaxWithdrawBasisPoints)) {
		return cosmos.ErrUnknownRequest("basis points is larger than maximum withdraw basis points")
	}
	if !m.WithdrawalAsset.IsEmpty() && !m.WithdrawalAsset.IsSwitch() && !m.WithdrawalAsset.Equals(m.Asset) {
		return cosmos.ErrUnknownRequest("withdrawal asset must be empty, switch, or pool asset")
	}
	return nil
}

// GetSigners defines whose signature is required
func (m *MsgWithdrawLiquidity) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
