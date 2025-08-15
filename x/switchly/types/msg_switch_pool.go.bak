package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

var (
	_ sdk.Msg              = &MsgSwitchPoolDeposit{}
	_ sdk.HasValidateBasic = &MsgSwitchPoolDeposit{}
	_ sdk.LegacyMsg        = &MsgSwitchPoolDeposit{}

	_ sdk.Msg              = &MsgSwitchPoolWithdraw{}
	_ sdk.HasValidateBasic = &MsgSwitchPoolWithdraw{}
	_ sdk.LegacyMsg        = &MsgSwitchPoolWithdraw{}
)

// NewMsgSwitchPoolDeposit create new MsgSwitchPoolDeposit message
func NewMsgSwitchPoolDeposit(signer cosmos.AccAddress, tx common.Tx) *MsgSwitchPoolDeposit {
	return &MsgSwitchPoolDeposit{
		Signer: signer,
		Tx:     tx,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgSwitchPoolDeposit) ValidateBasic() error {
	if !m.Tx.Chain.Equals(common.SWITCHLYChain) {
		return cosmos.ErrUnauthorized("chain must be SWITCHLYChain")
	}
	if len(m.Tx.Coins) != 1 {
		return cosmos.ErrInvalidCoins("coins must be length 1 (SWITCH)")
	}
	if !m.Tx.Coins[0].Asset.Chain.IsSWITCHLYChain() {
		return cosmos.ErrInvalidCoins("coin chain must be SWITCHLYChain")
	}
	if !m.Tx.Coins[0].IsSwitch() {
		return cosmos.ErrInvalidCoins("coin must be SWITCH")
	}
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress("signer must not be empty")
	}
	if m.Tx.Coins[0].Amount.IsZero() {
		return cosmos.ErrUnknownRequest("coins amount must not be zero")
	}
	return nil
}

// GetSigners defines whose signature is required
func (m *MsgSwitchPoolDeposit) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}

// NewMsgSwitchPoolWithdraw create new MsgSwitchPoolWithdraw message
func NewMsgSwitchPoolWithdraw(signer cosmos.AccAddress, tx common.Tx, basisPoints cosmos.Uint, affAddr common.Address, affBps cosmos.Uint) *MsgSwitchPoolWithdraw {
	return &MsgSwitchPoolWithdraw{
		Signer:               signer,
		Tx:                   tx,
		BasisPoints:          basisPoints,
		AffiliateAddress:     affAddr,
		AffiliateBasisPoints: affBps,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgSwitchPoolWithdraw) ValidateBasic() error {
	if !m.Tx.Coins.IsEmpty() {
		return cosmos.ErrInvalidCoins("coins must be empty (zero amount)")
	}
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress("signer must not be empty")
	}
	if m.BasisPoints.IsZero() || m.BasisPoints.GT(cosmos.NewUint(constants.MaxBasisPts)) {
		return cosmos.ErrUnknownRequest("invalid basis points")
	}
	if m.AffiliateBasisPoints.GT(cosmos.NewUint(constants.MaxBasisPts)) {
		return cosmos.ErrUnknownRequest("invalid affiliate basis points")
	}
	if !m.AffiliateBasisPoints.IsZero() && m.AffiliateAddress.IsEmpty() {
		return cosmos.ErrInvalidAddress("affiliate basis points with no affiliate address")
	}

	return nil
}

// GetSigners defines whose signature is required
func (m *MsgSwitchPoolWithdraw) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
