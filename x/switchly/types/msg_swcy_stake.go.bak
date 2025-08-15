package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

var (
	_ sdk.Msg              = &MsgSWCYStake{}
	_ sdk.HasValidateBasic = &MsgSWCYStake{}
	_ sdk.LegacyMsg        = &MsgSWCYStake{}
)

// NewMsgSWCYStake create new MsgSWCYStake message
func NewMsgSWCYStake(tx common.Tx, signer sdk.AccAddress) *MsgSWCYStake {
	return &MsgSWCYStake{
		Tx:     tx,
		Signer: signer,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgSWCYStake) ValidateBasic() error {
	if !m.Tx.Chain.Equals(common.SWITCHLYChain) {
		return cosmos.ErrUnauthorized("chain must be SWITCHLYChain")
	}
	if len(m.Tx.Coins) != 1 {
		return cosmos.ErrInvalidCoins("coins must be length 1 (SWCY)")
	}
	if !m.Tx.Coins[0].IsSWCY() {
		return cosmos.ErrInvalidCoins("coin must be SWCY")
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
func (m *MsgSWCYStake) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
